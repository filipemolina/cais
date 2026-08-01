package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
)

// ImageResult is one row of a `docker search` result. docker search
// --format json emits all four fields as strings, including the boolean
// and the number (docs/plans/image-search.md §Research, verified against
// Docker 29.6.0) - Stars and Official are computed at decode time so
// callers never re-parse a string.
type ImageResult struct {
	Name        string
	Description string
	Stars       int
	Official    bool
}

// SearchImages runs `docker search` for term and returns up to limit
// results. It shells out - the same shape as DockerStats and DockerCompose
// in this package - and is not unit tested at this layer; parseSearchOutput
// below carries the decoding logic and is.
//
// A non-nil error here is not necessarily a broken installation: a query
// shaped like a registry hostname (e.g. "ghcr.io/foo/bar") makes the
// daemon route the search to that registry instead of Hub and fail with a
// 404 - verified 2026-08-01, see docs/plans/image-search.md edge case 9.
// Callers must treat every error from this function the same way they
// treat zero results: quietly, never as an alarming failure (D6a).
func SearchImages(term string, limit int) ([]ImageResult, error) {
	cmd := exec.Command("docker", "search", "--format", "json", "--no-trunc", "--limit", strconv.Itoa(limit), term)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("docker search: %w: %s", err, bytes.TrimSpace(output))
	}
	return parseSearchOutput(output)
}

// parseSearchOutput decodes docker search --format json's output: one JSON
// object per line, not a JSON array (verified 2026-08-01). A StarCount that
// doesn't parse as a number degrades to 0 rather than failing the whole
// decode - a cosmetic field is not worth discarding an otherwise-good
// result over.
func parseSearchOutput(output []byte) ([]ImageResult, error) {
	var results []ImageResult

	for _, line := range bytes.Split(output, []byte("\n")) {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		var raw struct {
			Name        string `json:"Name"`
			Description string `json:"Description"`
			StarCount   string `json:"StarCount"`
			IsOfficial  string `json:"IsOfficial"`
		}
		if err := json.Unmarshal(line, &raw); err != nil {
			return nil, fmt.Errorf("decoding docker search output: %w", err)
		}

		stars, _ := strconv.Atoi(raw.StarCount)
		results = append(results, ImageResult{
			Name:        raw.Name,
			Description: raw.Description,
			Stars:       stars,
			Official:    raw.IsOfficial == "true",
		})
	}

	return results, nil
}
