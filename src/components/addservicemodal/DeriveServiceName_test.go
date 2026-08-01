package addservicemodal

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/filipemolina/stack-stitcher/src/cmds"
	"github.com/filipemolina/stack-stitcher/src/utils"
)

func TestDeriveServiceName(t *testing.T) {
	cases := []struct {
		image string
		want  string
	}{
		{"nginx", "nginx"},
		{"linuxserver/sonarr", "sonarr"},
		{"linuxserver/sonarr:4.7.5", "sonarr"},
		{"postgres@sha256:e4acc22c57ff", "postgres"},
		{"ghcr.io/foo/bar", "bar"},
		{"ghcr.io/foo/bar:v2", "bar"},
	}

	for _, c := range cases {
		if got := deriveServiceName(c.image); got != c.want {
			t.Errorf("deriveServiceName(%q) = %q, want %q", c.image, got, c.want)
		}
	}
}

func TestEnterOnAHighlightedRowUsesThatImage(t *testing.T) {
	m := New("compose.yaml", []string{"web"}).(Model)
	m = typeInto(m, "sonarr")

	results := []utils.ImageResult{
		{Name: "linuxserver/sonarr", Description: "A Sonarr container"},
		{Name: "linuxserver/radarr", Description: "A Radarr container"},
	}
	// Six keystrokes above bumped generation to 6; the result must carry
	// that exact generation or it is discarded as stale (D3).
	updated, _ := m.Update(cmds.SearchImagesMsg{Generation: 6, Results: results})
	m = updated.(Model)

	// The first row is highlighted by default (index 0) - no Down needed.
	updated, _ = m.Update(specialKey(tea.KeyEnter))
	m = updated.(Model)

	if m.step != stepConfirm {
		t.Fatalf("step = %v, want stepConfirm", m.step)
	}
	if got := m.image.Value(); got != "linuxserver/sonarr" {
		t.Errorf("image = %q, want the highlighted row's name", got)
	}
	if got := m.serviceName.Value(); got != "sonarr" {
		t.Errorf("serviceName = %q, want the derived name", got)
	}
}

func TestEnterWithNoResultsUsesTheTypedText(t *testing.T) {
	m := New("compose.yaml", []string{"web"}).(Model)
	m = typeInto(m, "ghcr.io/foo/bar:v2")
	// No results were fed in - the table is empty, so Enter must fall
	// through to the typed text (D3's escape hatch: the exact, common shape
	// of non-Hub reference docker search cannot serve).
	updated, _ := m.Update(specialKey(tea.KeyEnter))
	m = updated.(Model)

	if m.step != stepConfirm {
		t.Fatalf("step = %v, want stepConfirm", m.step)
	}
	if got := m.image.Value(); got != "ghcr.io/foo/bar:v2" {
		t.Errorf("image = %q, want the typed text verbatim", got)
	}
}

func TestEnterWithNothingTypedAndNoResultsDoesNothing(t *testing.T) {
	m := New("compose.yaml", []string{"web"}).(Model)

	updated, cmd := m.Update(specialKey(tea.KeyEnter))
	m = updated.(Model)

	if m.step != stepSearch {
		t.Fatalf("step = %v, want stepSearch - Enter with an empty query does nothing", m.step)
	}
	if cmd != nil {
		t.Errorf("Enter with an empty query produced a command: %v", cmd())
	}
}
