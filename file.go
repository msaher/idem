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
	l := &Log{Name: "file"}
	if fc.F_path[0] != '/' {
		err := BadPathErr
		h.Err = err
		l.Err = err
		h.Logs = append(h.Logs, l)
		return nil, err
	}

	var res FileResult
	err := run(h, fc, "idem_file", &res)
	if err == NoOp {
		return nil, err
	}

	l.Changed = res.Changed
	l.Err = err
	l.Result = &res
	h.Logs = append(h.Logs, l)
	return &res, err
}
