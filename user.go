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

// User returns a new user configuration for the given username. Defaults to
// state "present" with home directory creation enabled
func User(name string) *UserConfig {
	return &UserConfig {
		F_name: name,
		F_state: "present",
		F_createHome: true,
	}
}

// State sets the desired state: "present" or "absent".
func (cfg *UserConfig) State(s string) *UserConfig {
	cfg.F_state = s
	return cfg
}

// Append sets whether groups should be appended or replace existing groups.
func (cfg *UserConfig) Append(v bool) *UserConfig {
	cfg.F_append = true
	return cfg
}

// Groups sets the desired groups for the user.
func (cfg *UserConfig) Groups(groups ...string) *UserConfig {
	cfg.F_groups = groups
	return cfg
}

// Password sets the user's *hashed* password.
func (cfg *UserConfig) Password(p string) *UserConfig {
	cfg.F_password = p
	return cfg
}

// CreateHome sets whether to create a home directory. Defaults to true.
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
