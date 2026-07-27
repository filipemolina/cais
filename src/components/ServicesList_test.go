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

// Enter is an alias for space here too: same binding, same verb.
func TestEnterSelectsTheHighlightedService(t *testing.T) {
	list := drive(t, ServicesList(nil, 80, 24),
		cmds.SetServicesListMsg(servicesOf("api", "db", "web")),
		cmds.SetFocusMsg(1),
	)

	model, cmd := list.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	var selected string
	for _, msg := range messagesFrom(cmd) {
		if sel, ok := msg.(cmds.SetSelectedServiceMsg); ok {
			selected = types.ServiceConfig(sel).Name
		}
	}
	if want := "api"; selected != want {
		t.Errorf("enter selected %q, want %q", selected, want)
	}

	after, ok := model.(ServicesListModel)
	if !ok {
		t.Fatalf("expected a ServicesListModel, got %T", model)
	}
	if want := "api"; after.activeService != want {
		t.Errorf("active service after enter: got %q, want %q", after.activeService, want)
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
