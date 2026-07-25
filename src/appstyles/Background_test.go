package appstyles

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

func TestFillBackgroundPaintsSpacesAfterAReset(t *testing.T) {
	pill := lipgloss.NewStyle().Background(Accent).Render("Details")

	// This is the shape JoinVertical produces: a styled run, then bare
	// padding out to the width of a wider sibling block.
	block := pill + strings.Repeat(" ", 20)

	if !HasBackgroundBleed(block) {
		t.Fatalf("precondition: expected raw block to bleed, got %q", block)
	}

	filled := FillBackground(BackgroundPanel, block)

	if HasBackgroundBleed(filled) {
		t.Errorf("FillBackground left unpainted spaces: %q", filled)
	}

	if got, want := lipgloss.Width(filled), lipgloss.Width(block); got != want {
		t.Errorf("FillBackground changed width: got %d, want %d", got, want)
	}
}

func TestFillBackgroundKeepsInnerBackgrounds(t *testing.T) {
	pill := lipgloss.NewStyle().Background(Accent).Render("Details")
	filled := FillBackground(BackgroundPanel, pill+"   ")

	accentSeq := backgroundSeq(Accent)
	if !strings.Contains(filled, accentSeq) {
		t.Errorf("FillBackground dropped the inner accent background %q from %q", accentSeq, filled)
	}
}

func TestFillBackgroundPreservesPlainText(t *testing.T) {
	filled := FillBackground(BackgroundPanel, "Name: web\nImage: nginx")

	if got := ansi.Strip(filled); got != "Name: web\nImage: nginx" {
		t.Errorf("FillBackground altered the text: got %q", got)
	}

	if got, want := strings.Count(filled, "\n"), 1; got != want {
		t.Errorf("FillBackground changed the line count: got %d newlines, want %d", got, want)
	}
}

func TestFillBackgroundNilAndEmpty(t *testing.T) {
	if got := FillBackground(nil, "hello"); got != "hello" {
		t.Errorf("nil background should be a no-op, got %q", got)
	}

	if got := FillBackground(BackgroundPanel, ""); got != "" {
		t.Errorf("empty block should stay empty, got %q", got)
	}
}
