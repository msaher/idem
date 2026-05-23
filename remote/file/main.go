package main

import (
	"encoding/json"
	"errors"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/msaher/idem/share"
)

// TODO: support content

func currentState(path string, s *share.FileState) error {
	// get current state
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		s.State = "absent"
		return nil
	} else if err != nil {
		s.State = "unknown"
		return err
	}

	s.Path = path
	if info.IsDir() {
		s.State = "directory"
	} else { // TODO: check links
		s.State = "file"
	}

	s.Mode = info.Mode()

	// check owner
	// NOTE: not portable in non-unix
	stat, ok := info.Sys().(*syscall.Stat_t)
	if ok {
		// uid := stat.Uid
		// gid := stat.Gid
		u, _ := user.LookupId(strconv.Itoa(int(stat.Uid)))
		s.Owner = u.Username
	}

	return nil
}

func run(req *share.FileConfig, before *share.FileState) error {
	if before.State != req.F_state {
		// mismtach --> delete
		err := os.RemoveAll(req.F_path)
		if err != nil {
			return err
		}

		switch req.F_state {
		case "file":
			if req.F_mode == 0 {
				req.F_mode = 0644
			}
			parents := filepath.Dir(req.F_path)
			err := os.MkdirAll(parents, 0755)
			if err != nil {
				return err
			}
			f, err := os.OpenFile(req.F_path, os.O_CREATE, os.FileMode(req.F_mode))
			if err != nil {
				return err
			}
			f.Close()
		case "directory":
			if req.F_mode == 0 {
				req.F_mode = 0755
			}
			err := os.MkdirAll(req.F_path, os.FileMode(req.F_mode))
			if err != nil {
				return err
			}
		case "absent":
			break
		}
	}

	// nothing to do
	if req.F_state == "absent" {
		return nil
	}

	// set owner
	// -1 means do not change the current value
	uid := -1
	gid := -1
	if req.F_owner != "" {
		u, err := user.Lookup(req.F_owner)
		if err != nil {
			return err
		}
		uidInt, err := strconv.Atoi(u.Uid)
		if err != nil {
			return err
		}
		uid = uidInt
	}

	if req.F_group != "" {
		g, err := user.LookupGroup(req.F_group)
		if err != nil {
			return err
		}
		gidInt, err := strconv.Atoi(g.Gid)
		if err != nil {
			return err
		}
		gid = gidInt
	}

	if err := os.Chown(req.F_path, uid, gid); err != nil {
		return err
	}

	if err := os.Chmod(req.F_path, req.F_mode); err != nil {
		return err
	}

	return nil
}

func main() {
	var req share.FileConfig
	var res share.FileResult
	res.Before = &share.FileState{}
	res.After = &share.FileState{}
	if err := json.NewDecoder(os.Stdin).Decode(&req); err != nil {
		panic(err) // unreachable
	}

	err := currentState(req.F_path, res.Before)
	if err != nil {
		res.Error = err.Error()
		share.Write(&res)
		return
	}

	runErr := run(&req, res.Before)
	err = currentState(req.F_path, res.After)
	if err != nil {
		res.Error = err.Error()
	} else if runErr != nil {
		res.Error = runErr.Error()
	}

	if *res.Before != *res.After {
		res.Changed = true
	}

	share.Write(&res)
}
