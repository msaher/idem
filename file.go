package idem

import (
	"io/fs"
	"errors"

	"github.com/msaher/idem/share"
)

type FileConfig share.FileConfig
type FileResult share.FileResult

var BadPathErr = errors.New("path must start with '/' character")

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


func (fc *FileConfig) State(s string) *FileConfig {
	fc.F_state = s
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
