package idem

import (
	"io/fs"

	"github.com/msaher/idem/share"
)

type FileConfig share.FileConfig
type FileResult share.FileResult

func File(path string) *FileConfig {
	return &FileConfig{
		F_path: path,
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
	var res FileResult
	err := run(h, fc, "idem_file", &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}
