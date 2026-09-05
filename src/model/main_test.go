package model

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// TestMain puts a stub `docker` in front of the real one for this package's
// tests.
//
// These are rig tests: they drive the TUI and assert on rendered frames. The
// only thing they want from docker is the answer "no containers are running",
// which is the state every fixture here describes. Without a docker on PATH
// they got an error banner instead and 19 of them failed - the reason the
// contributor docs had to list docker as a prerequisite for running the suite.
//
// With a real docker they passed only by luck: the machine happened to have
// nothing running under a compose project named after the fixture. A developer
// with a container called `web` up would have seen them flake. The stub removes
// both problems - the dependency and the ambient state - by answering the same
// way every time.
//
// This is deliberately PATH-level rather than a seam in utils: it fixes every
// docker call these tests make at once, including the ones made from inside
// goroutines the test never sees, and it needs no export.
func TestMain(m *testing.M) {
	if runtime.GOOS == "windows" {
		// The stub is a shell script. On Windows these tests keep whatever
		// docker is installed, which is the behaviour they had everywhere
		// before this existed.
		os.Exit(m.Run())
	}

	dir, err := os.MkdirTemp("", "cais-stub-docker")
	if err != nil {
		fmt.Fprintf(os.Stderr, "stub docker: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(dir)

	// `ps --format json` is the call that matters; an empty array is a valid
	// answer meaning no containers, and ParseContainers accepts it. Everything
	// else succeeds silently, which is enough for the preflight probe to stop
	// reporting docker as missing.
	script := "#!/bin/sh\nfor a in \"$@\"; do\n  if [ \"$a\" = \"ps\" ]; then echo '[]'; exit 0; fi\ndone\nexit 0\n"
	stub := filepath.Join(dir, "docker")
	if err := os.WriteFile(stub, []byte(script), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "stub docker: %v\n", err)
		os.RemoveAll(dir)
		os.Exit(1)
	}

	os.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	// Fail loudly rather than silently falling back to a real docker: a stub
	// that is not being used would make this whole file a lie.
	if got, err := exec.LookPath("docker"); err != nil || got != stub {
		fmt.Fprintf(os.Stderr, "stub docker: PATH resolves docker to %q (%v), want %q\n", got, err, stub)
		os.RemoveAll(dir)
		os.Exit(1)
	}

	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}
