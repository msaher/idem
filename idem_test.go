package idem_test

import (
	// "fmt"
	"testing"

	"github.com/msaher/idem"
	"golang.org/x/crypto/ssh"
)

func TestUser(t *testing.T) {
	sshConfig := &ssh.ClientConfig {
		User: "myuser",
		Auth: []ssh.AuthMethod {
			ssh.Password("myuserpass"),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	h := &idem.HostCtx {
		Host: "127.0.0.1",
		Port: 8022,
		Sudo: true,
		SshConfig: sshConfig,
	}
	cfg := idem.User("user123").
	Groups("wheel", "video", "bleh")
	res, err := cfg.Run(h)
	if err != nil {
		t.Logf("%#v", err)
	}

	t.Logf("%+v", res)
}

// func TestFile(t *testing.T) {
// 	h := &idem.HostConfig {
// 		Host: "127.0.0.1",
// 		Port: 8022,
// 		User: "myuser",
// 		Password: "myuserpass",
// 		Sudo: true,
// 	}
// 	cfg := idem.File("/a/b/c").State("directory")
// 	fmt.Println(cfg)
// 	res, err := cfg.Run(h)
// 	if err != nil {
// 		t.Logf("%#v", err)
// 	}
//
// 	t.Logf("%+v", res)
// }
