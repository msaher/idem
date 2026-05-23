package main

// TODO: this is half baked. Still have some stuff left to do but you get the idea
import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"slices"

	"github.com/msaher/idem/share"
)

func apply(req *share.UserConfig, before *share.UserState, changed *bool) error {
	if req.F_state == "present" && before.State == "absent" {
		out, err := exec.Command("adduser", "-D", req.F_name).CombinedOutput()
		if err != nil {
			return fmt.Errorf("Failed to run adduser: %s", string(out))
		}
		*changed = true
	} else if req.F_state == "absent" && before.State == "present" {
		out, err := exec.Command("deluser", req.F_name).CombinedOutput()
		if err != nil {
			return fmt.Errorf("Failed to run deluser: %s", string(out))
		}
		*changed = true
		return nil
	}

	// TODO: add non-append option goups.
	// At this point the groups already
	// exist within the system. We just have to add them
	for _, g := range req.F_groups {
		if slices.Index(before.Groups, g) == -1 { // not found
			out, err := exec.Command("addgroup", req.F_name, g).CombinedOutput()
			if err != nil {
				return fmt.Errorf("Failed to run addgroup: %s", out)
			}
			*changed = true
		}
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
