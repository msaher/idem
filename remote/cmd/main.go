package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"

	"github.com/msaher/idem/share"
)

func main() {
	var req share.CmdConfig
	var res share.CmdResult
	// never happens
	if err := json.NewDecoder(os.Stdin).Decode(&req); err != nil {
		panic(err)
	}

	if req.F_creates != "" {
		if _, err := os.Stat(req.F_creates); err == nil {
			share.Write(res)
			return
		}
	}

	if req.F_removes != "" {
		if _, err := os.Stat(req.F_removes); os.IsNotExist(err) {
			share.Write(res)
			return
		}
	}

	var cmd *exec.Cmd
	if len(req.F_argv) == 0 {
		share.Write(res)
		return
	} else if len(req.F_argv) == 1 {
		cmd = exec.Command(req.F_argv[0])
	} else {
		cmd = exec.Command(req.F_argv[0], req.F_argv[1:]...)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	res.Changed = true
	res.Stdout = stdout.String()
	res.Stderr = stderr.String()

	if err != nil {
		res.Error = err.Error()
	}

	if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
		res.ExitCode = exitErr.ExitCode()
	}

	share.Write(res)
}
