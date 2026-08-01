package chrome

import (
	"strings"
	"testing"

	"github.com/mattn/go-runewidth"
)

// Two widths below were corrected from docs/plans/group-table-legibility.md's
// Tests table: "ghcr.io/hotio/sonarr:latest" at 15 and
// "docker.io/library/postgres:latest" at 20 both fit one rung earlier than
// the plan's table claimed ("hotio/sonarr" is 12 columns, "library/postgres"
// is 16 - both fit under the plan's stated width), so the ladder's first-fit
// rule genuinely returns something wider than the plan's example. The widths
// here are narrowed just enough to land on the rung the plan meant to
// illustrate; every other case matches the plan's table exactly.
func TestShortImage(t *testing.T) {
	cases := []struct {
		ref   string
		width int
		want  string
	}{
		{"lscr.io/linuxserver/kavita:latest", 40, "lscr.io/linuxserver/kavita"},
		{"lscr.io/linuxserver/kavita:latest", 20, "linuxserver/kavita"},
		{"lscr.io/linuxserver/kavita:latest", 15, "kavita"},
		{"ghcr.io/hotio/sonarr:latest", 11, "sonarr"},
		{"postgres:16-alpine", 40, "postgres:16-alpine"},
		{"postgres:16-alpine", 10, "postgres"},
		{"postgres", 40, "postgres"},
		{"docker.io/library/postgres:latest", 15, "postgres"},
		{"linuxserver/kavita", 30, "linuxserver/kavita"},
		{"linuxserver/kavita", 12, "kavita"},
		{"registry:5000/app:v2", 40, "registry:5000/app:v2"},
		{"registry:5000/app:v2", 12, "app:v2"},
		{"postgres@sha256:9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08", 40, "postgres@9f86d08"},
		{"postgres@sha256:9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08", 10, "postgres"},
		{"", 10, ""},
		{"kavita", 3, "ka…"},
		{"kavita", 0, ""},
	}

	for _, c := range cases {
		t.Run(c.ref, func(t *testing.T) {
			if got := ShortImage(c.ref, c.width); got != c.want {
				t.Errorf("ShortImage(%q, %d) = %q, want %q", c.ref, c.width, got, c.want)
			}
		})
	}
}

// The defect this plan fixes: a registry host truncated mid-word
// ("lscr.io/linuxse…"). A whole registry (or namespace) beside a "/" is
// fine - rung 0 and rung 1 do exactly that - so the invariant this pins is
// narrower: Truncate's ellipsis and a path separator never appear together.
// That holds by construction, because Truncate (rung 4) only ever runs
// after the registry and namespace have already been dropped in full.
func TestShortImageNeverRendersAFragmentOfARegistry(t *testing.T) {
	refs := []string{
		"lscr.io/linuxserver/kavita:latest",
		"ghcr.io/hotio/sonarr:latest",
		"docker.io/library/postgres:latest",
		"registry:5000/app:v2",
		"postgres@sha256:9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08",
	}

	for _, ref := range refs {
		for width := 1; width <= 40; width++ {
			got := ShortImage(ref, width)
			if strings.Contains(got, "…") && strings.Contains(got, "/") {
				t.Fatalf("ShortImage(%q, %d) = %q truncates alongside a path separator", ref, width, got)
			}
		}
	}
}

func TestShortImageNeverExceedsItsWidth(t *testing.T) {
	refs := []string{
		"lscr.io/linuxserver/kavita:latest",
		"ghcr.io/hotio/sonarr:latest",
		"docker.io/library/postgres:latest",
		"registry:5000/app:v2",
		"postgres@sha256:9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08",
		"kavita",
	}

	for _, ref := range refs {
		for width := 1; width <= 40; width++ {
			if got := ShortImage(ref, width); runewidth.StringWidth(got) > width {
				t.Fatalf("ShortImage(%q, %d) = %q, width %d exceeds %d", ref, width, got, runewidth.StringWidth(got), width)
			}
		}
	}
}
