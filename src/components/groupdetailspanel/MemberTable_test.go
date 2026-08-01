package groupdetailspanel

import (
	"slices"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/compose-spec/compose-go/v2/types"
	"github.com/filipemolina/stack-stitcher/src/apptypes"
)

// headingsAt is the column headings the table shows at a given width, read back
// off the rendered header rather than off the widths that produced it.
func headingsAt(width int) []string {
	return strings.Fields(ansi.Strip(renderTableHeader(computeCols(width), width)))
}

// The bug this guards: the table shrank all seven columns to fit, and lipgloss
// pads to Width but does not truncate, so a column narrower than its own
// heading printed over the next one - NAMEIMAGSTATHEALT... - and at the
// narrowest widths wrapped the header onto a second and third line. Every
// heading on the row must be one of the seven, whole.
//
// The floor is 7 rather than 1 because that is the table's honest minimum: two
// columns for the status dot and five for NAME and its gap. Below it the panel
// has less room than one column needs, and the MaxHeight clip in
// renderTableHeader is all that is left - no terminal is that narrow.
func TestMemberTableHeadingsNeverCollide(t *testing.T) {
	for width := 120; width >= 7; width-- {
		header := renderTableHeader(computeCols(width), width)

		if got := lipgloss.Height(header); got != 1 {
			t.Errorf("width %d: header is %d lines tall: %q", width, got, ansi.Strip(header))
		}

		for _, got := range headingsAt(width) {
			if !slices.Contains([]string{"NAME", "IMAGE", "STATE", "HEALTH", "UPTIME", "PORTS"}, got) {
				t.Errorf("width %d: %q is not a whole heading: %q", width, got, ansi.Strip(header))
			}
		}
	}
}

// Columns are given up whole, in dropOrder, and never come back as the panel
// narrows - the same shed-rather-than-mangle rule the footer bar follows.
func TestMemberTableShedsColumnsInPriorityOrder(t *testing.T) {
	previous := map[string]bool{}
	first := true

	for width := 120; width >= 7; width-- {
		shown := headingsAt(width)

		present := map[string]bool{}
		for _, heading := range []string{"NAME", "IMAGE", "STATE", "HEALTH", "UPTIME", "PORTS"} {
			present[heading] = slices.Contains(shown, heading)
		}

		if !first {
			for heading, was := range previous {
				if !was && present[heading] {
					t.Fatalf("width %d: %s came back after being dropped", width, heading)
				}
			}
		}

		// Each pair is (dropped earlier, dropped later).
		for _, pair := range [][2]string{
			{"PORTS", "IMAGE"}, {"IMAGE", "HEALTH"}, {"HEALTH", "UPTIME"},
			{"UPTIME", "STATE"}, {"STATE", "NAME"},
		} {
			if present[pair[0]] && !present[pair[1]] {
				t.Errorf("width %d: %s survived while %s was dropped, which inverts dropOrder",
					width, pair[0], pair[1])
			}
		}

		previous, first = present, false
	}
}

// The name is the row's identity, so it is never given up - a table of states
// with nothing to attach them to says nothing at all.
func TestMemberTableAlwaysKeepsTheName(t *testing.T) {
	for width := 120; width >= 8; width-- {
		if !slices.Contains(headingsAt(width), "NAME") {
			t.Errorf("width %d dropped the NAME column: %q",
				width, ansi.Strip(renderTableHeader(computeCols(width), width)))
		}
	}
}

// A row is one line and fills its width exactly, whatever it holds: the columns
// it was given are the columns it prints, so a row cannot drift from the header
// above it or spill into the row below.
func TestMemberRowsMatchTheHeader(t *testing.T) {
	m := Model{
		services: []types.ServiceConfig{
			{Name: "audiobookshelf", Image: "ghcr.io/advplyr/audiobookshelf:latest", Profiles: []string{"media"}},
		},
		selectedGroup: "media",
	}

	for width := 120; width >= 7; width-- {
		cols := computeCols(width)

		header := renderTableHeader(cols, width)
		row := m.renderMemberRow(cols, width, m.services[0])

		if got := lipgloss.Height(row); got != 1 {
			t.Errorf("width %d: row is %d lines tall: %q", width, got, ansi.Strip(row))
		}
		if got, want := lipgloss.Width(row), lipgloss.Width(header); got != want {
			t.Errorf("width %d: row is %d columns wide, header %d", width, got, want)
		}
	}
}

// Every cell keeps a column of gap after it. Truncating to the full column let
// a long name run flush into the next value - `navidromedeluan/n…` - which
// reads as one field rather than two.
func TestMemberRowCellsDoNotTouch(t *testing.T) {
	m := Model{
		services: []types.ServiceConfig{
			{Name: "averyveryverylongservicename", Image: "ghcr.io/advplyr/audiobookshelf:latest", Profiles: []string{"media"}},
		},
		selectedGroup: "media",
	}

	for width := 120; width >= 20; width-- {
		cols := computeCols(width)

		// Indexed by rune, not by byte: the status dot and the ellipsis are one
		// cell each but three bytes.
		row := []rune(ansi.Strip(m.renderMemberRow(cols, width, m.services[0])))

		var shown []string
		for _, name := range columnOrder {
			if cols.get(name) > 0 {
				shown = append(shown, name)
			}
		}

		// Walk the column boundaries: the last cell of every column but the
		// last must be blank, which is the gap. The last column's trailing
		// space is the row's own padding and carries no meaning.
		at := 0
		for i, name := range shown {
			at += cols.get(name)

			if i == len(shown)-1 || at > len(row) {
				continue
			}

			if boundary := row[at-1]; boundary != ' ' {
				t.Errorf("width %d: the %s column runs flush into the next (%q at %d): %q",
					width, name, string(boundary), at-1, string(row))
			}
		}
	}
}

// Pins D1: the PORTS column reads the compose file, not docker compose ps.
// A container's runtime Ports string disagrees with the file the moment the
// service is edited and not recreated, and the file is the app's source of
// truth - see docs/plans/group-table-legibility.md.
func TestMemberRowPortsComeFromTheFile(t *testing.T) {
	m := Model{
		services: []types.ServiceConfig{
			{
				Name:     "navidrome",
				Profiles: []string{"media"},
				Ports: []types.ServicePortConfig{
					{Published: "4533", Target: 4533, Protocol: "tcp"},
				},
			},
		},
		containers: []apptypes.DockerContainer{
			{Service: "navidrome", State: "running", Ports: "0.0.0.0:9999->9999/tcp"},
		},
		selectedGroup: "media",
	}

	width := 120
	row := ansi.Strip(m.renderMemberRow(computeCols(width), width, m.services[0]))

	if strings.Contains(row, "9999") {
		t.Errorf("row reads the runtime Ports string, not the file: %q", row)
	}
	if !strings.Contains(row, "4533") {
		t.Errorf("row does not show the file's published port: %q", row)
	}
}
