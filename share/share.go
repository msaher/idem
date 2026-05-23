package share

import (
	"encoding/json"
	"os"
	"os/exec"
	"io/fs"
)

type UserState struct {
    State  string   `json:"state"` // "present" or "absent"
    Groups []string `json:"groups,omitempty"`
}

type UserResult struct {
	Changed bool `json:"changed"`
	Error string `json:"error,omitempty"`
	MissingGroups []string `json:"missing_groups,omitempty"`
	Before *UserState `json:"before"`
	After *UserState `json:"after"`
}

type UserConfig struct {
	F_state string `json:"state"`
	F_name string `json:"name"`
	F_groups []string `json:"groups"`
	F_append bool `json:"append"`
	F_password string `json:"password"`
	F_createHome bool `json:"create_home"`
}

type FileConfig struct {
    F_path  string `json:"path"`
    F_mode  fs.FileMode `json:"mode,omitempty"`
    F_owner string `json:"owner,omitempty"`
	F_group string `json:"group"`
    F_state string `json:"state"`
}

type FileState struct {
	State string
	Owner string
	Group string
	Mode fs.FileMode
	Path string
}

type FileResult struct {
	Changed bool `json:"changed"`
	Error string `json:"error,omitempty"`
	Before *FileState `json:"before"`
	After *FileState `json:"after"`
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

func Has(bin string) bool {
	_, err := exec.LookPath(bin)
	return err == nil
}
