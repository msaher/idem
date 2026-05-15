package idem

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
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

// TODO: old binary once we run
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
	out, err := binSes.Output(cmd)
	if err != nil {
		return out, err
	}

	return out, err
}

type UserError struct {
	MissingGroups []string `json:"missing_groups,omitempty"`
	Msg string `json:"msg,omitempty"` // for other errors
}

func (ue *UserError) Error() string {
	return fmt.Sprintf("%#v", ue)
}

type UserResult struct {
	Changed bool `json:"changed"`
	WouldChange bool `json:"would_change,omitempty"`
	Uerror *UserError `json:"error,omitempty"`
	err error
}

func (ur *UserResult) Err() error {
	if ur.Uerror != nil {
		return ur.Uerror
	}
	return ur.err
}

type UserConfig struct {
	F_name string `json:"name"`
	F_groups []string `json:"groups"`
	F_append bool `json:"append"`
	DryRun bool `json:"dry_run"`
}

func User(name string) *UserConfig {
	return &UserConfig {
		F_name: name,
	}
}

func (cfg *UserConfig) Append(v bool) *UserConfig {
	cfg.F_append = true
	return cfg
}

func (cfg *UserConfig) Groups(groups ...string) *UserConfig {
	cfg.F_groups = groups
	return cfg
}


func (u *UserConfig) Run(h *HostConfig) (res *UserResult) {
	// TODO: embed these source files and precompile them instead of compiling everytime
	res = &UserResult{}
	c := exec.Cmd{
		Path: "/usr/bin/go",
		Args: []string{"go", "build", "-o", "/tmp/idem_user", "./remote/user/"},
		Env: append(os.Environ(),
			"CGO_ENABLED=0",
		),
	}
	err := c.Run()
	if err != nil {
		res.err = err
		return
	}

	jsn, err := json.MarshalIndent(u, "", "\t")
	if err != nil {
		res.err = err
		return
	}
	out, err := runBin(h, bytes.NewReader(jsn), "/tmp/idem_user")
	if err != nil {
		res.err = err
		return
	}

	// read output
	r := bytes.NewReader(out)
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	err = dec.Decode(&res)
	if err != nil {
		res.err = err
		return
	}

	return
}
