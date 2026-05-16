package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/msaher/idem/share"
)

var packageMangers = map[string]string {
	"alpine": "apk",
	"ubuntu": "apt",
	"debian": "apt",
	"arch": "pacman",
}

var knownPackageManagers = map[string]struct{} {
	"apk": {},
	"pacman": {},
	"apt": {},
}

func resolvePackageManager(req *share.PackageConfig) (string, error) {
	var manager = req.F_manager
	if manager == "" {
		// get right os
		var osName string
		b, _ := os.ReadFile("/etc/os-release")
		for _, line := range strings.Split(string(b), "\n") {
			if strings.HasPrefix(line, "ID=") {
				osName = strings.Trim(strings.TrimPrefix(line, "ID="), `"`)
				break
			}
		}

		if osName == "" {
			return "", errors.New("cant determine operating system")
		}

		manager = packageMangers[osName]
		if manager == "" {
			return "", fmt.Errorf("operating system is %q, but don't know its package manager")
		}
	}

	// TODO: might make caller verify that before passing to remote binary
	if _, ok := knownPackageManagers[manager]; !ok {
		return "", fmt.Errorf("Unknown package manager %q", manager)
	}

	return manager, nil
}

func currentState(manager, name string, res *share.PackageResult) error {
	var err error
	switch manager {
	case "apk":
		err = exec.Command("apk", "info", "-e", name).Run()
	case "apt":
		err = exec.Command("dpkg", "-s", name).Run()
	case "pacman":
		err = exec.Command("pacman", "-Q", name).Run()
	default:
		panic("unreachable")
	}

	if err == nil {
		res.State = "present"
	} else {
		res.State = "absent"
	}

	return nil
}


func run(req *share.PackageConfig, res *share.PackageResult, manager string) error {
	if req.F_state == res.State {
		return nil
	}

	var cmd *exec.Cmd
	switch manager {
	case "apk":
		if req.F_state == "present" {
			cmd = exec.Command("apk", "add", req.F_name)
		} else {
			cmd = exec.Command("apk", "del", req.F_name)
		}

	case "apt":
		if req.F_state == "present" {
			cmd = exec.Command("apt-get", "install", "-y", req.F_name)
		} else {
			cmd = exec.Command("apt-get", "remove", "-y", req.F_name)
		}

	case "pacman":
		if req.F_state == "present" {
			cmd = exec.Command("pacman", "-S", "--noconfirm", req.F_name)
		} else {
			cmd = exec.Command("pacman", "-R", "--noconfirm", req.F_name)
		}

	default:
		panic("unreachable")
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Failed to install with %q:\n%s", manager, string(out))
	}

	return nil
}

func write(res any) {
	b, err := json.MarshalIndent(res, "", "\t")
	if err != nil {
		panic(err)
	}
	b = append(b, '\n')

	os.Stdout.Write(b)
}

func main() {
	var req share.PackageConfig
	// never happens
	if err := json.NewDecoder(os.Stdin).Decode(&req); err != nil {
		panic(err)
	}

	var res share.PackageResult
	manager, err := resolvePackageManager(&req)
	if err != nil {
		res.Error = err.Error()
		write(res)
		return
	}

	if err := currentState(manager, req.F_name, &res); err != nil {
		res.Error = err.Error()
		write(res)
		return
	}

	if req.F_state == res.State {
		write(res)
		return
	}

	if err := run(&req, &res, manager); err != nil {
		res.Error = err.Error()
		write(res)
		return
	}

	if err := currentState(manager, req.F_name, &res); err != nil {
		res.Error = err.Error()
		write(res)
		return
	}

	if res.State != req.F_state {
		res.Changed = true
	}

	write(res)
}
