package idem_test

import (
	"testing"
	"github.com/msaher/idem"
)

func TestFoo(t *testing.T) {
	cfg := idem.User("user123").
	Groups("wheel", "video")

	h := &idem.HostConfig {
		Host: "127.0.0.1",
		Port: 8022,
		User: "myuser",
		Password: "myuserpass",
		Sudo: true,
	}

	res := idem.Run(h, cfg)
	t.Logf("changed: %v\n", res.Changed)
	for _, r := range res.Commands {
		t.Logf("%#v", r)
	}
	if res.Err != nil {
		t.Logf("%#v", res.Err)
	}
}
