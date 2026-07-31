package serviceslist

import (
	"testing"

	"charm.land/bubbles/v2/list"

	"github.com/filipemolina/stack-stitcher/src/apptypes"
	"github.com/filipemolina/stack-stitcher/src/cmds"

	tea "charm.land/bubbletea/v2"
	"github.com/compose-spec/compose-go/v2/types"
)

// messagesFrom flattens what a command produced, walking batches, so a test can
// assert on a message without caring how it got bundled.
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

func servicesOf(names ...string) []types.ServiceConfig {
	services := make([]types.ServiceConfig, 0, len(names))
	for _, name := range names {
		services = append(services, types.ServiceConfig{Name: name})
	}

	return services
}

// drive feeds messages through the list in order and hands back the model.
func drive(t *testing.T, model tea.Model, msgs ...tea.Msg) Model {
	t.Helper()

	for _, msg := range msgs {
		model, _ = model.Update(msg)
	}

	list, ok := model.(Model)
	if !ok {
		t.Fatalf("expected a Model, got %T", model)
	}

	return list
}

func TestActiveRowFollowsTheSelectedService(t *testing.T) {
	list := drive(t, New(nil, 80, 24),
		cmds.SetServicesListMsg(servicesOf("api", "db", "web")),
		cmds.SetSelectedServiceMsg(types.ServiceConfig{Name: "web"}),
	)

	if got, want := list.listDelegate.activeIndex, 2; got != want {
		t.Errorf("active row: got %d, want %d", got, want)
	}
}

// The list and the selection arrive as two messages batched together, and
// tea.Batch makes no promise about their order. Whichever lands first, the
// pair has to converge on the same row.
func TestActiveRowConvergesWhenSelectionArrivesFirst(t *testing.T) {
	list := drive(t, New(nil, 80, 24),
		cmds.SetSelectedServiceMsg(types.ServiceConfig{Name: "web"}),
		cmds.SetServicesListMsg(servicesOf("api", "db", "web")),
	)

	if got, want := list.listDelegate.activeIndex, 2; got != want {
		t.Errorf("active row: got %d, want %d", got, want)
	}
}

// The reason the name is stored rather than the row number: a reload that
// changes the list would otherwise leave the highlight on whatever service
// moved into the old row.
func TestActiveRowTracksTheServiceAcrossAReorder(t *testing.T) {
	model := New(nil, 80, 24)

	list := drive(t, model,
		cmds.SetServicesListMsg(servicesOf("api", "db", "web")),
		cmds.SetSelectedServiceMsg(types.ServiceConfig{Name: "web"}),
		// "cache" sorts ahead of "web", pushing it down a row.
		cmds.SetServicesListMsg(servicesOf("api", "cache", "db", "web")),
	)

	if got, want := list.listDelegate.activeIndex, 3; got != want {
		t.Errorf("active row after reload: got %d, want %d", got, want)
	}
}

func TestNoActiveRowWhenTheSelectedServiceIsGone(t *testing.T) {
	list := drive(t, New(nil, 80, 24),
		cmds.SetServicesListMsg(servicesOf("api", "db", "web")),
		cmds.SetSelectedServiceMsg(types.ServiceConfig{Name: "web"}),
		cmds.SetServicesListMsg(servicesOf("api", "db")),
	)

	if got := list.listDelegate.activeIndex; got != -1 {
		t.Errorf("active row after the service was removed: got %d, want -1", got)
	}
}

// Enter starts the selected service. Selection happens automatically on cursor
// movement, so we move the cursor first, then press enter.
func TestEnterStartsTheHighlightedService(t *testing.T) {
	list := drive(t, New(nil, 80, 24),
		cmds.SetServicesListMsg(servicesOf("api", "db", "web")),
		cmds.SetFocusMsg(1),
	)

	// Move the cursor down to trigger auto-select (cursor goes from 0 to 1).
	model, _ := list.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	moved, ok := model.(Model)
	if !ok {
		t.Fatalf("expected a Model, got %T", model)
	}

	// Verify auto-select happened (index 1 = db).
	if moved.activeService != "db" {
		t.Fatalf("auto-select did not fire: activeService = %q", moved.activeService)
	}

	// Now press enter to start the service.
	model, cmd := moved.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	var started bool
	for _, msg := range messagesFrom(cmd) {
		if dockerMsg, ok := msg.(cmds.RunDockerActionMsg); ok {
			if dockerMsg.Action == "start" && dockerMsg.Target == "db" && !dockerMsg.IsGroup {
				started = true
			}
		}
	}
	if !started {
		t.Errorf("enter did not start db, got %#v", messagesFrom(cmd))
	}
}

// Nothing is active until something is selected. The zero value would point
// at row 0 and render the first service as though the user had picked it.
func TestNoActiveRowBeforeAnySelection(t *testing.T) {
	list := drive(t, New(nil, 80, 24),
		cmds.SetServicesListMsg(servicesOf("api", "db", "web")),
	)

	if got := list.listDelegate.activeIndex; got != -1 {
		t.Errorf("active row before any selection: got %d, want -1", got)
	}
}

// Regression test: when memory usage stats are polled while a filter is
// active, the filtered items must remain visible. Previously,
// updateServiceStatuses called list.SetItems and discarded the returned
// tea.Cmd. SetItems clears filteredItems and returns a cmd that re-
// applies the filter; if that cmd is discarded, filteredItems stays nil
// and every filtered row disappears on the next render cycle.
// This test verifies that updateServiceStatuses returns the cmd from
// SetItems so the runtime can re-populate filteredItems correctly.
func TestFilterSurvivesStatsPolling(t *testing.T) {
	services := servicesOf("api", "cache", "db", "web")

	model := drive(t, New(services, 80, 24),
		cmds.SetFocusMsg(1),
	)

	// Type a filter that matches only "api" and "cache": the keystrokes
	// are handled by the inner list because it owns the keyboard while
	// filtering.
	filtered := drive(t, model,
		tea.KeyPressMsg{Code: '/', Text: "/"},
		tea.KeyPressMsg{Code: 'a', Text: "a"},
		tea.KeyPressMsg{Code: tea.KeyEnter},
	)

	if filtered.list.FilterState() != list.FilterApplied {
		t.Fatalf("precondition: filter state is %v, want FilterApplied", filtered.list.FilterState())
	}

	// Sanity-check that the filter works correctly before polling.
	initialVisible := len(filtered.list.VisibleItems())
	if initialVisible == 0 {
		t.Fatal("precondition: filter should show visible items")
	}

	// Stats polling produces GetContainerStatsMsg.
	statsMsg := cmds.GetContainerStatsMsg{
		Containers: []apptypes.DockerContainer{
			{Service: "api", State: "running"},
			{Service: "cache", State: "running"},
			{Service: "db", State: "running"},
			{Service: "web", State: "running"},
		},
	}

	// Update returns a cmd that (when executed by the Bubble Tea
	// runtime) re-applies the filter via FilterMatchesMsg, which
	// repopulates filteredItems so VisibleItems keeps working.
	_, cmd := filtered.Update(statsMsg)

	// The cmd must not be nil — that was the root cause of the bug.
	if cmd == nil {
		t.Fatal("updateServiceStatuses returned nil cmd; SetItems filter-re-application was discarded, leaving filteredItems nil")
	}

	// Run the cmd and walk the resulting msgs to confirm at least one
	// FilterMatchesMsg is produced. This is the msg the list's Update
	// handler uses to repopulate filteredItems.
	msgs := messagesFrom(cmd)
	var hasFilterMatches bool
	for _, m := range msgs {
		if _, ok := m.(list.FilterMatchesMsg); ok {
			hasFilterMatches = true
			break
		}
	}
	if !hasFilterMatches {
		t.Error("stats polling did not produce a FilterMatchesMsg; the SetItems cmd for re-applying the filter was not returned")
	}
}
