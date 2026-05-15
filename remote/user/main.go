package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"slices"
)

type Request struct {
	Name string `json:"name,omitempty"`
	Groups []string `json:"groups,omitempty"`
	Append bool `json:"append"`
	DryRun bool `json:"dry_run"`
}

type UserResult struct {
	Changed bool `json:"changed"`
	WouldChange bool `json:"would_change,omitempty"`
	MissingGroups []string `json:"missing_groups,omitempty"`
	Error string `json:"error,omitempty"`
}

func run() (res *UserResult, err error) {
	res = &UserResult{}
	var want Request
	err = json.NewDecoder(os.Stdin).Decode(&want)
	if err != nil {
		return
	}

	// get current state
	var found bool
	u, err := user.Lookup(want.Name)
	if _, ok := errors.AsType[user.UnknownUserError](err); ok {
		found = false
		err = nil
	} else if err != nil {
		return
	} else {
		found = true
	}

	if !found {
		if want.DryRun {
			res.WouldChange = true
			return
		}
		var out []byte
		out, err = exec.Command("adduser", "-D", want.Name).CombinedOutput()
		if err != nil {
			err = fmt.Errorf("Failed to run adduser: %s", string(out))
			return
		}
		res.Changed = true
	}

	// TODO: create home directory?
	// TODO: set passwords
	// TODO: support append := false

	// if not created, look up the user again
	if u == nil {
		u, err = user.Lookup(want.Name)
		if err != nil {
			panic(err)
		}
	}
	userGroupIds, err := u.GroupIds()
	if err != nil {
		return
	}

	var missingGroups []string
	var group *user.Group
	for _, g := range want.Groups {
		group, err = user.LookupGroup(g)
		if _, ok := errors.AsType[user.UnknownGroupError](err); ok {
			missingGroups = append(missingGroups, g)
			continue
		}

		if slices.Index(userGroupIds, group.Gid) == -1 {
			if want.DryRun {
				res.WouldChange = true
				continue
			}
			var out []byte
			out, err = exec.Command("addgroup", u.Username, group.Name).CombinedOutput()
			if err != nil {
				err = fmt.Errorf("Failed to run addgroup: %s", out)
				return
			}
			res.Changed = true
		}
	}

	if missingGroups != nil {
		res.MissingGroups = missingGroups
		res.Error = "missing groups"
	}

	return
}

func main() {
	res, err := run()
	if err != nil && res.Error == "" {
		res.Error = err.Error()
	}

	b, err := json.MarshalIndent(res, "", "\t")
	if err != nil {
		panic(err)
	}
	b = append(b, '\n')

	os.Stdout.Write(b)
}
