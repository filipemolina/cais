package model

import (
	"cmp"
	"errors"
	"fmt"
	"maps"
	"path/filepath"
	"slices"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/compose-spec/compose-go/v2/types"
	"github.com/filipemolina/stack-stitcher/src/apptypes"
	"github.com/filipemolina/stack-stitcher/src/cmds"
	"github.com/filipemolina/stack-stitcher/src/components"
	"github.com/filipemolina/stack-stitcher/src/components/aboutmodal"
	"github.com/filipemolina/stack-stitcher/src/components/chrome"
	"github.com/filipemolina/stack-stitcher/src/components/confirmmodal"
	"github.com/filipemolina/stack-stitcher/src/components/errormodal"
	"github.com/filipemolina/stack-stitcher/src/components/helpoverlay"
	"github.com/filipemolina/stack-stitcher/src/components/logsmodal"
	"github.com/filipemolina/stack-stitcher/src/components/servicechecklistmodal"
	"github.com/filipemolina/stack-stitcher/src/constants"
	"github.com/filipemolina/stack-stitcher/src/keys"
	"github.com/filipemolina/stack-stitcher/src/utils"
)

// calculateBodyLayout returns the exact box each body panel must render
// into: the row count left after the nav, keybinding bar, and optional error
// banner, split across the two panels so that
// left + BODY_GUTTER_WIDTH + right == the terminal width.
//
// The left panel gets LEFT_PANEL_WIDTH of the row (after the gutter is taken
// out) and the right panel gets whatever is left, so rounding can never make
// the two panels overflow or leave a ragged column. Both panels are held at
// MIN_PANEL_WIDTH where the terminal allows it; below that the row is split
// evenly and the panels clip their own content.
func (m AppModel) calculateBodyLayout() cmds.SetBodyLayoutMsg {
	menuHeight := lipgloss.Height(m.components.MainMenu.View().Content)
	keybarHeight := lipgloss.Height(m.components.KeybindingBar.View().Content)

	errorBanner := 0
	if m.lastError != "" {
		errorBanner = 1
	}

	height := m.config.terminalHeight - menuHeight - keybarHeight - errorBanner
	if height < 0 {
		height = 0
	}

	available := m.config.terminalWidth - constants.BODY_GUTTER_WIDTH
	if available < 0 {
		available = 0
	}

	var left int
	switch {
	case available < 2*constants.MIN_PANEL_WIDTH:
		left = available / 2
	default:
		left = int(float32(available) * constants.LEFT_PANEL_WIDTH)
		left = max(left, constants.MIN_PANEL_WIDTH)
		left = min(left, available-constants.MIN_PANEL_WIDTH)
	}

	return cmds.SetBodyLayoutMsg{
		LeftWidth:  left,
		RightWidth: available - left,
		Height:     height,
	}
}

// broadcastBodyLayout returns a command that sends the current body layout
// to the active page's components.
func (m AppModel) broadcastBodyLayout() tea.Cmd {
	return cmds.SetBodyLayout(
		m.config.bodyLayout.LeftWidth,
		m.config.bodyLayout.RightWidth,
		m.config.bodyLayout.Height,
	)
}

// rebroadcastBodyLayoutIfChanged recalculates the body layout and, if it
// differs from the stored value, updates the stored value and returns a
// command to broadcast it. It is used when the error banner appears or
// disappears, because the banner consumes one row.
func (m *AppModel) rebroadcastBodyLayoutIfChanged() tea.Cmd {
	newLayout := m.calculateBodyLayout()
	if newLayout == m.config.bodyLayout {
		return nil
	}
	m.config.bodyLayout = newLayout
	return m.broadcastBodyLayout()
}

// reportForegroundError puts an error the user's own action produced in front
// of them: a modal when the screen is free, the error banner when a modal
// already owns it.
//
// The fallback is the whole point. Guarding the modal on activeModal == nil is
// right - a modal the user opened deliberately is not something a late-
// arriving error gets to close out from under them - but on its own it threw
// the error away. Pressing s and then ? before the action failed reported
// nothing at all: no modal, no banner, a docker action that silently did
// nothing. The banner is the quieter channel, which is what an error that has
// to wait its turn should get.
//
// Returns the layout command the banner needs, since it costs a row; it is nil
// on the modal path, and appending a nil command is harmless.
func (m *AppModel) reportForegroundError(message string) tea.Cmd {
	if m.activeModal == nil {
		// The banner is untouched, so whoever owned it still does - including
		// a background poll whose error is still showing there. Clearing
		// lastErrorFromPoll here would strand that error, since a recovered
		// poll only clears what it put up itself.
		m.activeModal = errormodal.New(message, m.config.terminalWidth)
		return nil
	}

	// Taking the banner over means taking ownership of it: this error is the
	// user's action failing, not the poll's, so a later successful poll must
	// not clear it.
	m.lastError = message
	m.lastErrorFromPoll = false
	return m.rebroadcastBodyLayoutIfChanged()
}

// configSyncCmds re-derives the ordered services/groups lists from the
// loaded compose project and broadcasts them. Messages only reach the
// currently active page's components (see UpdateInnerComponent), so this
// needs to run both right after the config loads AND whenever the active
// page changes - otherwise a page that wasn't active at load time (e.g.
// Services, since Home is active first) would never receive its services.
func (m AppModel) configSyncCmds() []tea.Cmd {
	if m.config.configProject == nil {
		return nil
	}

	var syncCmds []tea.Cmd

	length := len(m.config.configProject.Services) + len(m.config.configProject.DisabledServices)
	orderedServices := make([]types.ServiceConfig, 0, length)

	orderedServicesMap := m.config.configProject.Services
	maps.Copy(orderedServicesMap, m.config.configProject.DisabledServices)

	for _, service := range orderedServicesMap {
		orderedServices = append(orderedServices, service)
	}

	slices.SortFunc(orderedServices, func(a, b types.ServiceConfig) int {
		return cmp.Compare(a.Name, b.Name)
	})

	// Re-select what the user had selected before the reload. A missing name
	// (nothing selected yet, or it was removed or renamed outside the app)
	// gives -1, and falling back to the first entry is the only answer left.
	syncCmds = append(syncCmds, cmds.SetServicesList(orderedServices))
	if len(orderedServices) > 0 {
		index := slices.IndexFunc(orderedServices, func(service types.ServiceConfig) bool {
			return service.Name == m.selection.serviceName
		})
		syncCmds = append(syncCmds, cmds.SetSelectedService(orderedServices[max(0, index)]))
	}

	orderedGroups := m.allGroupNames()

	syncCmds = append(syncCmds, cmds.SetGroupsList(orderedGroups))
	if len(orderedGroups) > 0 {
		index := slices.Index(orderedGroups, m.selection.groupName)
		syncCmds = append(syncCmds, cmds.SetSelectedGroup(orderedGroups[max(0, index)]))
	}

	return syncCmds
}

// shouldPollContainers reports whether the background tick should shell out
// to `docker compose ps`. It skips while a modal owns the screen (least
// surprising, and avoids racing the bootstrap/confirm flows), while an
// external editor holds the terminal (the app is suspended, so a poll would
// only queue work for the resume), and while no compose project is loaded -
// without a compose file the poll can only fail, and its error would clobber
// the bootstrap message in the banner.
func (m AppModel) shouldPollContainers() bool {
	return m.activeModal == nil && !m.externalEditorOpen && m.config.configProject != nil
}

// keyboardOwner is implemented by a component that takes every keystroke for
// itself while it is in some state - today only a list with a filter being typed
// into it, where n, d and q are letters rather than commands.
//
// It is an interface rather than a message the component broadcasts because the
// answer has to be true on the very keystroke that changes it: the / that starts
// a filter and the letters typed after it arrive before any command the list
// returned has been run.
type keyboardOwner interface {
	OwnsKeyboard() bool
}

// keyboardOwned reports whether a component on the active page has taken over
// the keyboard, in which case AppModel keeps its hands off the letter keys the
// same way it does while a modal is open.
func (m AppModel) keyboardOwned() bool {
	for _, component := range m.pages[m.activePage] {
		if owner, ok := component.(keyboardOwner); ok && owner.OwnsKeyboard() {
			return true
		}
	}

	return false
}

// escKeeper is implemented by a component that needs esc for itself without
// owning the whole keyboard: a focused list holding an applied filter, where
// esc alone clears it. Same shape as keyboardOwner - the answer has to be
// right on the keystroke that asks.
type escKeeper interface {
	KeepsEsc() bool
}

// filterStater is implemented by the body lists, so the help overlay can
// snapshot the filter state for its availability dimming.
type filterStater interface {
	FilterState() list.FilterState
}

// helpContext snapshots what the help overlay needs to dim the keys that do
// nothing right now. A modal freezes the screen it opened from - panels see
// no keys while one is up - so the snapshot cannot go stale.
func (m AppModel) helpContext() keys.Context {
	ctx := keys.Context{
		Page:          m.activePage,
		Focused:       m.focusedComponent,
		Editing:       m.inlineEditing,
		PendingAction: m.pendingAction != nil,
	}

	switch m.activePage {
	case "Home":
		ctx.ListEmpty = len(m.allGroupNames()) == 0
		ctx.Selected = m.selection.groupName != ""
	case "Services":
		ctx.ListEmpty = m.config.configProject == nil || len(m.config.configProject.Services) == 0
		ctx.Selected = m.selection.serviceName != ""
	}

	for _, component := range m.pages[m.activePage] {
		if list, ok := component.(filterStater); ok {
			ctx.Filter = list.FilterState()
			break
		}
	}

	return ctx
}

// escKept reports whether esc belongs to a component on the active page.
// AppModel's "back" yields to that: moving focus away from a filtered list
// would strand the filter on a panel that no longer answers esc.
func (m AppModel) escKept() bool {
	for _, component := range m.pages[m.activePage] {
		if keeper, ok := component.(escKeeper); ok && keeper.KeepsEsc() {
			return true
		}
	}

	return false
}

// pageForNavKey returns the page a global navigation key jumps to, or "" if
// the key is not one. Three forms, one destination:
//
//   - a digit: 1 is the first tab in apptypes.PageTitles, and so on. The
//     primary scheme, and the one the nav renders on each tab.
//   - [ and ]: step through the tabs in order, wrapping around.
//   - alt+<letter>: the original scheme, kept as an alias for the terminals
//     that send Option as Alt.
//
// Everything here runs inside Update's keyboardOwned guard, so while a filter
// is being typed a digit is a letter; and after the modal check, so typing in
// a text field can never navigate away.
func (m AppModel) pageForNavKey(msg tea.KeyPressMsg) string {
	if page := pageForDigit(msg); page != "" {
		return page
	}
	if page := m.pageForStep(msg); page != "" {
		return page
	}
	return pageForChord(msg)
}

// pageForDigit returns the page a digit key jumps to: 1 is the first page in
// apptypes.PageTitles, and so on. "" if the key is not a digit, or is a digit
// with no page behind it.
//
// Matched on the key's code with no modifiers held, so shifted digits (which
// arrive as the punctuation on that key) and ctrl+1 are left alone.
func pageForDigit(msg tea.KeyPressMsg) string {
	key := msg.Key()
	if key.Mod != 0 {
		return ""
	}

	idx := int(key.Code - '1')
	if idx < 0 || idx >= len(apptypes.PageTitles) {
		return ""
	}

	return apptypes.PageTitles[idx]
}

// pageForStep returns the page [ or ] steps to from the active one: the next
// or previous entry in apptypes.PageTitles, wrapping around. "" for any other
// key, and the active page itself when it is the only one there is.
func (m AppModel) pageForStep(msg tea.KeyPressMsg) string {
	step := 0
	switch {
	case key.Matches(msg, keys.Global.NextPage):
		step = 1
	case key.Matches(msg, keys.Global.PrevPage):
		step = -1
	default:
		return ""
	}

	pages := apptypes.PageTitles
	current := max(0, slices.Index(pages, m.activePage))

	return pages[(current+step+len(pages))%len(pages)]
}

// pageForChord returns the page an alt+<letter> chord jumps to, or "" if the
// key is not a page chord.
//
// It matches on the modifier field rather than on msg.String(), because
// String() returns the printable text for a key ("g") and only falls back to
// the keystroke form ("alt+g") when there is none. Requiring Mod to be exactly
// ModAlt also means ctrl+alt+g and alt+shift+g are left alone.
func pageForChord(msg tea.KeyPressMsg) string {
	key := msg.Key()

	if key.Mod != tea.ModAlt {
		return ""
	}

	return apptypes.PageForShortcut(string(key.Code))
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// This var contains all the cmds that should be executed
	// at the end. Those can come from this model or from any of the
	// nested models in m.components
	var finalCmds []tea.Cmd

	// ctrl+c quits from anywhere, ahead of every other claim on the keyboard:
	// a modal, a text input, a filter being typed. It is the one key nothing
	// gets to swallow, which is why it is a binding of its own and not part of
	// Global.Quit.
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok && key.Matches(keyMsg, keys.Global.ForceQuit) {
		return m, tea.Quit
	}

	// While a modal is open, it owns all key input exclusively - the
	// underlying panels and Tab/quit handling are frozen until it closes.
	if m.activeModal != nil {
		if _, ok := msg.(tea.KeyPressMsg); ok {
			var modalCmd tea.Cmd
			m.activeModal, modalCmd = m.activeModal.Update(msg)
			return m, modalCmd
		}
	}

	switch msg := msg.(type) {
	// Handle keyboard events
	case tea.KeyPressMsg:
		// A filtering list owns the keyboard the way a modal does, with one
		// difference: the component below still has to receive the keystroke,
		// because it is the filter input. So this drops out of AppModel's own
		// key handling rather than returning, and the panels are updated as
		// usual at the bottom of this function.
		//
		// Tab is included in what AppModel gives up, so it does nothing while a
		// filter is being typed. Enter applies the filter and esc abandons it;
		// after either, tab moves panels again.
		if m.keyboardOwned() {
			break
		}

		// Digits, brackets and alt+<letter> all jump straight to a page.
		// Handled here rather than in MainMenu because the nav is not
		// focusable, so it never sees keys.
		if page := m.pageForNavKey(msg); page != "" {
			if page != m.activePage {
				finalCmds = append(finalCmds, cmds.SetActivePage(page))
			}
			break
		}

		switch {
		case key.Matches(msg, keys.Global.Quit):
			return m, tea.Quit

		case key.Matches(msg, keys.Global.Help):
			finalCmds = append(finalCmds, cmds.OpenHelpModal())

		case key.Matches(msg, keys.Global.About):
			finalCmds = append(finalCmds, cmds.OpenAboutModal())

		case key.Matches(msg, keys.Global.Theme):
			finalCmds = append(finalCmds, cmds.OpenThemePicker())

		case key.Matches(msg, keys.Global.NextPanel):
			tabCmd := m.ChangeFocus(nil)
			finalCmds = append(finalCmds, tabCmd)

		case key.Matches(msg, keys.Global.PrevPanel):
			idx := int(-1)
			tabCmd := m.ChangeFocus(&idx)
			finalCmds = append(finalCmds, tabCmd)

		// esc is "back": out of the details panel, to the list, and off an
		// error banner. Everything with a stronger claim on esc has already
		// had it - a modal closes itself above, a filter being typed owns the
		// keyboard above, and a focused list holding an applied filter keeps
		// it (escKept) - so by here esc belongs to the app. An error banner
		// showing is the next claim: dismiss it first, and the next esc backs
		// out of the details panel. It is the same one-key-one-job ladder a
		// filtered list clears on. A recovered poll already clears its own
		// banner; this is the manual dismissal for the errors that stay until
		// the next successful foreground operation.
		case key.Matches(msg, keys.Global.Back):
			if m.lastError != "" && !m.escKept() {
				m.lastError = ""
				m.lastErrorFromPoll = false
				break
			}
			if !m.escKept() && m.focusedComponent != constants.COMPONENT_BODY_LIST {
				leftPanel := constants.COMPONENT_BODY_LIST
				finalCmds = append(finalCmds, m.ChangeFocus(&leftPanel))
			}

		// n creates a group from either panel on Home. Handled here rather
		// than in GroupsList so it works regardless of which panel is focused.
		case key.Matches(msg, keys.List.New):
			if m.activePage == "Home" {
				finalCmds = append(finalCmds, cmds.OpenCreateGroupModal())
			}
		}

	// This is executed once when the app loads and after every
	// window resize.
	case tea.WindowSizeMsg:
		m.config.terminalWidth = msg.Width
		m.config.terminalHeight = msg.Height
		m.config.bodyLayout = m.calculateBodyLayout()
		finalCmds = append(finalCmds, m.broadcastBodyLayout())

	// Commands from the cmds folder
	case cmds.SetActivePageMsg:
		m.activePage = string(msg)

		// Each page starts at its primary (left) panel. Set activePage first so
		// the deferred focus message is routed to the page we just opened,
		// rather than the one we left.
		leftPanel := constants.COMPONENT_BODY_LIST
		finalCmds = append(finalCmds, m.ChangeFocus(&leftPanel))

		// Refresh container state, and re-sync services/groups, so the
		// newly active page's components have data to show even if they
		// weren't active when it was first loaded. Before config loads,
		// there is no compose file to query; waiting avoids a failing ps
		// command racing the bootstrap error on an empty directory.
		if m.config.configProject != nil {
			finalCmds = append(finalCmds, cmds.GetRunningContainers(m.config.configFileName))
		}
		finalCmds = append(finalCmds, m.configSyncCmds()...)
		if homeStatsCmd := m.broadcastHomeStats(); homeStatsCmd != nil {
			finalCmds = append(finalCmds, homeStatsCmd)
		}
		if cfCmd := m.recomposeFilesCmdIfActive(); cfCmd != nil {
			finalCmds = append(finalCmds, cfCmd)
		}
		finalCmds = append(finalCmds, m.broadcastBodyLayout())

	case cmds.RefreshContainersTickMsg:
		// Re-issue the tick whether or not this cycle refreshes, so the
		// poll keeps running for the life of the app.
		finalCmds = append(finalCmds, cmds.RefreshContainersTick())

		if m.shouldPollContainers() {
			finalCmds = append(finalCmds, cmds.GetRunningContainersBackground(m.config.configFileName))
		}

	case cmds.GetRunningContainersMsg:
		if msg.Err != nil {
			m.lastError = msg.Err.Error()
			m.lastErrorFromPoll = msg.Background
		} else {
			// A background success clears the banner only if the poll itself
			// put it up; a foreground success (page switch, finished action)
			// always clears it.
			if !msg.Background || m.lastErrorFromPoll {
				m.lastError = ""
				m.lastErrorFromPoll = false
			}
			// Fetch runtime stats for enriched container data.
			// For background polls, set waitingForStats so UpdateInnerComponent
			// skips forwarding this message to components - they will receive the
			// enriched version via GetContainerStatsMsg instead, avoiding a flicker.
			// Foreground actions (page switch, docker action) need immediate
			// updates so the UI reflects the new state right away.
			if msg.Background {
				m.waitingForStats = true
			}
			finalCmds = append(finalCmds, cmds.GetContainerStats(msg.Containers, msg.Background))
		}
		if bodyCmd := m.rebroadcastBodyLayoutIfChanged(); bodyCmd != nil {
			finalCmds = append(finalCmds, bodyCmd)
		}

	case cmds.GetContainerStatsMsg:
		m.waitingForStats = false
		// A stats failure is non-fatal and deliberately not surfaced: the
		// containers still arrive, only without their runtime numbers, so the
		// count is recomputed either way. Skipping this on Err would strand
		// the count at whatever the last successful stats call left, because
		// a background poll's own message never reaches the panels.
		if msg.Containers != nil {
			count := 0
			for _, container := range msg.Containers {
				if container.State == "running" {
					count++
				}
			}
			m.containers.runningCount = count
			if homeStatsCmd := m.broadcastHomeStats(); homeStatsCmd != nil {
				finalCmds = append(finalCmds, homeStatsCmd)
			}
		}
		if bodyCmd := m.rebroadcastBodyLayoutIfChanged(); bodyCmd != nil {
			finalCmds = append(finalCmds, bodyCmd)
		}

	case cmds.RunDockerActionMsg:
		// The panel asked for the action; AppModel is what knows which compose
		// file it has to run against.
		m.pendingAction = &chrome.PendingAction{
			Action:  msg.Action,
			Target:  msg.Target,
			IsGroup: msg.IsGroup,
		}
		finalCmds = append(finalCmds, cmds.SetPendingAction(msg.Action, msg.Target, msg.IsGroup))
		finalCmds = append(finalCmds, cmds.RunDockerAction(
			msg.Action, msg.Target, msg.IsGroup, m.config.configFileName,
		))

	case cmds.DockerActionMsg:
		// Clear the pending action state.
		m.pendingAction = nil
		finalCmds = append(finalCmds, cmds.ClearPendingAction())

		// Docker action errors are foreground: show a modal instead of the banner.
		if msg.Err != nil {
			finalCmds = append(finalCmds, m.reportForegroundError(msg.Err.Error()))
		} else {
			// Success: clear the banner and check for running containers.
			m.lastError = ""
			m.lastErrorFromPoll = false
			finalCmds = append(finalCmds, cmds.GetRunningContainers(m.config.configFileName))
		}
		if bodyCmd := m.rebroadcastBodyLayoutIfChanged(); bodyCmd != nil {
			finalCmds = append(finalCmds, bodyCmd)
		}

	case cmds.GetConfigMsg:
		if msg.Err != nil {
			m.lastError = msg.Err.Error()
			m.lastErrorFromPoll = false
			// No compose file where we looked: offer to create one there.
			// The error banner is still set above, so an Esc from the modal
			// leaves a visible explanation.
			//
			// Only when nothing else owns the screen. A second failed load
			// arriving while this modal is up would otherwise replace it with
			// a fresh one, wiping out a filename half-typed into it - and a
			// modal the user opened deliberately is not something a
			// background reload gets to close.
			if errors.Is(msg.Err, utils.ErrNoComposeFile) && m.activeModal == nil {
				m.activeModal = components.CreateComposeFileModal(m.config.source.Dir)
			}
			if bodyCmd := m.rebroadcastBodyLayoutIfChanged(); bodyCmd != nil {
				finalCmds = append(finalCmds, bodyCmd)
			}
			break
		}

		m.config.configFileName = msg.FileName
		m.config.configProject = msg.Project
		// Tests construct GetConfigMsg by hand without the candidates; the
		// winner on its own is still a complete answer.
		m.config.configFiles = msg.Files
		if len(m.config.configFiles) == 0 && msg.FileName != "" {
			m.config.configFiles = []string{msg.FileName}
		}
		finalCmds = append(finalCmds, cmds.GetRunningContainers(m.config.configFileName))
		// The footer starts out saying no file is loaded, so only a successful
		// load has anything to report - a failed one leaves the previous answer
		// standing, which is still the file the docker commands would act on.
		var others []string
		if len(m.config.configFiles) > 1 {
			others = m.config.configFiles[1:]
		}
		finalCmds = append(finalCmds, cmds.SetComposeFile(msg.FileName, others))
		finalCmds = append(finalCmds, m.configSyncCmds()...)
		if homeStatsCmd := m.broadcastHomeStats(); homeStatsCmd != nil {
			finalCmds = append(finalCmds, homeStatsCmd)
		}
		if cfCmd := m.recomposeFilesCmdIfActive(); cfCmd != nil {
			finalCmds = append(finalCmds, cfCmd)
		}
		if bodyCmd := m.rebroadcastBodyLayoutIfChanged(); bodyCmd != nil {
			finalCmds = append(finalCmds, bodyCmd)
		}

	case cmds.OpenCreateGroupModalMsg:
		if m.config.configProject != nil {
			m.activeModal = components.GroupNameModal(
				m.allGroupNames(), m.config.configProject.ServiceNames(),
				m.config.terminalHeight,
			)
		}

	case cmds.OpenLogsModalMsg:
		var startCmd tea.Cmd
		m.activeModal, startCmd = logsmodal.New(
			msg.Target, msg.IsGroup, m.config.configFileName,
			m.config.terminalWidth, m.config.terminalHeight,
		)
		finalCmds = append(finalCmds, startCmd)

	case cmds.OpenDeleteGroupModalMsg:
		groupName := string(msg)
		m.activeModal = confirmmodal.New(
			fmt.Sprintf("Delete group %q?", groupName),
			cmds.DeleteGroup(m.config.configFileName, groupName),
		)

	case cmds.OpenEditGroupModalMsg:
		if m.config.configProject != nil {
			members := m.groupMembers(msg.GroupName)
			m.activeModal = servicechecklistmodal.NewForEdit(
				msg.GroupName, m.config.configProject.ServiceNames(), members,
				m.config.terminalHeight,
			)
		}

	case cmds.OpenRenameGroupModalMsg:
		if m.config.configProject != nil {
			m.activeModal = components.GroupNameModalForRename(
				msg.GroupName, m.allGroupNames(),
			)
		}

	// Observed, not handled: these are on their way to the panels, and
	// AppModel only notes what was picked so a reload can restore it.
	case cmds.SetSelectedServiceMsg:
		m.selection.serviceName = types.ServiceConfig(msg).Name

	case cmds.SetSelectedGroupMsg:
		m.selection.groupName = string(msg)

	case cmds.OpenEditorMsg:
		if m.config.configFileName == "" {
			m.lastError = "No compose file to edit"
			m.lastErrorFromPoll = false
			break
		}

		m.externalEditorOpen = true
		finalCmds = append(finalCmds, cmds.RunEditor(m.config.configFileName))

	case cmds.SetEditingStateMsg:
		m.inlineEditing = bool(msg)

	case cmds.RequestInlineEditMsg:
		if m.config.configFileName == "" {
			finalCmds = append(finalCmds, func() tea.Msg {
				return cmds.InlineEditReadyMsg{
					ServiceName: msg.ServiceName,
					Err:         fmt.Errorf("no compose file to edit"),
				}
			})
			break
		}

		fragment, err := utils.ExtractServiceFragment(m.config.configFileName, msg.ServiceName)
		finalCmds = append(finalCmds, func() tea.Msg {
			return cmds.InlineEditReadyMsg{
				ServiceName: msg.ServiceName,
				Fragment:    fragment,
				Err:         err,
			}
		})

	case cmds.RequestSaveServiceMsg:
		if m.config.configFileName == "" {
			finalCmds = append(finalCmds, func() tea.Msg {
				return cmds.ServiceSavedMsg{
					ServiceName: msg.ServiceName,
					Err:         fmt.Errorf("no compose file to save to"),
				}
			})
			break
		}

		finalCmds = append(finalCmds, func() tea.Msg {
			return cmds.ServiceSavedMsg{
				ServiceName: msg.ServiceName,
				Err:         utils.ApplyServiceFragment(m.config.configFileName, msg.ServiceName, msg.Fragment),
			}
		})

	case cmds.ServiceSavedMsg:
		// The panel keeps the editor open and shows the error inline, so the
		// app banner is not touched. A successful save is followed by the
		// usual config reload.
		if msg.Err == nil {
			finalCmds = append(finalCmds, cmds.GetConfig(m.config.source))
			if cfCmd := m.recomposeFilesCmdIfActive(); cfCmd != nil {
				finalCmds = append(finalCmds, cfCmd)
			}
		}

	case cmds.OpenServiceEditorMsg:
		if m.config.configFileName == "" {
			m.lastError = "No compose file to edit"
			m.lastErrorFromPoll = false
			break
		}

		m.externalEditorOpen = true
		finalCmds = append(finalCmds, cmds.EditService(m.config.configFileName, msg.ServiceName))

	case cmds.ServiceEditedMsg:
		m.externalEditorOpen = false
		m.lastErrorFromPoll = false

		if msg.Err != nil {
			// The compose file is untouched. Carry the underlying message
			// through - "invalid compose file" on its own tells the user
			// nothing they can act on.
			errMsg := fmt.Sprintf("Editing %s: %s", msg.ServiceName, msg.Err)
			finalCmds = append(finalCmds, m.reportForegroundError(errMsg))
			break
		}

		m.lastError = ""
		finalCmds = append(finalCmds, cmds.GetConfig(m.config.source))
		if cfCmd := m.recomposeFilesCmdIfActive(); cfCmd != nil {
			finalCmds = append(finalCmds, cfCmd)
		}

	case cmds.EditorClosedMsg:
		m.externalEditorOpen = false
		m.lastErrorFromPoll = false

		if msg.Err != nil {
			finalCmds = append(finalCmds, m.reportForegroundError(msg.Err.Error()))
			break
		}

		// Reload unconditionally: the user may have saved anything, or
		// nothing, and re-reading is cheaper than working out which.
		m.lastError = ""
		finalCmds = append(finalCmds, cmds.GetConfig(m.config.source))
		if cfCmd := m.recomposeFilesCmdIfActive(); cfCmd != nil {
			finalCmds = append(finalCmds, cfCmd)
		}

	case cmds.OpenHelpModalMsg:
		m.activeModal = helpoverlay.New(
			m.helpContext(),
			m.config.configFiles,
			m.config.terminalWidth,
		)

	case cmds.OpenAboutModalMsg:
		m.activeModal = aboutmodal.New()

	case cmds.OpenErrorModalMsg:
		finalCmds = append(finalCmds, m.reportForegroundError(msg.Message))

	case cmds.OpenThemePickerMsg:
		m.activeModal = components.ThemePickerModal(m.config.terminalHeight)

	case cmds.ThemeAppliedMsg:
		// CloseModal already cleared activeModal. Report a persist
		// error in the banner; a success needs no feedback — the whole
		// UI is already repainted in the new theme.
		m.lastErrorFromPoll = false
		if msg.Err != nil {
			m.lastError = msg.Err.Error()
		} else {
			m.lastError = ""
		}
		if bodyCmd := m.rebroadcastBodyLayoutIfChanged(); bodyCmd != nil {
			finalCmds = append(finalCmds, bodyCmd)
		}

	case cmds.OpenConfirmModalMsg:
		m.activeModal = confirmmodal.New(msg.Message, msg.Follow)

	case cmds.CloseModalMsg:
		m.activeModal = nil
		if msg.Follow != nil {
			finalCmds = append(finalCmds, msg.Follow)
		}

	case cmds.CreateGroupRequestMsg:
		// Same split as RunDockerActionMsg: the modal knows the group, AppModel
		// knows the file it has to be written into.
		finalCmds = append(finalCmds, cmds.CreateGroup(
			m.config.configFileName, msg.Name, msg.Services,
		))

	case cmds.EditGroupRequestMsg:
		// Same split as CreateGroupRequestMsg: the modal knows the group
		// and members, AppModel knows the file they must be written into.
		finalCmds = append(finalCmds, cmds.EditGroup(
			m.config.configFileName, msg.GroupName, msg.Members,
		))

	case cmds.RenameGroupRequestMsg:
		// Same split as CreateGroupRequestMsg: the modal knows the names,
		// AppModel knows the file the rename must be written into.
		finalCmds = append(finalCmds, cmds.RenameGroup(
			m.config.configFileName, msg.GroupName, msg.NewName,
		))

	case cmds.EditGroupMsg:
		m.lastErrorFromPoll = false
		if msg.Err != nil {
			finalCmds = append(finalCmds, m.reportForegroundError(msg.Err.Error()))
		} else {
			m.lastError = ""
			finalCmds = append(finalCmds, cmds.GetConfig(m.config.source))
		}
		if bodyCmd := m.rebroadcastBodyLayoutIfChanged(); bodyCmd != nil {
			finalCmds = append(finalCmds, bodyCmd)
		}

	case cmds.RenameGroupMsg:
		m.lastErrorFromPoll = false
		if msg.Err != nil {
			finalCmds = append(finalCmds, m.reportForegroundError(msg.Err.Error()))
		} else {
			m.lastError = ""
			// Keep the renamed group selected: configSyncCmds re-selects
			// selection.groupName after the reload, and it still holds the
			// old name until now. On failure the old selection stands.
			m.selection.groupName = msg.NewName
			finalCmds = append(finalCmds, cmds.GetConfig(m.config.source))
		}
		if cfCmd := m.recomposeFilesCmdIfActive(); cfCmd != nil {
			finalCmds = append(finalCmds, cfCmd)
		}
		if bodyCmd := m.rebroadcastBodyLayoutIfChanged(); bodyCmd != nil {
			finalCmds = append(finalCmds, bodyCmd)
		}

	case cmds.CreateGroupMsg:
		m.lastErrorFromPoll = false
		if msg.Err != nil {
			finalCmds = append(finalCmds, m.reportForegroundError(msg.Err.Error()))
		} else {
			m.lastError = ""
			finalCmds = append(finalCmds, cmds.GetConfig(m.config.source))
		}
		if cfCmd := m.recomposeFilesCmdIfActive(); cfCmd != nil {
			finalCmds = append(finalCmds, cfCmd)
		}
		if bodyCmd := m.rebroadcastBodyLayoutIfChanged(); bodyCmd != nil {
			finalCmds = append(finalCmds, bodyCmd)
		}

	case cmds.DeleteGroupMsg:
		m.lastErrorFromPoll = false
		if msg.Err != nil {
			finalCmds = append(finalCmds, m.reportForegroundError(msg.Err.Error()))
		} else {
			m.lastError = ""
			finalCmds = append(finalCmds, cmds.GetConfig(m.config.source))
		}
		if cfCmd := m.recomposeFilesCmdIfActive(); cfCmd != nil {
			finalCmds = append(finalCmds, cfCmd)
		}
		if bodyCmd := m.rebroadcastBodyLayoutIfChanged(); bodyCmd != nil {
			finalCmds = append(finalCmds, bodyCmd)
		}

	case cmds.CreateComposeFileMsg:
		m.lastErrorFromPoll = false
		if msg.Err != nil {
			finalCmds = append(finalCmds, m.reportForegroundError(msg.Err.Error()))
		} else {
			m.lastError = ""
			finalCmds = append(finalCmds, cmds.GetConfig(m.config.source))
		}
		if cfCmd := m.recomposeFilesCmdIfActive(); cfCmd != nil {
			finalCmds = append(finalCmds, cfCmd)
		}
		if bodyCmd := m.rebroadcastBodyLayoutIfChanged(); bodyCmd != nil {
			finalCmds = append(finalCmds, bodyCmd)
		}

	case cmds.ComposeFileContentsMsg:
		// Only the active page's components see messages, and this one is
		// for the Files page's viewport. Nothing to do here at the app
		// level - the message reaches the component via UpdateInnerComponent
		// above.

	case cmds.OpenComposeFilePickerMsg:
		// Browsing only makes sense with a file loaded: the picker lists the
		// YAML files alongside the active one, and with none loaded there is
		// no directory to look in (the bootstrap modal owns that state).
		if m.config.configFileName != "" {
			finalCmds = append(finalCmds, cmds.ListComposeFiles(
				filepath.Dir(m.config.configFileName),
			))
		}

	case cmds.ComposeFileListMsg:
		if msg.Err != nil {
			m.lastError = msg.Err.Error()
			m.lastErrorFromPoll = false
			break
		}
		if len(msg.Files) == 0 {
			break
		}
		m.activeModal = components.ComposeFilePickerModal(
			msg.Dir, msg.Files, filepath.Base(m.config.configFileName),
			m.config.terminalHeight,
		)

	case cmds.SwitchComposeFileMsg:
		// Switching is exactly passing --file at runtime: point the source
		// at the chosen path and reload. Every downstream consumer - the
		// docker calls, the writers, the footer, the lists - already flows
		// from the resolved file, so they follow without further work.
		m.config.source = utils.ComposeSource{File: msg.Path}
		finalCmds = append(finalCmds, cmds.GetConfig(m.config.source))
	}

	if m.activeModal != nil {
		var modalCmd tea.Cmd
		m.activeModal, modalCmd = m.activeModal.Update(msg)
		finalCmds = append(finalCmds, modalCmd)
	}

	// Update nested components
	var mainMenuCmd tea.Cmd
	m.components.MainMenu, mainMenuCmd = m.components.MainMenu.Update(msg)

	var keybindingBarCmd tea.Cmd
	m.components.KeybindingBar, keybindingBarCmd = m.components.KeybindingBar.Update(msg)

	var innerComponentsCmd tea.Cmd
	if m.shouldForwardToComponents(msg) {
		innerComponentsCmd = m.UpdateInnerComponent(m.activePage, msg)
	}
	finalCmds = append(finalCmds, mainMenuCmd, keybindingBarCmd, innerComponentsCmd)

	return m, tea.Batch(finalCmds...)
}
