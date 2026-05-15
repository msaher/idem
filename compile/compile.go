// Compile remote binaries
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func main() {

	entries, err := os.ReadDir("remote")
	if err != nil {
		panic(err)
	}

	// TODO: compile for other architectures
	// Also, might want to make it run in parallel if things start to feel slow
	// TODO: add architectures
	for _, e := range entries {
		src := "./remote/" + e.Name()
		binPath := filepath.Join("compile", "bin", fmt.Sprintf("idem_%s", e.Name()))
		c := exec.Cmd{
			Path: "/usr/bin/go",
			Args: []string{"go", "build", "-o", binPath, src},
			Stdout: os.Stdout,
			Stderr: os.Stderr,
			Env: append(os.Environ(),
				"CGO_ENABLED=0",
			),
		}
		if err := c.Run(); err != nil {
		    fmt.Fprintf(os.Stderr, "failed to compile %s: %v\n", src, err)
		    os.Exit(1)
		}
	}
}
