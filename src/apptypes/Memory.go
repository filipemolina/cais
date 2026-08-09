package apptypes

import (
	"strings"

	"github.com/docker/go-units"
)

// DockerContainer is the enriched container type used in the app.
// It is defined in src/apptypes/DockerContainer.go (not shown here but assumed).
// We add the SumContainerMemory function in this file.

// SumContainerMemory adds the used side of every container's MemUsage.
// Returns the total used bytes and how many containers reported one.
func SumContainerMemory(containers []DockerContainer) (int64, int) {
	var total int64
	var count int
	for _, c := range containers {
		if c.MemUsage == "" {
			continue
		}
		// MemUsage format: "21.71MiB / 31.02GiB"
		parts := strings.Split(c.MemUsage, "/")
		if len(parts) != 2 {
			continue
		}
		usedStr := strings.TrimSpace(parts[0])
		used, err := units.RAMInBytes(usedStr)
		if err != nil {
			continue
		}
		total += used
		count++
	}
	return total, count
}
