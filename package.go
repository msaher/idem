package idem

import (
	"errors"
	"github.com/msaher/idem/share"
)

type PackageConfig share.PackageConfig
type PackageResult share.PackageResult

// Package declares the desired state of a package on the remote host.
func Package(name string) *PackageConfig {
	return &PackageConfig{F_name: name}
}

// Manager sets the package manager explicitly. Auto-detected from /etc/os-release if not set.
func (pc *PackageConfig) Manager(manager string) *PackageConfig {
	pc.F_manager = manager
	pc.F_state = "present"
	return pc
}

// State sets the desired state: "present" or "absent".
func (pc *PackageConfig) State(state string) *PackageConfig {
	pc.F_state = state
	return pc
}

func (pc *PackageConfig) Run(h *HostCtx) (*PackageResult, error) {
	var res PackageResult
	err := run(h, pc, "package", "idem_package", &res, &res.Changed)
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

	return &res, h.Err
}
