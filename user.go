package idem

import (
	"errors"

	_ "embed"

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


func (u *UserConfig) Run(h *HostCtx) (*UserResult, error) {
	var res UserResult
	err := run(h, u, "idem_user", &res)
	if err == NoOp {
		return nil, err
	}

	l := &Log {Name :"user"}
	if err != nil {
		l.Err = err
		h.Logs = append(h.Logs, l)
		return nil, err
	}

	if res.MissingGroups != nil {
		err = MissingGroupsErr
	}

	l.Err = err
	h.Logs = append(h.Logs, l)
	return &res, err
}
