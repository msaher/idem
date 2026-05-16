package main

import (
	"errors"
	"os"
	"encoding/json"
	"strconv"
	"os/user"
	"syscall"
	"path/filepath"

	"github.com/msaher/idem/share"
)

// TODO: support content

func currentState(path string, res *share.FileResult) error {
	// get current state
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		res.State = "absent"
		return nil
	} else if err != nil {
		res.State = "unknown"
		return err
	}

	res.Path = path
	if info.IsDir() {
		res.State = "directory"
	} else { // TODO: check links
		res.State = "file"
	}

	res.Mode = info.Mode()

	// check owner
	// NOTE: not portable in non-unix
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		// not Unix or unsupported
	}

	uid := stat.Uid
	// gid := stat.Gid

	u, _ := user.LookupId(strconv.Itoa(int(uid)))
	res.Owner = u.Username

	return nil
}

func run(req *share.FileConfig, res *share.FileResult) error {

	if res.State != req.F_state {
		if req.F_state != "absent" {
			err := os.RemoveAll(req.F_path)
			if err != nil {
				return err
			}
		}

		switch req.F_state {
		case "file":
			mode := req.F_mode
			if mode == 0 {
				mode = 0644
			}
			parents := filepath.Dir(req.F_path)
			err := os.MkdirAll(parents, 0755)
			if err != nil {
				return err
			}
			f, err := os.OpenFile(req.F_path, os.O_CREATE, os.FileMode(mode))
			if err != nil {
				return err
			}
			f.Close()
		case "directory":
			mode := req.F_mode
			if mode == 0 {
				mode = 0755
			}
			err := os.MkdirAll(req.F_path, os.FileMode(mode))
			if err != nil {
				return err
			}
		case "absent":
			err := os.RemoveAll(req.F_path)
			if err != nil {
				return err
			}
		}
	}

	// nothing to do
	if req.F_state == "absent" {
		return nil
	}

	// set owner
	uid := -1
	gid := -1
	if req.F_owner != "" {
		u, err := user.Lookup(req.F_owner)
		if err != nil {
			return err
		}
		uidInt, err := strconv.Atoi(u.Uid)
		if err != nil {
			panic(err) // unreachanble. uid has to be int
		}
		uid = uidInt

		if err := os.Chown(req.F_path, uid, gid); err != nil {
			return err
		}
	}

	// set mode
	if err := os.Chmod(req.F_path, req.F_mode); err != nil {
		return err
	}

	return nil
}

func write(res *share.FileResult) {
	b, err := json.MarshalIndent(res, "", "\t")
	if err != nil {
		panic(err)
	}
	b = append(b, '\n')

	os.Stdout.Write(b)
}

func main() {
	var req share.FileConfig
	err := json.NewDecoder(os.Stdin).Decode(&req)
	// should never happen
	if err != nil {
		panic(err)
	}

	var before share.FileResult
	var after share.FileResult
	err = currentState(req.F_path, &before)
	if err != nil {
		before.Error = err.Error()
		write(&before)
		return
	}

	runErr := run(&req, &before)
	err = currentState(req.F_path, &after)
	if err != nil {
		after.Error = err.Error()
	} else if runErr != nil {
		after.Error = runErr.Error()
	}

	if before != after {
		after.Changed = true
	}

	write(&after)
}
