package idem

import (
	"errors"
	"github.com/msaher/idem/share"
)

type PackageConfig share.PackageConfig
type PackageResult share.PackageResult

func Package(name string) *PackageConfig {
	return &PackageConfig{F_name: name}
}

func (pc *PackageConfig) Manager(manager string) *PackageConfig {
	pc.F_manager = manager
	pc.F_state = "present"
	return pc
}

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
