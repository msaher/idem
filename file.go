package idem

import (
	"io/fs"
	"errors"

	"github.com/msaher/idem/share"
)

type FileConfig share.FileConfig
type FileResult share.FileResult

var BadPathErr = errors.New("path must start with '/' character")

// File declares the desired state of a file or directory on the remote host.
func File(path string) *FileConfig {
	return &FileConfig{
		F_path: path,
		F_state: "file",
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


// State sets the desired state: "file", "directory", or "absent".
func (fc *FileConfig) State(s string) *FileConfig {
	fc.F_state = s
	return fc
}

func (fc *FileConfig) Group(g string) *FileConfig {
	fc.F_group = g
	return fc
}

// Link sets state to "link" and src as the symlink target.
func (fc *FileConfig) Link(src string) *FileConfig {
	fc.F_state = "link"
	fc.F_src = src
	return fc
}

func (fc *FileConfig) Run(h *HostCtx) (*FileResult, error) {
	if fc.F_path[0] != '/' {
		err := BadPathErr
		h.Err = err
		return nil, err
	}

	var res FileResult
	err := run(h, fc, "file", "idem_file", &res, &res.Changed)
	if err == NoOp {
		return nil, err
	}

	return &res, err
}
