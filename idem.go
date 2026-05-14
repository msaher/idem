package idem

import (
	"fmt"
	"path/filepath"
	"strings"
	"io"
	"os"
	"os/exec"

	"golang.org/x/crypto/ssh"
)

type HostConfig struct {
	Host string
	Port int
	User string
	Sudo bool
	Password string
}

func (h *HostConfig) Dial(network string) (*ssh.Client, error) {
	// get a client
	// this should be its own function later
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



type UserState struct {
	Name string
	Exists bool
	Groups []string
}


type UserConfig struct {
	Name string
	groups []string
	append bool
}

func User(name string) *UserConfig {
	return &UserConfig {
		Name: name,
	}
}

func (cfg *UserConfig) Append(v bool) *UserConfig {
	cfg.append = true
	return cfg
}

func (cfg *UserConfig) Groups(groups ...string) *UserConfig {
	cfg.groups = groups
	return cfg
}


func runBin(h *HostConfig, stdin io.Reader, path string) ([]byte, error) {
	client, err := h.Dial("tcp")
	if err != nil {
		return nil, err
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	base := filepath.Base(path)
	dstPath := filepath.Join("/tmp", base)
	err = poorManScp(client, file, dstPath)
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

func (u *UserConfig) Run(h *HostConfig) ([]byte, error) {
	// TODO: embed these source files and precompile them instead of compiling everytime
	c := exec.Cmd{
		Path: "/usr/bin/go",
		Args: []string{"go", "build", "-o", "/tmp/idem_user", "./remote/user/"},
		Env: append(os.Environ(),
			"CGO_ENABLED=0",
		),
	}
	err := c.Run()
	if err != nil {
		return nil, err
	}
	out, err := runBin(h, strings.NewReader(`{"name": "user123"}`), "/tmp/idem_user")
	return out, err
}
