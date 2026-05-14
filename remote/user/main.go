package main

import (
	"os"
	"os/exec"
	"errors"
	"os/user"
	"encoding/json"
	"slices"
	"fmt"
)

type Request struct {
	Name string `json:"name,omitempty"`
	Groups []string `json:"groups,omitempty"`
}

func run() (changed bool, err error) {
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
	} else if err != nil {
		return
	} else {
		found = true
	}

	if !found {
		var out []byte
		out, err = exec.Command("adduser", "-D", want.Name).CombinedOutput()
		if err != nil {
			fmt.Printf("%#v", string(out))
			return
		}
		changed = true
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
		}

		if slices.Index(userGroupIds, group.Gid) == -1 {
			err = exec.Command("addgroup", u.Username, group.Name).Run()
			if err != nil {
				return
			}
			changed = true
		}
	}
	return
}

func main() {
	changed, err := run()
	var code int
	var errMsg string
	if err != nil {
		code = 1
		errMsg = err.Error()
	}

	res := struct {
		Changed bool `json:"changed"`
		Error string `json:"error,omitempty"`
	} {
		Changed: changed,
		Error: errMsg,
	}

	b, err := json.MarshalIndent(res, "", "\t")
	if err != nil {
		panic(err)
	}
	b = append(b, '\n')

	os.Stdout.Write(b)
	os.Exit(code)
}
