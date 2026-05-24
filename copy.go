package idem

import (
	"io"
	"io/fs"
	"os"

	"github.com/msaher/idem/share"
)

type CopyConfig share.CopyConfig
type CopyResult share.CopyResult

func CopyFile(src string, dst string) *CopyConfig {
	return &CopyConfig {
		F_src: src,
		F_dest: dst,
	}
}

func CopyContent(content string, dst string) *CopyConfig {
	return &CopyConfig {
		F_content: content,
		F_dest: dst,
	}
}

func (cc *CopyConfig) Owner(o string) *CopyConfig {
	cc.F_owner = o
	return cc
}

func (cc *CopyConfig) Group(g string) *CopyConfig {
	cc.F_group = g
	return cc
}

func (cc *CopyConfig) Mode(m fs.FileMode) *CopyConfig  {
	cc.F_mode = m
	return cc
}

func (cc *CopyConfig) Run(h *HostCtx) (*CopyResult, error) {
	var res CopyResult
	if cc.F_dest[0] != '/' {
		err := BadPathErr
		h.Err = err
		return nil, err
	}

	if cc.F_content == "" {
		file, err := os.Open(cc.F_src)
		if err != nil {
			h.Err = err
			return nil, err
		}
		defer file.Close()
		content, err := io.ReadAll(file)
		if err != nil {
			h.Err = err
			return nil, err
		}
		cc.F_content = string(content)
	}

	err := run(h, cc, "copy", "idem_copy", &res, &res.Changed)
	if err == NoOp {
		return nil, err
	}

	return &res, err
}
