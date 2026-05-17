package idem

import (
	"errors"

	"github.com/msaher/idem/share"
)

type CmdConfig share.CmdConfig
type CmdResult share.CmdResult

func Command(name string, args ...string) *CmdConfig {
	argv := []string{name}
	argv = append(argv, args...)
	return &CmdConfig{F_argv: argv}
}

func (cc *CmdConfig) Creates(path string) *CmdConfig {
	cc.F_creates = path
	return cc
}

func (cc *CmdConfig) Removes(path string) *CmdConfig {
	cc.F_removes = path
	return cc
}

func (cc *CmdConfig) Run(h *HostCtx) (*CmdResult, error) {
	var res CmdResult
	err := run(h, cc, "command", "idem_cmd", &res, &res.Changed)
	if err != nil {
		return nil, err
	}
	if res.Error != "" {
		err = errors.New(res.Error)
		h.Logs[len(h.Logs)-1].Err = err
		h.Err = err
	}

	return &res, err
}
