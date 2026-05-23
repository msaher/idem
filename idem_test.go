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

type Container struct {
	Name string
	Host *idem.HostCtx
}

var (
	cont *Container
	shadowCont *Container

	sshConfig = &ssh.ClientConfig {
		User: "myuser",
		Auth: []ssh.AuthMethod {
			ssh.Password("myuserpass"),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
)

func (c *Container) Command(name string, arg ...string) *exec.Cmd {
	a := []string{"podman", "exec", c.Name}
	a = append(a, name)
	a = append(a, arg...)
	return exec.Command(a[0], a[1:]...)
}

func TestMain(m *testing.M) {
	cont = &Container{Name: "test_ssh"}
	cmd := exec.Command("podman", "run", "--detach", "--name", cont.Name, "--replace", "--rm", "-p", "8022:22", "demo_ssh")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		os.Exit(1)
	}
	cont.Host = &idem.HostCtx {
		Host: "127.0.0.1",
		Port: 8022,
		Sudo: true,
		SshConfig: sshConfig,
	}

	// with shadow utils
	shadowCont = &Container{Name: "test_ssh_shadow"}
	cmd = exec.Command("podman", "run", "--detach", "--name", shadowCont.Name, "--replace", "--rm", "-p", "8023:22", "demo_ssh")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		os.Exit(1)
	}
	shadowCont.Host = &idem.HostCtx {
		Host: "127.0.0.1",
		Port: 8023,
		Sudo: true,
		SshConfig: sshConfig,
	}

	// wait for container to go live
	time.Sleep(500 * time.Millisecond)

	// NOTE: might want to clean container later if successfuly
	_ = m.Run()
}

func TestUser(t *testing.T) {
	cases := []struct {
		name string
		cont *Container
	}{
		{"busybox", cont},
		{"shadow", shadowCont},
	}

	for _, tc := range cases {
		t.Run(tc.name + " create user", func(t *testing.T) {
			cfg := idem.User("testuser").Groups("wheel").Append(true)
			_, err := cfg.Run(tc.cont.Host)
			if err != nil {
				t.Fatalf("%v", err)
			}

			out, err := tc.cont.Command("id", "testuser").Output()
			if err != nil {
				t.Fatalf("Expected user to be created: %v\n%v", err, out)
			}

			if !strings.Contains(string(out), "wheel") {
				t.Fatalf("Expected user to be added to group")
			}

		})

		t.Run(tc.name + " idempotent", func(t *testing.T) {
			cfg := idem.User("testuser").Groups("wheel").Append(true)

			res, err := cfg.Run(tc.cont.Host)
			if err != nil {
				t.Fatalf("%v", err)
			}

			if res.Changed {
				t.Fatal("expected unchanged")
			}
		})

		t.Run(tc.name + " missing group", func(t *testing.T) {
			cfg := idem.User("baduser").Groups("doesnotexist")

			_, err := cfg.Run(tc.cont.Host)
			if err == nil {
				t.Fatal("expected error")
			}
			tc.cont.Host.Err = nil // for next test
		})

		t.Run(tc.name + " remove a user", func(t *testing.T) {
			u := "removeme"
			_, err := idem.User(u).State("present").Run(tc.cont.Host)
			if err != nil {
				t.Fatalf("%v", err)
			}

			err = tc.cont.Command("id", u).Run()
			if err != nil {
				t.Fatalf("Expected user to be created: %v", err)
			}

			res, err := idem.User(u).State("absent").Run(tc.cont.Host)
			if err != nil {
				t.Fatalf("%v", err)
			}
			if res.After.State != "absent" {
				t.Fatalf("Expected user to be absent")
			}
			err = tc.cont.Command("id", u).Run()
			if err == nil {
				t.Fatalf("Expected user to be absent. Result is lying!")
			}
			_, ok := errors.AsType[*exec.ExitError](err)
			if !ok {
				t.Fatalf("cant run container command: %v", err)
			}
		})

		t.Run(tc.name + " set password", func(t *testing.T) {
			username := "withpass"
			_, err := idem.User(username).Run(tc.cont.Host)
			if err != nil {
				t.Fatalf("%v", err)
			}
			before, _ := tc.cont.Command("getent", "shadow", username).Output()

			_, err = idem.User(username).Password("pass123").Run(tc.cont.Host)
			if err != nil {
				t.Fatalf("%v", err)
			}
			after, _ := tc.cont.Command("getent", "shadow", username).Output()
			if string(before) == string(after) {
				t.Fatal("password not changed")
			}
		})
	}

}

func TestFile(t *testing.T) {
	t.Run("create a file", func(t *testing.T) {
		pth := "/a/b/create_file"
		res, err := idem.File(pth).Run(cont.Host)
		if err != nil {
			t.Fatalf("%v", err)
		}
		err = cont.Command("test", "-f", pth).Run()
		if err != nil {
			t.Fatalf("Expected path to be created: %v\n%v", err, res)
		}
	})

	t.Run("remove a file", func(t *testing.T) {
		pth := "/a/b/remove_file"
		_, err := idem.File(pth).State("file").Run(cont.Host)
		if err != nil {
			t.Fatalf("%v", err)
		}
		_, err = idem.File(pth).State("absent").Run(cont.Host)
		if err != nil {
			t.Fatalf("%v", err)
		}
		err = cont.Command("test", "-f", pth).Run()
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
		_, err := idem.File(pth).State("directory").Run(cont.Host)
		if err != nil {
			t.Fatalf("%v", err)
		}
		_, err = idem.File(pth).State("absent").Run(cont.Host)
		if err != nil {
			t.Fatalf("%v", err)
		}
		err = cont.Command("test", "-d", pth).Run()
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
		_, err := idem.File(pth).State("file").Run(cont.Host)
		if err != nil {
			t.Fatalf("%v", err)
		}
		_, err = idem.File(pth).State("directory").Run(cont.Host)
		if err != nil {
			t.Fatalf("%v", err)
		}

		err = cont.Command("test", "-d", pth).Run()
		if err != nil {
			t.Fatalf("expected directory to exist: %v", err)
		}
	})

	t.Run("directory becomes a file", func(t *testing.T) {
		pth := "/a/b/become_file"
		_, err := idem.File(pth).State("directory").Run(cont.Host)
		if err != nil {
			t.Fatalf("%v", err)
		}
		_, err = idem.File(pth).State("file").Run(cont.Host)
		if err != nil {
			t.Fatalf("%v", err)
		}

		err = cont.Command("test", "-f", pth).Run()
		if err != nil {
			t.Fatalf("expected file to exist: %v", err)
		}
	})

	t.Run("set owner", func(t *testing.T) {
		owner := "myuser"
		pth := "/a/b/set_owner"
		res, err := idem.File(pth).Run(cont.Host)
		if err != nil {
			t.Fatalf("%v", err)
		}

		err = cont.Command("test", "-f", pth).Run()
		if err != nil {
			t.Fatalf("expected file to exist: %v", err)
		}

		if res.Owner == owner {
			t.Fatalf("expected owner other than %q. Is host context correct?", owner)
		}

		res, err = idem.File(pth).Owner(owner).Run(cont.Host)
		if err != nil {
			t.Fatalf("%v", err)
		}
		if res.Owner != owner {
			t.Fatalf("expected owner to be %q", owner)
		}
	})

	t.Run("set permissions", func(t *testing.T) {
		pth := "/a/b/set_perm"
		res, err := idem.File(pth).Mode(fs.FileMode(0755)).Run(cont.Host)
		if err != nil {
			t.Fatalf("%v", err)
		}
		err = cont.Command("test", "-f", pth).Run()
		if err != nil {
			t.Fatalf("Expected path to be created: %v\n%v", err, res)
		}

		res, err = idem.File(pth).Mode(fs.FileMode(0777)).Run(cont.Host)
		if err != nil {
			t.Fatalf("Expected path to be created: %v\n%v", err, res)
		}

		out, err := cont.Command("stat", "-c", "%a", pth).Output()
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
		_, err := idem.File(pth).Run(cont.Host)
		if err == nil {
			t.Fatalf("expected an error because of a bad path")
		}
		cont.Host.Err = nil
	})
}

func TestPackage(t *testing.T) {
	t.Run("installs a package", func(t *testing.T) {
		_, err := idem.Package("curl").State("present").Run(cont.Host)
		if err != nil {
			t.Fatalf("%v", err)
		}
	})

	t.Run("removes a package", func(t *testing.T) {
		_, err := idem.Package("curl").State("absent").Run(cont.Host)
		if err != nil {
			t.Fatalf("%v", err)
		}
	})
}

func TestCmd(t *testing.T) {
	t.Run("runs a command", func(t *testing.T) {
		res, err := idem.Command("ls", "-la").Run(cont.Host)
		if err != nil {
			t.Fatalf("%v", err)
		}
		if res.Stdout == "" {
			t.Fatalf("Expected some stdout")
		}
	})

	t.Run("idempotent with creates", func(t *testing.T) {
		somefile := "somefile"
		tsk := idem.Command("touch", "somefile").Creates(somefile)
		res, err := tsk.Run(cont.Host)
		if err != nil {
			t.Fatalf("%v", err)
		}

		if !res.Changed {
			t.Fatalf("expected change to be true")
		}

		res, err = tsk.Run(cont.Host)
		if err != nil {
			t.Fatalf("%v", err)
		}

		if res.Changed {
			t.Fatalf("expected change to be false")
		}
	})

	t.Run("idempotent with removes", func(t *testing.T) {
		deleteme := "deleteme"
		res, err := idem.Command("touch", deleteme).Run(cont.Host)
		if err != nil {
			t.Fatalf("%v", err)
		}

		if !res.Changed {
			t.Fatalf("expected change to be true")
		}

		tsk := idem.Command("rm", deleteme).Removes(deleteme)
		res, err = tsk.Run(cont.Host)
		if err != nil {
			t.Fatalf("%v", err)
		}

		res, err = tsk.Run(cont.Host)
		if res.Changed {
			t.Fatalf("expected change to be false")
		}
	})
}
