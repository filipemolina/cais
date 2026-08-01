package addservicemodal

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"charm.land/bubbles/v2/list"
	"github.com/charmbracelet/x/ansi"
	"github.com/filipemolina/stack-stitcher/src/cmds"
	"github.com/filipemolina/stack-stitcher/src/utils"
)

func frame(m Model) string {
	return ansi.Strip(m.View().Content)
}

func TestDelegateRendersNameAndOfficialSuffixOverTwoLines(t *testing.T) {
	d := resultsDelegate{width: 40}
	items := []list.Item{
		searchResultItem{result: utils.ImageResult{
			Name:        "linuxserver/sonarr",
			Description: "An all-in-one Sonarr container",
			Stars:       2127,
			Official:    true,
		}},
	}
	cl := list.New(items, d, 40, 2)

	var buf bytes.Buffer
	d.Render(&buf, cl, 0, items[0])

	got := ansi.Strip(buf.String())
	if !strings.Contains(got, "linuxserver/sonarr") {
		t.Errorf("line 1 missing the name, got:\n%s", got)
	}
	if !strings.Contains(got, "official") || !strings.Contains(got, "2127") {
		t.Errorf("line 1 missing the official/stars suffix, got:\n%s", got)
	}
	if !strings.Contains(got, "Sonarr container") {
		t.Errorf("line 2 missing the description, got:\n%s", got)
	}
	lines := strings.Split(got, "\n")
	if len(lines) < 2 || lines[0] == lines[1] {
		t.Errorf("name and description are not on separate lines, got:\n%s", got)
	}
}

func TestDelegateSuffixOmitsOfficialForCommunityImages(t *testing.T) {
	d := resultsDelegate{width: 40}
	items := []list.Item{
		searchResultItem{result: utils.ImageResult{
			Name:        "lscr.io/linuxserver/sonarr",
			Description: "A Sonarr container",
			Stars:       3,
			Official:    false,
		}},
	}
	cl := list.New(items, d, 40, 2)

	var buf bytes.Buffer
	d.Render(&buf, cl, 0, items[0])

	got := ansi.Strip(buf.String())
	if strings.Contains(got, "official") {
		t.Errorf("a community image was marked official, got:\n%s", got)
	}
	if !strings.Contains(got, "3 stars") {
		t.Errorf("missing the star count, got:\n%s", got)
	}
}

func TestViewShowsSearchingSpinnerWhileASearchIsInFlight(t *testing.T) {
	m := New("compose.yaml", []string{"web"}).(Model)
	m = typeInto(m, "ab") // generation is now 2, a debounce is armed

	updated, _ := m.Update(cmds.SearchDebounceMsg{Generation: 2})
	m = updated.(Model)

	if !m.searching {
		t.Fatal("searching should be true after a current debounce fires")
	}
	if !strings.Contains(frame(m), "searching") {
		t.Error("View does not show the searching state while in flight")
	}
}

func TestViewShowsTheQuietErrorNotTheRawText(t *testing.T) {
	m := New("compose.yaml", []string{"web"}).(Model)
	m = typeInto(m, "ab")
	updated, _ := m.Update(cmds.SearchImagesMsg{
		Generation: 2,
		Err:        errors.New("docker search: exit status 1: Error response from daemon: unexpected status code 404"),
	})
	m = updated.(Model)

	got := frame(m)
	if !strings.Contains(got, m.searchErr) {
		t.Errorf("View does not show the quiet error message, got:\n%s", got)
	}
	if strings.Contains(got, "exit status") {
		t.Error("View leaks the raw docker error text")
	}
}

func TestViewHintsToTypeBeforeTwoChars(t *testing.T) {
	m := New("compose.yaml", []string{"web"}).(Model)
	m = typeInto(m, "n")

	if !strings.Contains(frame(m), "Type to search Docker Hub") {
		t.Error("View does not show the type-to-search hint for a short query")
	}
}

func TestViewShowsTheResultsTableOnceTheyArrive(t *testing.T) {
	m := New("compose.yaml", []string{"web"}).(Model)
	m = typeInto(m, "son")
	updated, _ := m.Update(cmds.SearchImagesMsg{
		Generation: 3,
		Results: []utils.ImageResult{{
			Name:        "linuxserver/sonarr",
			Description: "An all-in-one Sonarr container",
			Stars:       2127,
		}},
	})
	m = updated.(Model)

	got := frame(m)
	if !strings.Contains(got, "linuxserver/sonarr") {
		t.Errorf("View does not show the results table, got:\n%s", got)
	}
	if strings.Contains(got, "Type to search") {
		t.Error("View shows the pre-search hint after results arrived")
	}
}
