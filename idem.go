package idem

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"

	"embed"

	"golang.org/x/crypto/ssh"
)

//go:embed compile/bin/*
var binaries embed.FS

type HostConfig struct {
	Host      string
	Port      int
	Sudo      bool
	SshConfig *ssh.ClientConfig
}

func (h *HostConfig) Dial(network string) (*ssh.Client, error) {
	port := h.Port
	if port == 0 {
		port = 22
	}
	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", h.Host, port), h.SshConfig)
	return client, err
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
    *ssh.ExitError
    CombinedOutput string
}

func (e *ExitErr) Error() string {
    return fmt.Sprintf(
        "ssh command failed (exit %d): %v\noutput:\n%s",
        e.ExitError.ExitStatus(),
        e.ExitError,
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

func run(h *HostConfig, req any, bin string, res any) error {
	jsn, err := json.MarshalIndent(req, "", "\t")
	if err != nil {
		panic(err)
	}

	// TODO: might want to reuse clients
	client, err := h.Dial("tcp")
	if err != nil {
		return err
	}
	defer client.Close()
	out, err, sent := runBin(client, bytes.NewReader(jsn), bin, h.Sudo)

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
		err = &ExitErr{sshExitErr, string(out)}
		return err
	} else if err != nil {
		return err
	}

	// read output
	r := bytes.NewReader(out)
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	err = dec.Decode(res)
	if err != nil {
		// We can't decode? binary shows unexpected output. This is a bug
		panic(map[string]any{
			"bin": bin,
			"output": string(out),
		})
	}

	return nil
}
