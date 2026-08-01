package addservicemodal

import (
	"errors"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/filipemolina/stack-stitcher/src/cmds"
	"github.com/filipemolina/stack-stitcher/src/utils"
)

// messagesFrom flattens what a command produced, walking batches, so a test
// can assert on a message without caring how it got bundled - the same
// helper groupslist's tests use.
func messagesFrom(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}

	msg := cmd()

	if batch, ok := msg.(tea.BatchMsg); ok {
		var msgs []tea.Msg
		for _, inner := range batch {
			msgs = append(msgs, messagesFrom(inner)...)
		}

		return msgs
	}

	return []tea.Msg{msg}
}

// typeInto drives Update with one letterKey per rune, discarding the
// returned commands - the debounce timers they carry are never meant to run
// in a test, only their message types matter.
func typeInto(m Model, s string) Model {
	for _, ch := range s {
		updated, _ := m.Update(letterKey(ch))
		m = updated.(Model)
	}
	return m
}

func TestTypingBelowTwoCharsNeverDebounces(t *testing.T) {
	m := New("compose.yaml", []string{"web"}).(Model)

	updated, cmd := m.Update(letterKey('n'))
	m = updated.(Model)

	if m.generation != 1 {
		t.Errorf("generation = %d, want 1 after a single keystroke", m.generation)
	}

	for _, msg := range messagesFrom(cmd) {
		if _, ok := msg.(cmds.SearchDebounceMsg); ok {
			t.Fatal("a single keystroke armed a search debounce")
		}
	}
}

func TestStaleDebounceIsIgnored(t *testing.T) {
	m := New("compose.yaml", []string{"web"}).(Model)
	m = typeInto(m, "abc") // three keystrokes, each bumping generation

	if m.generation != 3 {
		t.Fatalf("generation = %d, want 3 after three keystrokes", m.generation)
	}

	updated, cmd := m.Update(cmds.SearchDebounceMsg{Generation: 1})
	m = updated.(Model)

	for _, msg := range messagesFrom(cmd) {
		if _, ok := msg.(cmds.SearchImagesMsg); ok {
			t.Fatal("a superseded debounce fired a search")
		}
	}
	if m.searching {
		t.Error("a superseded debounce set searching")
	}
}

func TestStaleSearchResultIsDiscarded(t *testing.T) {
	m := New("compose.yaml", []string{"web"}).(Model)
	m = typeInto(m, "abc") // generation is now 3

	updated, _ := m.Update(cmds.SearchImagesMsg{
		Generation: 1,
		Results:    []utils.ImageResult{{Name: "linuxserver/sonarr"}},
	})
	m = updated.(Model)

	if n := len(m.results.Items()); n != 0 {
		t.Errorf("a stale result populated the list: %d items", n)
	}
}

func TestCurrentSearchResultPopulatesTheList(t *testing.T) {
	m := New("compose.yaml", []string{"web"}).(Model)
	m = typeInto(m, "abc") // generation is now 3

	results := []utils.ImageResult{
		{Name: "linuxserver/sonarr", Description: "A Sonarr container", Stars: 2127},
		{Name: "linuxserver/radarr", Description: "A Radarr container", Stars: 1796},
	}
	updated, _ := m.Update(cmds.SearchImagesMsg{Generation: 3, Results: results})
	m = updated.(Model)

	if n := len(m.results.Items()); n != len(results) {
		t.Errorf("got %d items, want %d", n, len(results))
	}
	if m.searching {
		t.Error("searching stayed true after a result arrived")
	}
	if m.searchErr != "" {
		t.Errorf("searchErr = %q, want empty after a successful search", m.searchErr)
	}
}

func TestASearchErrorSetsTheQuietMessageNotAPanic(t *testing.T) {
	m := New("compose.yaml", []string{"web"}).(Model)
	m = typeInto(m, "ab") // generation is now 2

	// The exact error shape SearchImages produces for a ghcr.io/... query
	// (docs/plans/image-search.md edge case 9): the daemon routes the search
	// to that registry instead of Hub, which answers 404.
	updated, _ := m.Update(cmds.SearchImagesMsg{
		Generation: 2,
		Err:        errors.New("docker search: exit status 1: Error response from daemon: unexpected status code 404"),
	})
	m = updated.(Model)

	want := "image search unavailable — type the full image reference and press enter"
	if m.searchErr != want {
		t.Errorf("searchErr = %q, want the quiet message %q - never the raw error text", m.searchErr, want)
	}
}
