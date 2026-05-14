package idem

import (
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"io"
	"os"
	"os/exec"

	"golang.org/x/crypto/ssh"
)

// TODO: worry about "%s" idk shell injections

type UserState struct {
	Name string
	Exists bool
	Groups []string
}

func getUserState(client *ssh.Client, name string, sudo bool) (*UserState, error) {
	state := &UserState{ Name: "name" }
	_, err := run(client, fmt.Sprintf("id %s", name), sudo) // injection?
	if err == nil {
		state.Exists = true
	} else {
		exitErr, _ := errors.AsType[*ssh.ExitError](err)
		if exitErr == nil { // command didnt even run
			return state, err
		}
		if exitErr.ExitStatus() != 1 { // run but weird exit
			return state, err
		}
	}

	if !state.Exists {
		return state, nil
	}

	// groups
	out, err := run(client, fmt.Sprintf("id -Gn %s", name), sudo)
	if err != nil {
		return state, err
	}
	out = out[:len(out)-1] // trim new line character
	state.Groups = strings.Split(out, " ")

	return state, nil
}

type CommandResult struct {
	Cmd string
	Output string
}

type Result struct {
	Changed bool
	Commands []*CommandResult
	Err error
}

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

func run(client *ssh.Client, cmd string, sudo bool) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	if sudo {
		cmd = "sudo " + cmd
	}
	out, err := session.CombinedOutput(cmd)
	return string(out), err
}

func runAppend(client *ssh.Client, cmd string, sudo bool, res *Result) error {
	out, err := run(client, cmd, sudo)
	if res != nil {
		res.Commands = append(res.Commands, &CommandResult { cmd, out })
	}
	if err != nil {
		res.Err = err
	}
	return err
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

// TODO: support other types... Make this polymorphic
func Run(h *HostConfig, want *UserConfig) (res *Result) {
	res = &Result{}
	client, err := h.Dial("tcp")
	if err != nil {
		res.Err = err
		return
	}

	state, err := getUserState(client, want.Name, h.Sudo)
	if err != nil {
		res.Err = fmt.Errorf("can't get user state: %w", err)
		return
	}

	if !state.Exists {
		// TODO: handle passwords
		err = runAppend(client, fmt.Sprintf("adduser -D %s", want.Name), h.Sudo, res)
		if err != nil {
			return
		}
		res.Changed = true
	}

	for _, g := range want.groups {
		// skip if already exists
		if slices.Index(state.Groups, g) != -1 {
			continue
		}

		cmd := fmt.Sprintf("addgroup %s %s", want.Name, g)
		err = runAppend(client, cmd, h.Sudo, res)
		if err != nil {
			return
		}
		res.Changed = true
		state.Groups = append(state.Groups, g) // <-- critical fix
	}

	return res
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
	defer w.Close()

	// fmt.Println("poorManScp enter")
	// _, err = io.Copy(w, r)
	// fmt.Println("poorManScp done")
	// w.Close()

	go func() {
		defer w.Close()
		io.Copy(w, r)
	}()

	// NOTE: dumb paths will cause bugs or injections, but its fine since we
	// pass the paths ourselves not from user input
	err = ses.Run(fmt.Sprintf("cat > %s", dstPath))
	return err
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

func RunBin(h *HostConfig, want *UserConfig) ([]byte, error) {
	// compile
	// TODO: embed these source files
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
