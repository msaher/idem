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

func apply(req *share.UserConfig, res *share.UserResult, changed *bool) error {
	if req.F_state == "present" && res.State == "absent" {
		out, err := exec.Command("adduser", "-D", req.F_name).CombinedOutput()
		if err != nil {
			return fmt.Errorf("Failed to run adduser: %s", string(out))
		}
		*changed = true
	} else if req.F_state == "absent" && res.State == "present" {
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
		if slices.Index(res.Groups, g) == -1 { // not found
			out, err := exec.Command("addgroup", req.F_name, g).CombinedOutput()
			if err != nil {
				return fmt.Errorf("Failed to run addgroup: %s", out)
			}
			*changed = true
		}
	}

	return nil
}

func currentState(req *share.UserConfig, res *share.UserResult) error {
	// check existence
	usr, err := user.Lookup(req.F_name)
	if _, ok := errors.AsType[user.UnknownUserError](err); ok {
		res.State = "absent"
		return nil
	} else if err != nil {
		return err
	} else {
		res.State = "present"
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
		res.Groups = append(res.Groups, g.Name)
	}
	return nil
}

func main() {
	var req share.UserConfig
	var before share.UserResult
	if err := json.NewDecoder(os.Stdin).Decode(&req); err != nil {
		panic(err) // unreachable
	}

	err := currentState(&req, &before)
	if err != nil {
		before.Error = err.Error()
		share.Write(&before)
		return
	}

	var missingGroups []string
	if req.F_state == "present" {
		// are the needed groups even present in the system?
		for _, g := range req.F_groups {
			_, err := user.LookupGroup(g)
			if _, ok := errors.AsType[user.UnknownGroupError](err); ok {
				missingGroups = append(missingGroups, g)
			} else if err != nil {
				before.Error = err.Error()
				share.Write(&before)
				return
			}
		}

		if len(missingGroups) > 0 {
			before.Error = "missing groups"
			before.MissingGroups = missingGroups
			share.Write(&before)
			return
		}
	}

	var after share.UserResult
	errAply := apply(&req, &before, &after.Changed)

	if err = currentState(&req, &after); err != nil {
		after.Error = "cant query after state: " + err.Error()
		share.Write(&after)
		return
	}

	if errAply != nil {
		after.Error = errAply.Error()
	}

	share.Write(&after)
}
