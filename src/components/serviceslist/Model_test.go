package serviceslist

import (
	"errors"
	"strings"
	"testing"

	"charm.land/bubbles/v2/list"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/filipemolina/cais/src/appstyles"
	"github.com/filipemolina/cais/src/components/chrome"

	"github.com/filipemolina/cais/src/apptypes"
	"github.com/filipemolina/cais/src/cmds"

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

// d opens the delete confirm for the highlighted service - the same key,
// same "delete the highlighted thing" meaning as the groups list's d.
// Deletion itself goes through AppModel and a confirm modal (see
// cmds.OpenDeleteServiceModal and utils.DeleteService), so this only pins
// that the list asks for it.
func TestDeleteOpensTheDeleteConfirmForTheHighlightedService(t *testing.T) {
	list := drive(t, New(nil, 80, 24),
		cmds.SetServicesListMsg(servicesOf("api", "db", "web")),
	)

	model, _ := list.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	moved, ok := model.(Model)
	if !ok {
		t.Fatalf("expected a Model, got %T", model)
	}
	if moved.activeService != "db" {
		t.Fatalf("auto-select did not fire: activeService = %q", moved.activeService)
	}

	_, cmd := moved.Update(tea.KeyPressMsg{Code: 'd', Text: "d"})

	var opened bool
	for _, msg := range messagesFrom(cmd) {
		if openMsg, ok := msg.(cmds.OpenDeleteServiceModalMsg); ok && string(openMsg) == "db" {
			opened = true
		}
	}
	if !opened {
		t.Errorf("d did not open the delete confirm for db, got %#v", messagesFrom(cmd))
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

	model := drive(t, New(services, 80, 24))

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

// settle feeds messages through the list and then keeps feeding back whatever
// the resulting commands produced, the way the Bubble Tea runtime does.
//
// `drive` above throws commands away, which is enough for most of these tests
// but hides everything downstream of list.FilterMatchesMsg - the message that
// actually populates the filtered rows. Anything asserting on behaviour under
// a filter has to go through here instead. Selections are pulled out of the
// loop and returned rather than fed back, so a test can see what the list
// asked AppModel to select.
func settle(t *testing.T, model Model, msgs ...tea.Msg) (Model, []types.ServiceConfig) {
	t.Helper()

	var selected []types.ServiceConfig

	queue := append([]tea.Msg{}, msgs...)
	for range 200 {
		if len(queue) == 0 {
			break
		}

		msg := queue[0]
		queue = queue[1:]

		next, cmd := model.Update(msg)
		updated, ok := next.(Model)
		if !ok {
			t.Fatalf("expected a Model, got %T", next)
		}
		model = updated

		for _, produced := range messagesFrom(cmd) {
			if produced == nil {
				continue
			}
			if pick, ok := produced.(cmds.SetSelectedServiceMsg); ok {
				selected = append(selected, types.ServiceConfig(pick))
				continue
			}
			queue = append(queue, produced)
		}
	}

	return model, selected
}

// visibleNames is what the list is actually showing, in order.
func visibleNames(m Model) []string {
	names := make([]string, 0, len(m.list.VisibleItems()))
	for _, item := range m.list.VisibleItems() {
		if service, ok := item.(apptypes.ServiceListItem); ok {
			names = append(names, service.Service.Name)
		}
	}

	return names
}

// applyFilter types term into the list's filter and accepts it.
func applyFilter(t *testing.T, m Model, term string) (Model, []types.ServiceConfig) {
	t.Helper()

	msgs := []tea.Msg{tea.KeyPressMsg{Code: '/', Text: "/"}}
	for _, r := range term {
		msgs = append(msgs, tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	msgs = append(msgs, tea.KeyPressMsg{Code: tea.KeyEnter})

	return settle(t, m, msgs...)
}

// Regression test: applying a filter while the cursor is already on the first
// row left the details panel showing a service the filter had just hidden.
//
// The selection used to be published only when list.Index() changed, and
// accepting a filter sends the cursor to the top - so a cursor already at the
// top kept index 0 while row 0 became a different service entirely, and no
// SetSelectedServiceMsg was ever sent.
func TestFilterSelectsTheRowItPutsUnderTheCursor(t *testing.T) {
	model := drive(t, New(servicesOf("api", "cache", "db", "web"), 80, 24),
		cmds.SetSelectedServiceMsg(servicesOf("api")[0]),
	)

	if got := model.list.Index(); got != 0 {
		t.Fatalf("precondition: cursor is at %d, want 0", got)
	}

	filtered, selected := applyFilter(t, model, "web")

	if got := visibleNames(filtered); len(got) == 0 || got[0] != "web" {
		t.Fatalf("precondition: filtered rows are %v, want web first", got)
	}
	if got := filtered.activeService; got != "web" {
		t.Errorf("activeService = %q, want %q", got, "web")
	}
	if len(selected) == 0 {
		t.Fatal("no SetSelectedServiceMsg was published; the details panel keeps the pre-filter service")
	}
	if got := selected[len(selected)-1].Name; got != "web" {
		t.Errorf("last published selection = %q, want %q", got, "web")
	}
}

// Regression test: the delegate's activeIndex is an index into the rows being
// drawn, and list.populatedView numbers those against VisibleItems. Deriving
// it from the full Items slice put the highlight on the wrong row under a
// filter, or past the end of the visible rows and so on no row at all.
func TestActiveIndexIsRelativeToTheVisibleRows(t *testing.T) {
	model := drive(t, New(servicesOf("api", "cache", "db", "web", "worker"), 80, 24),
		cmds.SetSelectedServiceMsg(servicesOf("api")[0]),
	)

	filtered, _ := applyFilter(t, model, "work")

	visible := visibleNames(filtered)
	if len(visible) == 0 {
		t.Fatal("precondition: the filter hid everything")
	}

	active := filtered.listDelegate.activeIndex
	if active < 0 || active >= len(visible) {
		t.Fatalf("activeIndex = %d, outside the %d visible rows %v",
			active, len(visible), visible)
	}
	if got := visible[active]; got != filtered.activeService {
		t.Errorf("activeIndex %d points at %q, but the selected service is %q",
			active, got, filtered.activeService)
	}
}

// Regression test: clearing the filter puts every row back, and the row under
// the cursor is once again what the details panel must be showing.
func TestClearingTheFilterResyncsTheSelection(t *testing.T) {
	model := drive(t, New(servicesOf("api", "cache", "db", "web", "worker"), 80, 24),
		cmds.SetSelectedServiceMsg(servicesOf("api")[0]),
	)

	filtered, _ := applyFilter(t, model, "work")
	if filtered.activeService != "worker" {
		t.Fatalf("precondition: activeService = %q, want worker", filtered.activeService)
	}

	cleared, selected := settle(t, filtered, tea.KeyPressMsg{Code: tea.KeyEsc})

	if got := cleared.list.FilterState(); got != list.Unfiltered {
		t.Fatalf("filter state = %v, want Unfiltered", got)
	}

	visible := visibleNames(cleared)
	under := visible[cleared.list.Index()]

	if cleared.activeService != under {
		t.Errorf("activeService = %q, but the cursor is on %q", cleared.activeService, under)
	}
	if len(selected) > 0 && selected[len(selected)-1].Name != under {
		t.Errorf("last published selection = %q, but the cursor is on %q",
			selected[len(selected)-1].Name, under)
	}

	active := cleared.listDelegate.activeIndex
	if active < 0 || active >= len(visible) || visible[active] != cleared.activeService {
		t.Errorf("activeIndex = %d does not point at %q in %v",
			active, cleared.activeService, visible)
	}
}

// A selection set by AppModel (after a config reload, say) must not be
// overridden by the row the cursor happens to be resting on. The cursor did
// not move, so the list has nothing new to report.
func TestExternalSelectionIsNotEchoedBack(t *testing.T) {
	model := drive(t, New(servicesOf("api", "cache", "db", "web"), 80, 24))

	updated, selected := settle(t, model,
		cmds.SetSelectedServiceMsg(servicesOf("db")[0]),
	)

	if got := updated.activeService; got != "db" {
		t.Errorf("activeService = %q, want db", got)
	}
	if len(selected) != 0 {
		t.Errorf("published %d selections, want none: AppModel's choice was echoed back", len(selected))
	}
}

// A config reload replaces every item. The list must stay silent about it:
// AppModel restores the selection after a reload, and a list that published
// whatever landed on row 0 would race that restore.
//
// This works because Update reads the cursor after its own switch has already
// applied SetServicesListMsg, so the rebuild is not something the identity
// check can see. Pinned here because moving that read earlier would be a
// subtle and hard-to-trace regression.
func TestReloadingTheListPublishesNothing(t *testing.T) {
	model := drive(t, New(servicesOf("api", "cache", "db"), 80, 24),
		cmds.SetSelectedServiceMsg(servicesOf("db")[0]),
	)

	updated, selected := settle(t, model,
		cmds.SetServicesListMsg(servicesOf("api", "cache", "db", "web")),
	)

	if len(selected) != 0 {
		t.Errorf("reload published %d selections (%v), want none", len(selected), selected)
	}
	if got := updated.activeService; got != "db" {
		t.Errorf("activeService = %q after a reload, want db", got)
	}
}

// refreshMsg is what the five-second container poll delivers.
func refreshMsg(names ...string) cmds.GetRunningContainersMsg {
	containers := make([]apptypes.DockerContainer, 0, len(names))
	for _, name := range names {
		containers = append(containers, apptypes.DockerContainer{Service: name, State: "running"})
	}

	return cmds.GetRunningContainersMsg{Containers: containers}
}

// Regression test: the container refresh must not walk the cursor back to the
// first match of a standing filter.
//
// list.SetItems blanks the filtered rows and re-applies the filter a message
// later; while they are blank the list clamps its cursor into an empty range,
// so every five seconds the selection jumped back to the top of the filter.
func TestARefreshLeavesTheFilteredCursorAlone(t *testing.T) {
	model := drive(t, New(servicesOf("alpha", "bravo", "webproxy", "worker"), 80, 24),
		cmds.SetSelectedServiceMsg(servicesOf("alpha")[0]),
	)

	filtered, _ := applyFilter(t, model, "w")
	moved, _ := settle(t, filtered, tea.KeyPressMsg{Code: tea.KeyDown})

	wantIndex, wantService := moved.list.Index(), moved.activeService
	if wantIndex == 0 {
		t.Fatalf("precondition: the cursor never left the first row (visible %v)", visibleNames(moved))
	}

	// Three ticks, because the flag that survives the blank window has to be
	// cleared each time or only the first refresh would be safe.
	refreshed := moved
	for i := range 3 {
		var selected []types.ServiceConfig
		refreshed, selected = settle(t, refreshed, refreshMsg("worker", "webproxy"))

		if got := refreshed.list.Index(); got != wantIndex {
			t.Fatalf("refresh %d moved the cursor to %d, want %d", i+1, got, wantIndex)
		}
		if got := refreshed.activeService; got != wantService {
			t.Fatalf("refresh %d changed the selection to %q, want %q", i+1, got, wantService)
		}
		if len(selected) != 0 {
			t.Fatalf("refresh %d published %d selections, want none", i+1, len(selected))
		}
	}

	// The rows are still filtered, and the highlight still points at the
	// selected row within them.
	visible := visibleNames(refreshed)
	active := refreshed.listDelegate.activeIndex
	if active < 0 || active >= len(visible) || visible[active] != wantService {
		t.Errorf("activeIndex = %d does not point at %q in %v", active, wantService, visible)
	}
}

// The status bar is the only thing on screen naming the filter once it has
// been accepted, so it is shown for exactly as long as a filter is standing.
func TestTheStatusBarNamesTheStandingFilter(t *testing.T) {
	model := drive(t, New(servicesOf("alpha", "bravo", "webproxy", "worker"), 80, 24),
		cmds.SetSelectedServiceMsg(servicesOf("alpha")[0]),
	)

	if got := ansi.Strip(model.View().Content); strings.Contains(got, "filtered") {
		t.Errorf("an unfiltered list is showing a status bar:\n%s", got)
	}

	filtered, _ := applyFilter(t, model, "w")

	view := ansi.Strip(filtered.View().Content)
	for _, want := range []string{`“w”`, "2 services", "2 filtered"} {
		if !strings.Contains(view, want) {
			t.Errorf("the status bar does not mention %q:\n%s", want, view)
		}
	}

	cleared, _ := settle(t, filtered, tea.KeyPressMsg{Code: tea.KeyEsc})

	if got := ansi.Strip(cleared.View().Content); strings.Contains(got, "filtered") {
		t.Errorf("the status bar outlived the filter:\n%s", got)
	}
}

// Every service row carries a status dot, in the same place the groups list
// puts its group dot. Unlike a group's, a service's is never hidden: a
// service either runs or it does not, and an absent dot would read as an
// absent answer rather than as "stopped".
func TestEveryServiceRowCarriesAStatusDot(t *testing.T) {
	model := drive(t, New(servicesOf("alpha", "bravo"), 40, 24),
		cmds.SetSelectedServiceMsg(servicesOf("alpha")[0]),
		cmds.GetRunningContainersMsg{Containers: []apptypes.DockerContainer{
			{Service: "alpha", State: "running"},
			{Service: "bravo", State: "exited"},
		}},
	)

	view := ansi.Strip(model.View().Content)

	var titleRows []string
	for _, line := range strings.Split(view, "\n") {
		if strings.Contains(line, "alpha") || strings.Contains(line, "bravo") {
			titleRows = append(titleRows, strings.TrimRight(line, " "))
		}
	}
	if len(titleRows) != 2 {
		t.Fatalf("expected two service rows, got %d:\n%s", len(titleRows), view)
	}

	for _, row := range titleRows {
		if !strings.HasSuffix(row, "●") {
			t.Errorf("row %q does not end in a status dot", row)
		}
	}
}

// The dot's colour is the whole point of it: running and stopped must not
// render the same. Compared on the styled output, since ansi.Strip is exactly
// what would hide a mistake here.
func TestTheStatusDotColoursRunningAndStoppedDifferently(t *testing.T) {
	rows := func(state string) string {
		m := drive(t, New(servicesOf("alpha"), 40, 24),
			cmds.SetSelectedServiceMsg(servicesOf("alpha")[0]),
			cmds.GetRunningContainersMsg{Containers: []apptypes.DockerContainer{
				{Service: "alpha", State: state},
			}},
		)

		return m.View().Content
	}

	running, stopped := rows("running"), rows("exited")

	if running == stopped {
		t.Error("a running service and a stopped one render identically")
	}
	if ansi.Strip(running) != ansi.Strip(stopped) {
		t.Error("running and stopped differ in more than colour; the dot glyph should be the same for both")
	}
}

// A name too long for the row yields the dot's column rather than pushing it
// off: the dot is the fact the row exists to carry.
func TestALongServiceNameYieldsToTheDot(t *testing.T) {
	model := drive(t, New(servicesOf("a-very-long-service-name-that-will-not-fit"), 40, 24))

	view := ansi.Strip(model.View().Content)

	var row string
	for _, line := range strings.Split(view, "\n") {
		if strings.Contains(line, "a-very-long") {
			row = strings.TrimRight(line, " ")
			break
		}
	}
	if row == "" {
		t.Fatalf("no service row rendered:\n%s", view)
	}
	if !strings.HasSuffix(row, "●") {
		t.Errorf("the long name pushed the dot off the row: %q", row)
	}
	if !strings.Contains(row, "…") {
		t.Errorf("the long name was not truncated: %q", row)
	}
}

// The stopped dot uses the same theme token as the STOPPED pill, so a row and
// the panel it opens agree on what stopped looks like - and so it follows the
// user's theme. Pinned because the obvious token to reach for is
// StatusStopped, which is the grey this was changed away from.
func TestTheStoppedDotMatchesTheStoppedPill(t *testing.T) {
	t.Cleanup(func() { appstyles.SetTheme(appstyles.DefaultTheme) })

	for theme := range appstyles.Themes {
		if !appstyles.SetTheme(theme) {
			t.Fatalf("theme %q is in the registry but SetTheme rejected it", theme)
		}

		model := drive(t, New(servicesOf("alpha"), 40, 24),
			cmds.GetRunningContainersMsg{Containers: []apptypes.DockerContainer{
				{Service: "alpha", State: "exited"},
			}},
		)

		want := lipgloss.NewStyle().
			Foreground(appstyles.Active.StatusError).
			Background(chrome.ListRowBg(false)).
			Render("●")

		if got := model.View().Content; !strings.Contains(got, want) {
			t.Errorf("theme %q: the stopped dot is not the STOPPED pill's colour", theme)
		}
	}
}

// mid delivers one message and deliberately does NOT drain the commands it
// produced, which is the frame the runtime would render before the follow-up
// messages arrive. Anything that goes blank for a cycle shows up here and
// nowhere else.
func mid(t *testing.T, m Model, msg tea.Msg) Model {
	t.Helper()

	next, _ := m.Update(msg)
	updated, ok := next.(Model)
	if !ok {
		t.Fatalf("expected a Model, got %T", next)
	}

	return updated
}

// Regression test: a refresh must not empty a filtered list, even for the one
// frame between the rebuild and the filter being re-applied.
//
// updateServiceStatuses used to replace every row with list.SetItems, which
// nils filteredItems and defers the rebuild to a command. The rows were
// therefore gone for a whole message cycle - long enough to be rendered - so
// a filtered list flashed "No services." every five seconds.
func TestARefreshDoesNotBlankAFilteredList(t *testing.T) {
	model := drive(t, New(servicesOf("alpha", "bravo", "webproxy", "worker"), 80, 24),
		cmds.SetSelectedServiceMsg(servicesOf("alpha")[0]),
	)

	filtered, _ := applyFilter(t, model, "w")
	before := visibleNames(filtered)
	if len(before) == 0 {
		t.Fatal("precondition: the filter hid everything")
	}

	frame := mid(t, filtered, refreshMsg("worker", "webproxy"))

	if got := visibleNames(frame); len(got) != len(before) {
		t.Errorf("mid-refresh the list shows %v, want %v", got, before)
	}
	if got := ansi.Strip(frame.View().Content); strings.Contains(got, "No services") {
		t.Errorf("mid-refresh the list rendered its empty state:\n%s", got)
	}
}

// The in-place refresh still has to deliver the new numbers: the rows the
// filter is showing are copies, so a status change only reaches the screen
// once the re-filter lands. A refresh that skipped the re-filter would be
// silent rather than blank, which is worse.
func TestARefreshUnderAFilterStillUpdatesTheRows(t *testing.T) {
	model := drive(t, New(servicesOf("alpha", "webproxy", "worker"), 80, 24),
		cmds.SetSelectedServiceMsg(servicesOf("alpha")[0]),
	)

	filtered, _ := applyFilter(t, model, "worker")
	for _, item := range filtered.list.VisibleItems() {
		if service, ok := item.(apptypes.ServiceListItem); ok && service.Status == "running" {
			t.Fatal("precondition: worker is already running")
		}
	}

	refreshed, _ := settle(t, filtered, refreshMsg("worker"))

	var found bool
	for _, item := range refreshed.list.VisibleItems() {
		service, ok := item.(apptypes.ServiceListItem)
		if !ok || service.Service.Name != "worker" {
			continue
		}
		found = true
		if service.Status != "running" {
			t.Errorf("worker's visible row still reads %q after the refresh", service.Status)
		}
	}
	if !found {
		t.Fatalf("worker is not among the visible rows: %v", visibleNames(refreshed))
	}
}

// Regression test: a service is not stopped just because docker has not
// answered yet.
//
// containerStatus returned "" both for "this service has no container" and
// for "nobody has looked yet", and the row painted "" the same red as
// stopped - so arriving on the Services page showed a page of red dots for
// the second before the first poll landed, and kept showing them for good
// when docker was unreachable.
func TestARowIsNotStoppedUntilDockerHasAnswered(t *testing.T) {
	model := drive(t, New(servicesOf("alpha", "bravo"), 40, 24))

	for _, item := range model.list.Items() {
		service, ok := item.(apptypes.ServiceListItem)
		if !ok {
			continue
		}
		if service.Status != "" {
			t.Errorf("%s reads %q before any poll, want unknown", service.Service.Name, service.Status)
		}
	}

	unknown := model.View().Content

	// Docker answers: alpha is up, bravo has no container at all. Both are
	// now known, and bravo is genuinely stopped.
	answered := drive(t, model,
		cmds.GetRunningContainersMsg{Containers: []apptypes.DockerContainer{
			{Service: "alpha", State: "running"},
		}},
	)

	for _, item := range answered.list.Items() {
		service, ok := item.(apptypes.ServiceListItem)
		if !ok {
			continue
		}
		want := "stopped"
		if service.Service.Name == "alpha" {
			want = "running"
		}
		if service.Status != want {
			t.Errorf("%s reads %q after the poll, want %q", service.Service.Name, service.Status, want)
		}
	}

	if answered.View().Content == unknown {
		t.Error("the rows render identically before and after docker answered")
	}
}

// A failed poll leaves the answer unknown rather than asserting everything is
// stopped: cais cannot see docker, which is not the same as docker being idle.
func TestAFailedPollLeavesTheRowsUnknown(t *testing.T) {
	model := drive(t, New(servicesOf("alpha"), 40, 24),
		cmds.GetRunningContainersMsg{Err: errors.New("docker daemon is not running")},
	)

	for _, item := range model.list.Items() {
		if service, ok := item.(apptypes.ServiceListItem); ok && service.Status != "" {
			t.Errorf("%s reads %q after a failed poll, want unknown", service.Service.Name, service.Status)
		}
	}
}

// The three states must not collapse into two on screen: unknown, running and
// stopped each get their own colour.
func TestTheStatusDotHasThreeDistinctStates(t *testing.T) {
	row := func(msgs ...tea.Msg) string {
		return drive(t, New(servicesOf("alpha"), 40, 24), msgs...).View().Content
	}

	unknown := row()
	running := row(cmds.GetRunningContainersMsg{Containers: []apptypes.DockerContainer{
		{Service: "alpha", State: "running"},
	}})
	stopped := row(cmds.GetRunningContainersMsg{Containers: []apptypes.DockerContainer{
		{Service: "alpha", State: "exited"},
	}})

	if unknown == running || unknown == stopped || running == stopped {
		t.Error("unknown, running and stopped do not all render differently")
	}
	if ansi.Strip(unknown) != ansi.Strip(running) || ansi.Strip(running) != ansi.Strip(stopped) {
		t.Error("the three states differ in more than colour; the glyph should be the same for all")
	}
}
