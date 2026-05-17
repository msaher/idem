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
	err := run(h, pc, "idem_package", &res)
	if err == NoOp {
		return nil, err
	}

	l := &Log {Name: "package"}
	if err != nil {
		l.Err = err
		h.Logs = append(h.Logs, l)
		return nil, err
	}
	if res.Error != "" {
		err = errors.New(res.Error)
		l.Err = err
		h.Err = err
	}
	l.Changed = res.Changed
	l.Result = &res
	h.Logs = append(h.Logs, l)

	return &res, h.Err
}
