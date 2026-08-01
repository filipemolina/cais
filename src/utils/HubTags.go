package utils

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// hubBaseURL is overridden in tests to point at an httptest.Server - the
// only test seam this file needs (this is the app's first net/http code;
// verified by grep, 2026-08-01 - there is no existing HTTP-client-testing
// pattern elsewhere in this repo to copy).
var hubBaseURL = "https://hub.docker.com"

type Tag struct {
	Name          string
	Architectures []string
}

var versionTagPattern = regexp.MustCompile(`^v?\d+(\.\d+)*$`)

// ListTags fetches up to limit tags for repo, newest first, from Docker
// Hub's tag API - there is no docker-search equivalent for tags (§Research).
// repo without a namespace (e.g. "nginx") is resolved under "library/",
// matching how official images are actually hosted.
func ListTags(repo string, limit int) ([]Tag, error) {
	ns := repo
	if !strings.Contains(repo, "/") {
		ns = "library/" + repo
	}

	url := fmt.Sprintf("%s/v2/repositories/%s/tags?page_size=%d&ordering=last_updated", hubBaseURL, ns, limit)
	client := http.Client{Timeout: 2 * time.Second}

	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetching tags for %s: %w", repo, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetching tags for %s: unexpected status %d", repo, resp.StatusCode)
	}

	var page struct {
		Results []struct {
			Name   string `json:"name"`
			Images []struct {
				Architecture string `json:"architecture"`
			} `json:"images"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		return nil, fmt.Errorf("decoding tags for %s: %w", repo, err)
	}

	tags := make([]Tag, len(page.Results))
	for i, r := range page.Results {
		archs := make([]string, len(r.Images))
		for j, img := range r.Images {
			archs[j] = img.Architecture
		}
		tags[i] = Tag{Name: r.Name, Architectures: archs}
	}
	return tags, nil
}

// BestDefaultTag scans tags (already ordered newest-first by ListTags) for
// the first name that is a bare version string, skipping compound tags
// like "4.0.19-develop" or "stable-alpine3.24-perl". Falls back to
// "latest" if nothing matches - verified 2026-08-01 against the live API
// that this is common, not rare: library/nginx has zero matches in its
// first 50 tags by last_updated, because it pushes every arch/variant
// combination together on each release. This is a real, permanent limit
// of the heuristic, not a bug (D4).
func BestDefaultTag(tags []Tag) string {
	for _, t := range tags {
		if versionTagPattern.MatchString(t.Name) {
			return t.Name
		}
	}
	return "latest"
}
