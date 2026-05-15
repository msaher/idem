package idem

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"

	"github.com/msaher/idem/share"
	"golang.org/x/crypto/ssh"
)

type FileConfig share.FileConfig
type FileResult share.FileResult

func File(path string) *FileConfig {
	return &FileConfig{
		F_path: path,
		F_mode: 0644,
	}
}

func (fc *FileConfig) Mode(m fs.FileMode) *FileConfig  {
	fc.F_mode = m
	return fc
}

func (fc *FileConfig) Owner(o string) *FileConfig {
	fc.F_owner = o
	return fc
}


func (fc *FileConfig) State(s string) *FileConfig {
	fc.F_state = s
	return fc
}

func (fc *FileConfig) Run(h *HostConfig) (*FileResult, error) {
	bin := "/tmp/idem_file"
	// TODO: embed these source files and precompile them instead of compiling everytime
	c := exec.Cmd{
		Path: "/usr/bin/go",
		Args: []string{"go", "build", "-o", bin, "./remote/file/"},
		Env: append(os.Environ(),
			"CGO_ENABLED=0",
		),
	}
	var compOut []byte
	compOut, err := c.CombinedOutput()
	if err != nil {
		err = fmt.Errorf("BUG: Failed to compile\n. Output: %s", compOut)
		panic(err)
	}

	jsn, err := json.MarshalIndent(fc, "", "\t")
	if err != nil {
		panic(err) // unreachable
	}

	client, err := h.Dial("tcp")
	if err != nil {
		return nil, err
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
		return nil, err
	} else if err != nil {
		return nil, err
	}

	// read output
	r := bytes.NewReader(out)
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	// TODO: maybe show output if we can't decode? Means binary outputted something other than json.
	res := &FileResult{}
	err = dec.Decode(&res)
	if err != nil {
		return res, err
	}

	return res, nil
}
