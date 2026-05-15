package idem_test

import (
	"fmt"
	"testing"

	"github.com/msaher/idem"
)

// func TestUser(t *testing.T) {
// 	h := &idem.HostConfig {
// 		Host: "127.0.0.1",
// 		Port: 8022,
// 		User: "myuser",
// 		Password: "myuserpass",
// 		Sudo: true,
// 	}
// 	cfg := idem.User("user123").
// 	Groups("wheel", "video", "bleh")
// 	res, err := cfg.Run(h)
// 	if err != nil {
// 		t.Logf("%#v", err)
// 	}
//
// 	t.Logf("%+v", res)
// }

func TestFile(t *testing.T) {
	h := &idem.HostConfig {
		Host: "127.0.0.1",
		Port: 8022,
		User: "myuser",
		Password: "myuserpass",
		Sudo: true,
	}
	cfg := idem.File("/a/b/c").State("file")
	fmt.Println(cfg)
	res, err := cfg.Run(h)
	if err != nil {
		t.Logf("%#v", err)
	}

	t.Logf("%+v", res)
}
