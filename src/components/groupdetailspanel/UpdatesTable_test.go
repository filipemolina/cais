package groupdetailspanel

import (
	"slices"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/compose-spec/compose-go/v2/types"

	"github.com/filipemolina/cais/src/utils"
)

// upHeadingsAt is the updates table's headings at a given width, read back off
// the rendered header.
func upHeadingsAt(width int) []string {
	return strings.Fields(ansi.Strip(renderUpTableHeader(upComputeCols(width), width)))
}

// The updates table obeys the same heading rule the member table does: every
// heading printed is one of the four, whole, on one line.
func TestUpTableHeadingsNeverCollide(t *testing.T) {
	for width := 120; width >= 9; width-- {
		header := renderUpTableHeader(upComputeCols(width), width)

		if got := lipgloss.Height(header); got != 1 {
			t.Errorf("width %d: header is %d lines tall: %q", width, got, ansi.Strip(header))
		}

		for _, got := range upHeadingsAt(width) {
			if !slices.Contains([]string{"SERVICE", "IMAGE", "LOCAL", "REMOTE"}, got) {
				t.Errorf("width %d: %q is not a whole heading: %q", width, got, ansi.Strip(header))
			}
		}
	}
}

// Columns are given up whole, in upDropOrder - remote, then local, then image
// - and never come back as the panel narrows.
func TestUpTableShedsColumnsInPriorityOrder(t *testing.T) {
	previous := map[string]bool{}
	first := true

	for width := 120; width >= 9; width-- {
		shown := upHeadingsAt(width)

		present := map[string]bool{}
		for _, heading := range []string{"SERVICE", "IMAGE", "LOCAL", "REMOTE"} {
			present[heading] = slices.Contains(shown, heading)
		}

		if !first {
			for heading, was := range previous {
				if !was && present[heading] {
					t.Fatalf("width %d: %s came back after being dropped", width, heading)
				}
			}
		}

		for _, pair := range [][2]string{
			{"REMOTE", "LOCAL"}, {"LOCAL", "IMAGE"}, {"IMAGE", "SERVICE"},
		} {
			if present[pair[0]] && !present[pair[1]] {
				t.Errorf("width %d: %s survived while %s was dropped, which inverts upDropOrder",
					width, pair[0], pair[1])
			}
		}

		previous, first = present, false
	}
}

// The service name is the row's identity, so it is never given up.
func TestUpTableAlwaysKeepsTheService(t *testing.T) {
	for width := 120; width >= 9; width-- {
		if !slices.Contains(upHeadingsAt(width), "SERVICE") {
			t.Errorf("width %d dropped the SERVICE column: %q",
				width, ansi.Strip(renderUpTableHeader(upComputeCols(width), width)))
		}
	}
}

// The table renders only when the selected group has a stale member, and
// lists exactly the stale ones - Unknown and UpToDate members are not rows.
func TestTheUpdatesTableRendersOnlyForAStaleMember(t *testing.T) {
	m := Model{
		services: []types.ServiceConfig{
			{Name: "stale", Image: "redis:7-alpine", Profiles: []string{"core"}},
			{Name: "quiet", Image: "alpine:latest", Profiles: []string{"core"}},
		},
		selectedGroup: "core",
	}

	if got := m.renderUpdatesTable(m.services, 80); got != "" {
		t.Errorf("no stale members, yet the table rendered:\n%s", ansi.Strip(got))
	}

	m.imageUpdates = map[string]utils.ImageUpdate{
		"stale": {Service: "stale", Image: "redis:7-alpine", State: utils.ImageStale,
			LocalDigest: "sha256:a", RemoteDigest: "sha256:b"},
		"quiet": {Service: "quiet", State: utils.ImageUnknown},
	}

	got := ansi.Strip(m.renderUpdatesTable(m.services, 80))
	if !strings.Contains(got, "stale") {
		t.Errorf("the stale member is not in the table:\n%s", got)
	}
	if strings.Contains(got, "quiet") {
		t.Errorf("an Unknown member rendered as a row:\n%s", got)
	}
}

// The digests show at twelve hex characters, which is the shape the plan
// pins and the length two digests can be told apart by.
func TestUpTableShowsTwelveHexDigests(t *testing.T) {
	m := Model{
		services: []types.ServiceConfig{
			{Name: "sonarr", Image: "lscr.io/linuxserver/sonarr:latest", Profiles: []string{"media"}},
		},
		selectedGroup: "media",
		imageUpdates: map[string]utils.ImageUpdate{
			"sonarr": {
				Service:      "sonarr",
				Image:        "lscr.io/linuxserver/sonarr:latest",
				LocalDigest:  "sha256:6ab0b6e7381779332f97b8ca76193e45b0756f38d4c0dcda72dbb3c32061ab99",
				RemoteDigest: "sha256:858f009f9709ce576febc734aa78b8f6d624b82571f9ddb6bda4377c833b3499",
				State:        utils.ImageStale,
			},
		},
	}

	got := ansi.Strip(m.renderUpdatesTable(m.services, 80))

	for _, hex := range []string{"6ab0b6e73817", "858f009f9709"} {
		if !strings.Contains(got, hex) {
			t.Errorf("the table does not show %s:\n%s", hex, got)
		}
	}
	for _, hex := range []string{"6ab0b6e7381779", "858f009f9709ce5"} {
		if strings.Contains(got, hex) {
			t.Errorf("the table shows more than twelve hex characters:\n%s", got)
		}
	}
}

// The NAME cell carries the glyph after the name, and only for a stale
// member - the same statement the services list makes at the row tail.
func TestAMemberWithAnUpdateCarriesTheGlyphInItsName(t *testing.T) {
	m := Model{
		services: []types.ServiceConfig{
			{Name: "stale", Profiles: []string{"core"}},
			{Name: "quiet", Profiles: []string{"core"}},
		},
		selectedGroup: "core",
		imageUpdates: map[string]utils.ImageUpdate{
			"stale": {Service: "stale", State: utils.ImageStale},
			"quiet": {Service: "quiet", State: utils.ImageUpToDate},
		},
	}

	row := ansi.Strip(m.renderMemberRow(computeCols(120), 120, m.services[0]))
	if !strings.Contains(row, "stale") || !strings.Contains(row, "↑") {
		t.Errorf("a stale member's NAME cell carries no glyph: %q", strings.TrimRight(row, " "))
	}

	row = ansi.Strip(m.renderMemberRow(computeCols(120), 120, m.services[1]))
	if strings.Contains(row, "quiet ↑") || strings.Contains(row, "↑") {
		t.Errorf("an UpToDate member drew the glyph: %q", strings.TrimRight(row, " "))
	}
}
