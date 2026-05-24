package idem

import (
	"bytes"
	"encoding/json"
	"errors"
	"embed"
	"fmt"
	"io"
	"path/filepath"
	"os/exec"
	"os"
	"slices"
	"sync"
	"strings"

	"golang.org/x/crypto/ssh"
)

//go:embed compile/bin/*
var binaries embed.FS

// Log represents an attempted change to a host.
type Log struct {
	Name string `json:"name"`
	Changed bool `json:"changed"`

	// Error message (extracted from the Result)
	Err error `json:"err,omitempty"`

	// Pointer to the corresponding result struct
	Result any `json:"result"`
}

var NoOp = errors.New("Host Context has its error set. No action was taken")

// HostCtx holds the connection state and execution context for a remote or local host.
type HostCtx struct {
	Host      string
	Port      int
	Sudo      bool
	SshConfig *ssh.ClientConfig
	Client *ssh.Client

	// Last encountered error. If this is set. Calls to Run(h) will return a NoOp error
	Err error

	Logs []*Log
	local bool
	cachedBinaries []string
}

// A Special HostCtx that runs locally. It does not ssh at all, but instead
// pushes the binaries in /tmp and executes them
var Local = &HostCtx{local: true}

func (h *HostCtx) dial() (*ssh.Client, error) {
	if h.Client != nil {
		return h.Client, nil
	}
	port := h.Port
	if port == 0 {
		port = 22
	}
	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", h.Host, port), h.SshConfig)
	h.Client = client
	return client, err
}

// Closes ssh connection (if any), and deletes the cached binaries
func (h *HostCtx) Done() error {
	if h.local {
		for _, b := range h.cachedBinaries {
			os.Remove(filepath.Join("/tmp", b))
		}
		h.cachedBinaries = nil
		return nil
	}

	client, err := h.dial()
	if err != nil {
		return err
	}
	defer client.Close()
	ses, err := client.NewSession()
	if err != nil {
		return err
	}
	defer ses.Close()

	err = ses.Run("rm /tmp/idem_*")
	if err != nil {
		return err
	}
	h.cachedBinaries = nil
	return nil
}

func poorManScp(client *ssh.Client, r io.Reader, dstPath string) error {
	ses, err := client.NewSession()
	if err != nil {
		return err
	}
	defer ses.Close()

	w, err := ses.StdinPipe()
	if err != nil {
		return err
	}

	// io.Copy will block if cat is not running
	go func() {
		defer w.Close()
		io.Copy(w, r)
	}()

	// NOTE: dumb paths will cause bugs or injections, but its fine since we
	// pass the paths ourselves not from user input
	err = ses.Run(fmt.Sprintf("cat > %s", dstPath))
	return err
}

type ExitErr struct {
	SshErr *ssh.ExitError
	ExecErr *exec.ExitError
    CombinedOutput string
}

func (e *ExitErr) ExitStatus() int {
	if e.SshErr != nil {
		return e.SshErr.ExitStatus()
	}
	return e.ExecErr.ExitCode()
}

func (e *ExitErr) Error() string {
	var prefix string
	if e.SshErr != nil {
		prefix = "ssh "
	}
    return fmt.Sprintf(
        "%scommand failed (exit %d): \noutput:\n%s",
		prefix,
        e.ExitStatus(),
        e.CombinedOutput,
    )
}

func runBin(h *HostCtx, stdin io.Reader, binName string) ([]byte, error) {
	client, err := h.dial()
	if err != nil {
		return nil, err
	}

	dstPath := filepath.Join("/tmp", binName)
	if !slices.Contains(h.cachedBinaries, binName) { // haven't pushed this before
		bin, err := binaries.ReadFile(filepath.Join("compile", "bin", binName))
		if err != nil {
			return nil, err
		}
		err = poorManScp(client, bytes.NewReader(bin), dstPath)
		if err != nil {
			return nil, err
		}

		ses, err := client.NewSession()
		if err != nil {
			return nil, err
		}
		defer ses.Close()

		// NOTE: better not put dumb dstPath
		err = ses.Run(fmt.Sprintf("chmod +x %s", dstPath))
		if err != nil {
			return nil, err
		}

		h.cachedBinaries = append(h.cachedBinaries, binName)
	}

	// now we have the binary. Lets run it
	binSes, err := client.NewSession()
	if err != nil {
		return nil, err
	}
	defer binSes.Close()

	binSes.Stdin = stdin
	cmd := dstPath
	if h.Sudo {
		cmd = "sudo " + dstPath
	}
	out, err := binSes.CombinedOutput(cmd)

	if err != nil {
		return out, err
	}

	return out, err
}

func run(h *HostCtx, req any, name string, bin string, res any, changed *bool) error {
	if h.Err != nil {
		return NoOp
	}
	var err error
	defer func() {
		h.Err = err
		l := &Log{Name: name, Err: err}
		if err == nil {
			l.Result = res
			l.Changed = *changed
		}
		h.Logs = append(h.Logs, l)
	}()

	jsn, err := json.MarshalIndent(req, "", "\t")
	if err != nil {
		panic(err)
	}

	var out []byte
	if h.local {
		pth := filepath.Join("/tmp", bin)
		if !slices.Contains(h.cachedBinaries, bin) {
			var b []byte
			b, err = binaries.ReadFile(filepath.Join("compile", "bin", bin))
			if err != nil {
				return err
			}
			file, err := os.OpenFile(pth, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
			if err != nil {
				return err
			}
			_, err = file.Write(b)
			if err != nil {
				return err
			}
			file.Close()
			h.cachedBinaries = append(h.cachedBinaries, bin)
		}


		cmd := exec.Command(pth)
		cmd.Stdin = bytes.NewReader(jsn)
		out, err = cmd.CombinedOutput()

		if execErr, ok := errors.AsType[*exec.ExitError](err); ok {
			return &ExitErr{ExecErr: execErr, CombinedOutput: string(out)}
		} else if err != nil {
			return err
		}
	} else { // ssh
		out, err = runBin(h, bytes.NewReader(jsn), bin)
		// check if it ran successfully
		if sshExitErr, ok := errors.AsType[*ssh.ExitError](err); ok {
			return &ExitErr{SshErr: sshExitErr, CombinedOutput: string(out)}
		} else if err != nil {
			return err
		}
	}

	// read output
	r := bytes.NewReader(out)
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	err = dec.Decode(res)
	if err != nil {
		// We can't decode? binary shows unexpected output. This is a bug
		panic(fmt.Errorf("bin: %s. Output:\n%s", bin, string(out)))
	}

	return nil
}

// Summary returns a human-readable summary of all changes made during this session
func (h *HostCtx) Summary() string {
    var sb strings.Builder
    host := h.Host
    if h.local {
        host = "local"
    }
    fmt.Fprintf(&sb, "=== %s ===\n", host)
    for _, l := range h.Logs {
        status := "ok"
        if l.Changed {
            status = "changed"
        }
        if l.Err != nil {
            status = "failed"
        }
        fmt.Fprintf(&sb, "[%s] %s\n", status, l.Name)
        if l.Err != nil {
            fmt.Fprintf(&sb, "  error: %v\n", l.Err)
        }
    }
    return sb.String()
}

// Since structured errors are propogated only through the remote binary the
// result types are the only means of communication and thus must include
// everything including what went wrong. Since go is a language that expects two
// errors values we return a second error value thats identical to the result type
// but since it will be consuing of Results.Err() was a thing and results were
// compatible with error interface we create a wrapper type... that's essentially
// an alias with an Err method that pretty prints.


// ssh wrappers

// NewHost creates a new HostCtx for the given address and SSH config.
func NewHost(addr string, sshConfig *ssh.ClientConfig) *HostCtx {
	return &HostCtx {
		Host: addr,
		SshConfig: sshConfig,
	}
}

// should not be used once the hostctx is actually used. We could seperate
// these two into a builder step but it increases the api surface just to prevent
// an unlikely mistake.

// WithPort sets the SSH port. Defaults to 22.
func (h *HostCtx) WithPort(p int) *HostCtx {
	h.Port = p
	return h
}

// WithSudo runs all commands with sudo.
func (h *HostCtx) WithSudo(s bool) *HostCtx {
	h.Sudo = s
	return h
}

// ForEach runs f in parallel. This is useful when you need to provision
// multiple servers wihtout having to wait for each one to
// complete
func ForEach(hs []*HostCtx, f func(h *HostCtx)) {
	var wg sync.WaitGroup
	for _, h := range hs {
		wg.Go(func() {
			f(h)
		})
	}

	wg.Wait()
}
