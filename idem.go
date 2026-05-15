package idem

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh"

)

type HostConfig struct {
	Host string
	Port int
	User string
	Sudo bool
	Password string
	DryRun bool
}

func (h *HostConfig) Dial(network string) (*ssh.Client, error) {
	sshConfig := &ssh.ClientConfig {
		User: h.User,
		Auth: []ssh.AuthMethod {
			ssh.Password(h.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO: dont use this
	}
	port := h.Port
	if port == 0 {
		port = 22
	}
	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", h.Host, port), sshConfig)
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

func runBin(client *ssh.Client, stdin io.Reader, path string, sudo bool) ([]byte, error, bool) {
	sent := false
	file, err := os.Open(path)
	if err != nil {
		return nil, err, sent
	}
	defer file.Close()
	base := filepath.Base(path)
	dstPath := filepath.Join("/tmp", base)
	err = poorManScp(client, file, dstPath)
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
