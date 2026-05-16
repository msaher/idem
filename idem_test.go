package idem_test

import (
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/msaher/idem"
	"golang.org/x/crypto/ssh"
)

const (
	containerName = "test_ssh"
)

func containerCommand(name string, arg ...string) *exec.Cmd {
	a := []string{"podman", "exec", containerName}
	a = append(a, name)
	a = append(a, arg...)
	return exec.Command(a[0], a[1:]...)
}

func TestMain(m *testing.M) {
	containerName := "test_ssh"
	port := "8022"
	cmd := exec.Command("podman", "run", "--detach", "--name", containerName, "--replace", "--rm", "-p", port+":22", "demo_ssh")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		os.Exit(1)
	}

	// wait for container to go live
	time.Sleep(500 * time.Millisecond)

	// NOTE: might want to clean container later if successfuly
	_ = m.Run()
}


var sshConfig = &ssh.ClientConfig {
	User: "myuser",
	Auth: []ssh.AuthMethod {
		ssh.Password("myuserpass"),
	},
	HostKeyCallback: ssh.InsecureIgnoreHostKey(),
}

var h = &idem.HostCtx {
	Host: "127.0.0.1",
	Port: 8022,
	Sudo: true,
	SshConfig: sshConfig,
}


func TestUser(t *testing.T) {
    t.Run("create user", func(t *testing.T) {
        cfg := idem.User("testuser").Groups("wheel")

        _, err := cfg.Run(h)
        if err != nil {
            t.Fatalf("%v", err)
        }

		out, err := containerCommand("id", "testuser").Output()
		if err != nil {
			t.Fatalf("Expected user to be created: %v", err)
		}

		if !strings.Contains(string(out), "wheel") {
			t.Fatalf("Expected user to be added to group")
		}

    })

    t.Run("idempotent", func(t *testing.T) {
        cfg := idem.User("testuser").Groups("wheel")

        res, err := cfg.Run(h)
        if err != nil {
            t.Fatalf("%v", err)
        }

        if res.Changed {
            t.Fatal("expected unchanged")
        }
    })

    t.Run("missing group", func(t *testing.T) {
        cfg := idem.User("baduser").Groups("doesnotexist")

        _, err := cfg.Run(h)
        if err == nil {
            t.Fatal("expected error")
        }
    })
}

func TestFile(t *testing.T) {
	t.Run("create a file", func(t *testing.T) {
		pth := "/a/b/create_file"
		res, err := idem.File(pth).Run(h)
		if err != nil {
			t.Fatalf("%v", err)
		}
		err = containerCommand("test", "-f", pth).Run()
		if err != nil {
			t.Fatalf("Expected path to be created: %v\n%v", err, res)
		}
	})

	t.Run("remove a file", func(t *testing.T) {
		pth := "/a/b/remove_file"
		_, err := idem.File(pth).State("file").Run(h)
		if err != nil {
			t.Fatalf("%v", err)
		}
		_, err = idem.File(pth).State("absent").Run(h)
		if err != nil {
			t.Fatalf("%v", err)
		}
		err = containerCommand("test", "-f", pth).Run()
		exitErr, ok := errors.AsType[*exec.ExitError](err)
		if !ok {
			t.Fatalf("cant run containerCommand: %v", err)
		}
		if exitErr.ExitCode() == 0 {
			t.Fatalf("file still exists in continer")
		}
	})

	t.Run("remove a directory", func(t *testing.T) {
		pth := "/a/b/remove_directory"
		_, err := idem.File(pth).State("directory").Run(h)
		if err != nil {
			t.Fatalf("%v", err)
		}
		_, err = idem.File(pth).State("absent").Run(h)
		if err != nil {
			t.Fatalf("%v", err)
		}
		err = containerCommand("test", "-d", pth).Run()
		exitErr, ok := errors.AsType[*exec.ExitError](err)
		if !ok {
			t.Fatalf("cant run containerCommand: %v", err)
		}
		if exitErr.ExitCode() == 0 {
			t.Fatalf("directory still exists in continer")
		}
	})

	t.Run("file becomes a directory", func(t *testing.T) {
		pth := "/a/b/become_dir"
		_, err := idem.File(pth).State("file").Run(h)
		if err != nil {
			t.Fatalf("%v", err)
		}
		_, err = idem.File(pth).State("directory").Run(h)
		if err != nil {
			t.Fatalf("%v", err)
		}

		err = containerCommand("test", "-d", pth).Run()
		if err != nil {
			t.Fatalf("expected directory to exist: %v", err)
		}
	})

	t.Run("directory becomes a file", func(t *testing.T) {
		pth := "/a/b/become_file"
		_, err := idem.File(pth).State("directory").Run(h)
		if err != nil {
			t.Fatalf("%v", err)
		}
		_, err = idem.File(pth).State("file").Run(h)
		if err != nil {
			t.Fatalf("%v", err)
		}

		err = containerCommand("test", "-f", pth).Run()
		if err != nil {
			t.Fatalf("expected file to exist: %v", err)
		}
	})

	t.Run("set owner", func(t *testing.T) {
		owner := "myuser"
		pth := "/a/b/set_owner"
		res, err := idem.File(pth).Run(h)
		if err != nil {
			t.Fatalf("%v", err)
		}

		err = containerCommand("test", "-f", pth).Run()
		if err != nil {
			t.Fatalf("expected file to exist: %v", err)
		}

		if res.Owner == owner {
			t.Fatalf("expected owner other than %q. Is host context correct?", owner)
		}

		res, err = idem.File(pth).Owner(owner).Run(h)
		if err != nil {
			t.Fatalf("%v", err)
		}
		if res.Owner != owner {
			t.Fatalf("expected owner to be %q", owner)
		}
	})

	t.Run("set permissions", func(t *testing.T) {
		pth := "/a/b/set_perm"
		res, err := idem.File(pth).Mode(fs.FileMode(0755)).Run(h)
		if err != nil {
			t.Fatalf("%v", err)
		}
		err = containerCommand("test", "-f", pth).Run()
		if err != nil {
			t.Fatalf("Expected path to be created: %v\n%v", err, res)
		}

		res, err = idem.File(pth).Mode(fs.FileMode(0777)).Run(h)
		if err != nil {
			t.Fatalf("Expected path to be created: %v\n%v", err, res)
		}

		out, err := containerCommand("stat", "-c", "%a", pth).Output()
		if err != nil {
			t.Fatalf("%v", err)
		}
		out = out[:len(out)-1] // trim newline
		perm, err := strconv.ParseInt(string(out), 8, 0)
		if err != nil {
			t.Fatalf("Unexpected stat output: %v", string(out))
		}

		if perm != 0777 {
			t.Fatalf("Unexpected permission: %o", perm)
		}
	})

	t.Run("no relative paths", func(t *testing.T) {
		pth := "a/b/c" // bad: must start with root
		_, err := idem.File(pth).Run(h)
		if err == nil {
			t.Fatalf("expected an error because of a bad path")
		}
	})
}

func TestPackage(t *testing.T) {
	t.Run("installs a package", func(t *testing.T) {
		res, err := idem.Package("curl").State("present").Run(h)
		if err != nil {
			var msg string
			if res != nil {
				msg = res.Error
			}
			t.Fatalf("%v\n%s", err, msg)
		}
	})

	t.Run("removes a package", func(t *testing.T) {
		_, err := idem.Package("curl").State("absent").Run(h)
		if err != nil {
			t.Fatalf("%v", err)
		}
	})
}
