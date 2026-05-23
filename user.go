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
		F_state: "present",
		F_createHome: true,
	}
}

// TODO: what to do with invalid states
func (cfg *UserConfig) State(s string) *UserConfig {
	cfg.F_state = s
	return cfg
}

func (cfg *UserConfig) Append(v bool) *UserConfig {
	cfg.F_append = true
	return cfg
}

func (cfg *UserConfig) Groups(groups ...string) *UserConfig {
	cfg.F_groups = groups
	return cfg
}

func (cfg *UserConfig) Password(p string) *UserConfig {
	cfg.F_password = p
	return cfg
}

func (cfg *UserConfig) CreateHome(b bool) *UserConfig {
	cfg.F_createHome = b
	return cfg
}

func (u *UserConfig) Run(h *HostCtx) (*UserResult, error) {
	var res UserResult
	err := run(h, u, "user", "idem_user", &res, &res.Changed)
	if err == NoOp {
		return nil, err
	}

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
