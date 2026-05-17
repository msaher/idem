package idem

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"os/exec"
	"os"

	"embed"

	"golang.org/x/crypto/ssh"
)

//go:embed compile/bin/*
var binaries embed.FS

type Log struct {
	Name string
	Changed bool
	Err error
	Result any
}

var NoOp = errors.New("Host Context has its error set. No action was taken")

type HostCtx struct {
	Host      string
	Port      int
	Sudo      bool
	SshConfig *ssh.ClientConfig
	Client *ssh.Client
	Err error
	Logs []*Log
	local bool
}

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

// TODO: h.Done(). Closes connection. Removes binaries
func (h *HostCtx) Close() error {
	if h.Client != nil {
		return h.Client.Close()
	}
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
    return fmt.Sprintf(
        "ssh command failed (exit %d): \noutput:\n%s",
        e.ExitStatus(),
        e.CombinedOutput,
    )
}

func runBin(client *ssh.Client, stdin io.Reader, binName string, sudo bool) ([]byte, error, bool) {
	sent := false
	bin, err := binaries.ReadFile(filepath.Join("compile", "bin", binName))
	if err != nil {
		return nil, err, sent
	}
	dstPath := filepath.Join("/tmp", binName)
	err = poorManScp(client, bytes.NewReader(bin), dstPath)
	if err != nil {
		return nil, err, sent
	}
	sent = true

	ses, err := client.NewSession()
	if err != nil {
		return nil, err, sent
	}
	defer ses.Close()

	// NOTE: better not put dumb dstPath
	err = ses.Run(fmt.Sprintf("chmod +x %s", dstPath))
	if err != nil {
		return nil, err, sent
	}

	// now we have the binary. Lets run it
	binSes, err := client.NewSession()
	if err != nil {
		return nil, err, sent
	}
	defer binSes.Close()

	binSes.Stdin = stdin
	cmd := dstPath
	if sudo {
		cmd = "sudo " + dstPath
	}
	out, err := binSes.CombinedOutput(cmd)

	if err != nil {
		return out, err, sent
	}

	return out, err, sent
}

func run(h *HostCtx, req any, bin string, res any) error {
	if h.Err != nil {
		return NoOp
	}
	var err error
	defer func() {
		h.Err = err
	}()

	jsn, err := json.MarshalIndent(req, "", "\t")
	if err != nil {
		panic(err)
	}

	var out []byte
	if h.local {
		var b []byte
		b, err = binaries.ReadFile(filepath.Join("compile", "bin", bin))
		if err != nil {
			return nil
		}
		pth := filepath.Join("/tmp", bin)
		defer os.Remove(pth)
		file, err := os.OpenFile(pth, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
		if err != nil {
			return err
		}
		_, err = file.Write(b)
		file.Close()
		if err != nil {
			return err
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
		var client *ssh.Client
		client, err = h.dial()
		if err != nil {
			return err
		}
		// TODO: this is a bit silly. runBin should be responsible for deleting
		var sent bool
		out, err, sent = runBin(client, bytes.NewReader(jsn), bin, h.Sudo)

		// remove binary if we sent it successfully
		defer func() {
			if sent {
				ses, err := client.NewSession()
				// give up. can't do it
				if err != nil {
					return
				}
				ses.Run(fmt.Sprintf("rm %s", bin))
				ses.Close()
			}
		}()

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

// Since structured errors are propogated only through the remote binary the
// result types are the only means of communication and thus must include
// everything including what went wrong. Since go is a language that expects two
// errors values we return a second error value thats identical to the result type
// but since it will be consuing of Results.Err() was a thing and results were
// compatible with error interface we create a wrapper type... that's essentially
// an alias with an Err method that pretty prints.
