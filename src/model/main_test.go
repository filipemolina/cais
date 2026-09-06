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
//
// The stub answers the read-only probes and refuses anything that would change
// state. It used to exit 0 for everything it did not recognise, which meant a
// test could assert its way through a docker action that had never run - the
// same shape as the three tests D8 caught passing for the wrong reason. Every
// test in this package passes with mutations refused, so nothing was relying
// on that fake success.
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

	// No defer for the cleanup: every path out of here ends in os.Exit, which
	// does not run deferred calls. Each one removes the directory itself.
	stub := filepath.Join(dir, "docker")
	if err := os.WriteFile(stub, []byte(stubDockerScript), 0o755); err != nil {
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

// stubDockerScript answers the read-only probes and refuses everything else.
const stubDockerScript = `#!/bin/sh
# ps can appear at any position: the app reaches it both as ` + "`docker ps`" + ` and
# as ` + "`docker compose ... ps`" + `. An empty array is a valid answer meaning no
# containers, and ParseContainers accepts it.
for a in "$@"; do
  if [ "$a" = "ps" ]; then echo '[]'; exit 0; fi
done

# The read-only probes. These must succeed: the preflight check runs
# ` + "`docker compose version`" + ` and ` + "`docker version`" + ` at startup, and a failure
# there puts a "docker unavailable" modal over every test in the package.
case "$1 $2" in
  "compose version"|"context inspect"|"system df") exit 0 ;;
esac
case "$1" in
  version|info|stats) exit 0 ;;
esac

# Everything else changes state. Refuse it loudly rather than reporting a
# success that never happened - a fake success is how an error path stops
# being tested without anyone noticing.
echo "stub docker: refusing unstubbed command: $*" >&2
exit 127
`
