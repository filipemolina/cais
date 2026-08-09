package utils

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/docker/go-units"
)

// DockerSystemDf runs `docker system df --format json` and returns the raw NDJSON output.
func DockerSystemDf() (string, error) {
	command := exec.Command("docker", "system", "df", "--format", "json")
	output, err := command.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("docker system df failed: %w: %s", err, string(output))
	}
	return string(output), nil
}

// DiskUsage represents one row of `docker system df`.
type DiskUsage struct {
	Type        string // "Images", "Containers", "Local Volumes", "Build Cache"
	TotalCount  int
	Active      int
	Size        int64 // bytes
	Reclaimable int64 // bytes
}

// ParseSystemDf parses the NDJSON output of `docker system df --format json`.
// It accepts both a JSON array and NDJSON (one object per line).
func ParseSystemDf(output string) ([]DiskUsage, error) {
	output = strings.TrimSpace(output)
	if output == "" {
		return []DiskUsage{}, nil
	}

	var result []DiskUsage
	// Split lines, ignore empty lines.
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Remove trailing commas if present (NDJSON may have them?)
		line = strings.TrimSuffix(line, ",")
		var entry map[string]string
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			// If line is not a JSON object, maybe the whole output is a JSON array.
			// We'll fallback to trying to unmarshal the whole output as an array.
			continue
		}
		var du DiskUsage
		du.Type = entry["Type"]
		fmt.Sscanf(entry["TotalCount"], "%d", &du.TotalCount)
		fmt.Sscanf(entry["Active"], "%d", &du.Active)
		var err error
		du.Size, err = units.FromHumanSize(entry["Size"])
		if err != nil {
			return nil, fmt.Errorf("parsing size %q: %w", entry["Size"], err)
		}
		// Reclaimable may have a percentage like "42.28GB (70%)" or just "1.311GB".
		reclaimableStr := entry["Reclaimable"]
		// Extract the first token (size) before any space.
		if idx := strings.Index(reclaimableStr, " "); idx != -1 {
			reclaimableStr = reclaimableStr[:idx]
		}
		du.Reclaimable, err = units.FromHumanSize(reclaimableStr)
		if err != nil {
			return nil, fmt.Errorf("parsing reclaimable %q: %w", entry["Reclaimable"], err)
		}
		result = append(result, du)
	}
	if len(result) == 0 {
		// Try to parse as a JSON array.
		var arr []map[string]string
		if err := json.Unmarshal([]byte(output), &arr); err != nil {
			return nil, fmt.Errorf("could not parse docker system df output: %w", err)
		}
		for _, entry := range arr {
			var du DiskUsage
			du.Type = entry["Type"]
			fmt.Sscanf(entry["TotalCount"], "%d", &du.TotalCount)
			fmt.Sscanf(entry["Active"], "%d", &du.Active)
			var err error
			du.Size, err = units.FromHumanSize(entry["Size"])
			if err != nil {
				return nil, fmt.Errorf("parsing size %q: %w", entry["Size"], err)
			}
			reclaimableStr := entry["Reclaimable"]
			if idx := strings.Index(reclaimableStr, " "); idx != -1 {
				reclaimableStr = reclaimableStr[:idx]
			}
			du.Reclaimable, err = units.FromHumanSize(reclaimableStr)
			if err != nil {
				return nil, fmt.Errorf("parsing reclaimable %q: %w", entry["Reclaimable"], err)
			}
			result = append(result, du)
		}
	}
	return result, nil
}
