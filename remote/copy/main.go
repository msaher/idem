package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"path/filepath"
	"os"
	"os/user"
	"strconv"
	"strings"
	"syscall"

	"github.com/msaher/idem/share"
)

func currentState(dst string, s *share.CopyState) error {
	info, err := os.Stat(dst)
	if errors.Is(err, os.ErrNotExist) {
		s.Exists = false
		return nil
	} else if err != nil {
		return err
	}

	// check owner
	// NOTE: not portable in non-unix
	stat, ok := info.Sys().(*syscall.Stat_t)
	if ok {
		// uid := stat.Uid
		// gid := stat.Gid
		u, _ := user.LookupId(strconv.Itoa(int(stat.Uid)))
		g, _ := user.LookupGroupId(strconv.Itoa(int(stat.Gid)))
		s.Owner = u.Username
		s.Group = g.Name
	}

	s.Mode = info.Mode()

	// compute hash
	f, err := os.Open(dst)
	if err != nil {
		return err
	}
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}

	s.Hash = hex.EncodeToString(h.Sum(nil))
	return nil
}

func run(req *share.CopyConfig, before *share.CopyState) error {
	var create bool
	if !before.Exists {
		create = true
	} else {
		h := sha256.New()
		_, err := io.Copy(h, strings.NewReader(req.F_content))
		if err != nil {
			return err
		}

		if hex.EncodeToString(h.Sum(nil)) != before.Hash {
			create = true
		}
	}

	if create {
		err := os.MkdirAll(filepath.Dir(req.F_dest), 0755)
		if err != nil {
			return err
		}
		if req.F_mode == 0 {
			req.F_mode = 0644
		}
		file, err := os.OpenFile(req.F_dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(req.F_mode))
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(file, strings.NewReader(req.F_content))
		if err != nil {
			return err
		}
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

	if err := os.Chown(req.F_dest, uid, gid); err != nil {
		return err
	}

	return nil
}

func main() {
	var req share.CopyConfig
	var res share.CopyResult
	res.Before = &share.CopyState{}
	res.After = &share.CopyState{}
	if err := json.NewDecoder(os.Stdin).Decode(&req); err != nil {
		panic(err) // unreachable
	}

	err := currentState(req.F_dest, res.Before)
	if err != nil {
		res.Error = err.Error()
		share.Write(&res)
		return
	}

	err = run(&req, res.Before)
	if err != nil {
		res.Error = err.Error()
		share.Write(&res)
		return
	}

	err = currentState(req.F_dest, res.After)
	if err != nil && res.Error == "" {
		res.Error = err.Error()
	}

	if *res.Before != *res.After {
		res.Changed = true
	}

	share.Write(&res)
}
