package utils

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// DockerMemTotal runs `docker info --format '{{.MemTotal}}'` and returns the total memory in bytes.
func DockerMemTotal() (int64, error) {
	command := exec.Command("docker", "info", "--format", "{{.MemTotal}}")
	output, err := command.CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("docker info failed: %w: %s", err, string(output))
	}
	// Output is like "33304059904" (bytes) or may have trailing newline.
	str := strings.TrimSpace(string(output))
	mem, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parsing MemTotal %q: %w", str, err)
	}
	return mem, nil
}
