package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"slices"
	"strings"

	"github.com/msaher/idem/share"
)

func apply(req *share.UserConfig, before *share.UserState, changed *bool) error {
	switch req.F_state {
	case "present":
		if before.State == "absent" {
			var cmd *exec.Cmd
			if share.Has("useradd") {
				args := []string{"useradd"}
				if req.F_createHome {
					args = append(args, "--create-home")
				} else {
					args = append(args, "--no-create-home")
				}
				args = append(args, req.F_name)
				cmd = exec.Command(args[0], args[1:]...)
			} else {
				args := []string{"adduser", "-D"}
				if !req.F_createHome {
					args = append(args, "-H")
				}
				args = append(args, req.F_name)
				cmd = exec.Command(args[0], args[1:]...)
			}
			out, err := cmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("Failed to run %s: %s", cmd.Path, string(out))
			}
			*changed = true
		}
	case "absent":
		if before.State == "present" {
			var cmd *exec.Cmd
			if share.Has("userdel") {
				cmd = exec.Command("userdel", req.F_name)
			} else {
				cmd = exec.Command("deluser", req.F_name)
			}
			out, err := cmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("Failed to run %s: %s", cmd.Path, string(out))
			}
			*changed = true
			return nil
		}
		return nil
	}

	switch share.Has("usermod") && len(req.F_groups) > 0 {
	case true:
		args := []string{"usermod", "-G"}
		if req.F_append {
			args = append(args, "-a")
		}
		args = append(args, req.F_groups...)
		args = append(args, req.F_name)
		cmd := exec.Command(args[0], args[1:]...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("Failed to run %s: %s", cmd.Args, string(out))
		}

		// Did we change anything?
		for _, g := range req.F_groups {
			if slices.Index(before.Groups, g) == -1 {
				*changed = true
			}
		}
	case false:
		// non-append is such a pain. Unsupported for now
		for _, g := range req.F_groups {
			if slices.Index(before.Groups, g) != -1 {
				continue
			}
			var cmd *exec.Cmd
			cmd = exec.Command("addgroup", req.F_name, g)
			out, err := cmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("Failed to add group (%s) %s: %s", cmd.Path, g, out)
			}
			*changed = true
		}
	}


	if req.F_password != "" {
		cmd := exec.Command("chpasswd")

		cmd.Stdin = strings.NewReader(
			fmt.Sprintf("%s:%s", req.F_name, req.F_password),
		)

		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("chpasswd failed: %s", out)
		}

		*changed = true
	}

	return nil
}

func currentState(req *share.UserConfig, state *share.UserState) error {
	// check existence
	usr, err := user.Lookup(req.F_name)
	if _, ok := errors.AsType[user.UnknownUserError](err); ok {
		state.State = "absent"
		return nil
	} else if err != nil {
		return err
	} else {
		state.State = "present"
	}

	// check groups
	gids, err := usr.GroupIds()
	if err != nil {
		return err
	}
	for _, gid := range gids {
		g, err := user.LookupGroupId(gid)
		if err != nil {
			return err
		}
		state.Groups = append(state.Groups, g.Name)
	}
	return nil
}

func main() {
	var req share.UserConfig
	var res share.UserResult
	res.Before = &share.UserState{}
	res.After = &share.UserState{}
	if err := json.NewDecoder(os.Stdin).Decode(&req); err != nil {
		panic(err) // unreachable
	}

	err := currentState(&req, res.Before)
	if err != nil {
		res.Error = err.Error()
		share.Write(&res)
		return
	}

	// look for missing groups
	var missingGroups []string
	if req.F_state == "present" {
		// are the needed groups even present in the system?
		for _, g := range req.F_groups {
			_, err := user.LookupGroup(g)
			if _, ok := errors.AsType[user.UnknownGroupError](err); ok {
				missingGroups = append(missingGroups, g)
			} else if err != nil {
				res.Error = err.Error()
				share.Write(&res)
				return
			}
		}

		if len(missingGroups) > 0 {
			res.Error = "missing groups"
			res.MissingGroups = missingGroups
			share.Write(&res)
			return
		}
	}

	// NOTE: doing non-append with busy box is not supported
	if len(req.F_groups) > 0 && !req.F_append && !share.Has("usermod") {
		res.Error = "setting groups without appending in busybox is NOT supported"
		share.Write(&res)
		return
	}


	errAply := apply(&req, res.Before, &res.Changed)

	if err = currentState(&req, res.After); err != nil {
		res.Error = "cant query after state: " + err.Error()
		share.Write(&res)
		return
	}

	if errAply != nil {
		res.Error = errAply.Error()
	}

	share.Write(&res)
}
