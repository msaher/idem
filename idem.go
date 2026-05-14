package idem

import (
	"errors"
	"fmt"
	"slices"
	"strings"

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
		port = 80
	}

	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", h.Host, port), sshConfig)
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
