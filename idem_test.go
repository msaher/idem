package idem_test

import (
	"testing"
	"github.com/msaher/idem"
	"encoding/json"
)

func TestUser(t *testing.T) {
	h := &idem.HostConfig {
		Host: "127.0.0.1",
		Port: 8022,
		User: "myuser",
		Password: "myuserpass",
		Sudo: true,
	}
	cfg := idem.User("user123").
	Groups("wheel", "video", "bleh")
	res := cfg.Run(h)

	bleh, _ := json.Marshal(cfg)
	t.Logf("%+v", string(bleh))
	t.Logf("%#v", res.Err())
	t.Logf("%+v", res)
}
