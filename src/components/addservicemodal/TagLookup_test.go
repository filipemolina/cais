package addservicemodal

import (
	"errors"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/filipemolina/stack-stitcher/src/cmds"
)

// The one test the plan requires for Phase 2B (D4's race guard): a
// TagLookupMsg arriving after the user has typed a different image value
// must be dropped, never applied over their edit.
func TestTagLookupNeverOverwritesAUserEdit(t *testing.T) {
	m := toConfirm(t, New("compose.yaml", []string{"web"}).(Model), "redis")

	// The user types a different image while the lookup is in flight.
	updated, _ := m.Update(specialKey(tea.KeyTab)) // focus the image field
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyPressMsg{Code: 'r', Text: "r"}) // image is now "rredis"
	m = updated.(Model)

	// The stale result arrives: Repo matches the pre-fill, but the field no
	// longer holds it.
	updated, _ = m.Update(cmds.TagLookupMsg{Repo: "redis", BestTag: "8.10.0"})
	m = updated.(Model)

	if got := m.image.Value(); got != "redisr" {
		t.Errorf("image = %q, want the user's typed value 'redisr' - the stale tag upgrade must be dropped", got)
	}
}

func TestTagLookupUpgradesTheUneditedField(t *testing.T) {
	m := toConfirm(t, New("compose.yaml", []string{"web"}).(Model), "redis")

	// No keystroke in between: the field still holds the pre-fill.
	updated, _ := m.Update(cmds.TagLookupMsg{Repo: "redis", BestTag: "8.10.0"})
	m = updated.(Model)

	if got := m.image.Value(); got != "redis:8.10.0" {
		t.Errorf("image = %q, want the tag-upgraded redis:8.10.0", got)
	}
}

func TestTagLookupErrorIsSilent(t *testing.T) {
	m := toConfirm(t, New("compose.yaml", []string{"web"}).(Model), "redis")

	updated, _ := m.Update(cmds.TagLookupMsg{Err: errors.New("hub unreachable")})
	m = updated.(Model)

	if got := m.image.Value(); got != "redis" {
		t.Errorf("image = %q, want the bare pre-fill untouched after a failed lookup", got)
	}
}
