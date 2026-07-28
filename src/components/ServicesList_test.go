package components

import (
	"testing"

	"github.com/filipemolina/stack-stitcher/src/cmds"

	tea "charm.land/bubbletea/v2"
	"github.com/compose-spec/compose-go/v2/types"
)

func servicesOf(names ...string) []types.ServiceConfig {
	services := make([]types.ServiceConfig, 0, len(names))
	for _, name := range names {
		services = append(services, types.ServiceConfig{Name: name})
	}

	return services
}

// drive feeds messages through the list in order and hands back the model.
func drive(t *testing.T, model tea.Model, msgs ...tea.Msg) ServicesListModel {
	t.Helper()

	for _, msg := range msgs {
		model, _ = model.Update(msg)
	}

	list, ok := model.(ServicesListModel)
	if !ok {
		t.Fatalf("expected a ServicesListModel, got %T", model)
	}

	return list
}

func TestActiveRowFollowsTheSelectedService(t *testing.T) {
	list := drive(t, ServicesList(nil, 80, 24),
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
	list := drive(t, ServicesList(nil, 80, 24),
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
	model := ServicesList(nil, 80, 24)

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
	list := drive(t, ServicesList(nil, 80, 24),
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
	list := drive(t, ServicesList(nil, 80, 24),
		cmds.SetServicesListMsg(servicesOf("api", "db", "web")),
		cmds.SetFocusMsg(1),
	)

	// Move the cursor down to trigger auto-select (cursor goes from 0 to 1).
	model, _ := list.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	moved, ok := model.(ServicesListModel)
	if !ok {
		t.Fatalf("expected a ServicesListModel, got %T", model)
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
	list := drive(t, ServicesList(nil, 80, 24),
		cmds.SetServicesListMsg(servicesOf("api", "db", "web")),
	)

	if got := list.listDelegate.activeIndex; got != -1 {
		t.Errorf("active row before any selection: got %d, want -1", got)
	}
}
