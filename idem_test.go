package idem_test

import (
	"os"
	"os/exec"
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
