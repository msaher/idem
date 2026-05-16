package share

import (
	"encoding/json"
	"os"
	"io/fs"
)

type UserResult struct {
	Changed bool `json:"changed"`
	WouldChange bool `json:"would_change,omitempty"`
	MissingGroups []string `json:"missing_groups,omitempty"`
	Error string `json:"error,omitempty"`
}

type UserConfig struct {
	F_name string `json:"name"`
	F_groups []string `json:"groups"`
	F_append bool `json:"append"`
}


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

type PackageConfig struct {
	F_name string `json:"name"`
	F_manager string `json:"manager"`
	F_state string `json:"state"`
}

type PackageResult struct {
	State string `json:"state"`
	Changed bool `json:"changed"`
	Error string `json:"error,omitempty"`
}

type CmdConfig struct {
    F_argv    []string `json:"argv"`
    F_creates string `json:"creates,omitempty"` // skip if exists
    F_removes string `json:"removes,omitempty"` // skip if absent
}

type CmdResult struct {
    Changed  bool   `json:"changed"`
    Stdout   string `json:"stdout"`
    Stderr   string `json:"stderr"`
    ExitCode int    `json:"exit_code"`
    Error    string `json:"error,omitempty"`
}


func Write(res any) {
	b, err := json.MarshalIndent(res, "", "\t")
	if err != nil {
		panic(err)
	}
	b = append(b, '\n')

	os.Stdout.Write(b)
}

