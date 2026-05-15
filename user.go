package idem

import (
	"bytes"
	"encoding/json"
	"errors"
	"os/exec"
	"os"
	"fmt"

	"golang.org/x/crypto/ssh"

	"github.com/msaher/idem/share"
)

var (
	MissingGroupsErr = errors.New("missing groups")
)

type UserResult share.UserResult

type UserConfig share.UserConfig

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


func (u *UserConfig) Run(h *HostConfig) (res *UserResult, err error) {
	// TODO: embed these source files and precompile them instead of compiling everytime
	c := exec.Cmd{
		Path: "/usr/bin/go",
		Args: []string{"go", "build", "-o", "/tmp/idem_user", "./remote/user/"},
		Env: append(os.Environ(),
			"CGO_ENABLED=0",
		),
	}
	var compOut []byte
	compOut, err = c.CombinedOutput()
	if err != nil {
		err = fmt.Errorf("BUG: Failed to compile\n. Output: %s", compOut)
		panic(err)
	}

	// prepare payload
	jsn, err := json.MarshalIndent(u, "", "\t")
	if err != nil {
		return
	}
	// TODO: might want to reuse clients
	client, err := h.Dial("tcp")
	if err != nil {
		return
	}
	defer client.Close()
	var sent bool
	out, err, sent := runBin(client, bytes.NewReader(jsn), "/tmp/idem_user", h.Sudo)

	// remove binary if we sent it successfully
	defer func() {
		if sent {
			ses, err := client.NewSession()
			// give up. can't do it
			if err != nil {
				return
			}
			ses.Run(fmt.Sprintf("rm %s", "/tmp/idem_user"))
			ses.Close()
		}
	}()

	// check if it ran successfully
	if sshExitErr, ok := errors.AsType[*ssh.ExitError](err); ok {
		err = &ExitErr{sshExitErr, string(out)}
		return
	} else if err != nil {
		return
	}

	// read output
	r := bytes.NewReader(out)
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	// TODO: maybe show output if we can't decode? Means binary outputted something other than json.
	err = dec.Decode(&res)
	if err != nil {
		return
	}

	if res.MissingGroups != nil {
		err = MissingGroupsErr
	} else if res.Error != "" {
		err = errors.New(res.Error)
	}

	return
}
