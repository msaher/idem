package share

import (
	"io/fs"
)

type FileConfig struct {
    F_path  string `json:"path"`
    F_mode  fs.FileMode `json:"mode,omitempty"`
    F_owner string `json:"owner,omitempty"`
    F_state string `json:"state"`
}

type FileResult struct {
	Changed bool `json:"changed"`
	Path string `json:"Path"`
	State string `json:"state"`
	Mode fs.FileMode `json:"mode"`
	Owner string `json:"owner"`
	Error string `json:"error,omitempty"`
}
