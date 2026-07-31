package model

import (
	"slices"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/compose-spec/compose-go/v2/types"
	"github.com/filipemolina/stack-stitcher/src/cmds"
	"github.com/filipemolina/stack-stitcher/src/components"
	"github.com/filipemolina/stack-stitcher/src/components/chrome"
	"github.com/filipemolina/stack-stitcher/src/constants"
	"github.com/filipemolina/stack-stitcher/src/utils"
)

type navigationModel struct {
	currentPage string
}

type configModel struct {
	// source is where to look for the compose file, as the flags left it.
	// Every reload re-resolves from it rather than reusing configFileName, so
	// a file created after startup (the bootstrap flow) is found.
	source         utils.ComposeSource
	configFileName string
	// configFiles is every compose-file candidate that exists in the
	// directory, in Docker's priority order (configFileName is the first).
	// The help overlay lists them; configFileName alone is kept because most
	// consumers only care about the winner.
	configFiles    []string
	configProject  *types.Project
	terminalWidth  int
	terminalHeight int
	// bodyLayout is the box the body panels render into. AppModel owns it
	// (see calculateBodyLayout) and broadcasts it; components never derive
	// their own size from the terminal dimensions.
	bodyLayout cmds.SetBodyLayoutMsg
}

type containersModel struct {
	runningContainers []list.Item
	runningCount      int
}

// selectionModel remembers what the user had selected, by name, so a config
// reload can put it back. Reloads happen after every write to the compose
// file, and re-selecting the first service each time would throw the user
// back to the top of the list the moment they changed anything.
type selectionModel struct {
	serviceName string
	groupName   string
}

type Components struct {
	MainMenu      tea.Model
	KeybindingBar tea.Model
}

type AppModel struct {
	navigation       navigationModel
	config           configModel
	containers       containersModel
	selection        selectionModel
	pages            map[string][]tea.Model
	activePage       string
	components       Components
	focusedComponent int
	lastError        string
	// lastErrorFromPoll records whether the banner is showing an error from
	// the background container poll, so the next successful poll can clear
	// it without touching errors from other sources (e.g. a failed action).
	lastErrorFromPoll bool
	activeModal       tea.Model
	// externalEditorOpen is set while an editor has been handed the
	// terminal. The app is suspended for that whole time, so background work
	// would only pile up messages to process on resume.
	externalEditorOpen bool
	// inlineEditing is true while the service details panel is editing a
	// service inline. It is broadcast back to the panel via the same message
	// the panel uses to announce it, so the help overlay and the footer can
	// show the editor keys.
	inlineEditing bool
	// pendingAction tracks a docker action that is currently running.
	// While set, action keys are disabled and a spinner is shown.
	pendingAction *chrome.PendingAction
	// waitingForStats is true while a GetContainerStats command is in flight.
	// When set, GetRunningContainersMsg is not forwarded to components to
	// avoid a flicker where stats disappear for one render cycle.
	waitingForStats bool
}

// ChangeFocus moves focus through constants.FocusableComponents and returns the
// command that tells the components which of them is now focused.
//
// Pass nil to advance (Tab), or -1 to go back (Shift+Tab). Any other value is
// treated as a component id to focus directly, and is ignored if that component
// is not focusable.
//
// The cycle position is derived from m.focusedComponent rather than tracked
// separately, so the two can never disagree. They are not the same number: the
// nav is component 0 but is not in the cycle, so the ids are not the cycle
// indices.
func (m *AppModel) ChangeFocus(index *int) tea.Cmd {
	order := constants.FocusableComponents
	if len(order) == 0 {
		return nil
	}

	// A component id that is not in the cycle (or an unset one) reads as
	// position 0, so the first Tab lands on the first focusable component.
	cursor := max(0, slices.Index(order, m.focusedComponent))

	switch {
	case index == nil:
		cursor = (cursor + 1) % len(order)

	case *index == -1:
		cursor = (cursor - 1 + len(order)) % len(order)

	default:
		if !slices.Contains(order, *index) {
			return nil
		}
		cursor = slices.Index(order, *index)
	}

	m.focusedComponent = order[cursor]
	focused := m.focusedComponent

	return func() tea.Msg { return cmds.SetFocusMsg(focused) }
}

// allGroupNames returns every distinct group referenced by any service
// in the loaded compose project, sorted. Returns nil if no project is
// loaded yet.
func (m AppModel) allGroupNames() []string {
	if m.config.configProject == nil {
		return nil
	}

	var groups []string
	for _, service := range m.config.configProject.Services {
		groups = append(groups, service.Profiles...)
	}

	groups = utils.Deduplicate(groups)
	slices.Sort(groups)

	return groups
}

// groupMembers returns the names of every service that currently carries
// groupName as a profile tag, sorted for stable checklist ordering.
func (m AppModel) groupMembers(groupName string) []string {
	if m.config.configProject == nil {
		return nil
	}

	var members []string
	for _, service := range m.config.configProject.Services {
		for _, profile := range service.Profiles {
			if profile == groupName {
				members = append(members, service.Name)
				break
			}
		}
	}

	slices.Sort(members)
	return members
}

// recomposeFilesCmdIfActive returns a command that reads the raw compose
// file for the Files page's viewport, or nil when the Files page is not
// active. Called on page switch and after any write through the app, so
// the viewport stays in sync with disk.
func (m AppModel) recomposeFilesCmdIfActive() tea.Cmd {
	if m.activePage != "Compose Files" || m.config.configFileName == "" {
		return nil
	}
	return cmds.GetComposeFileContents(m.config.configFileName)
}

// homeStats returns the counts shown in the home page status header:
// groups (distinct Compose profiles), services (total in the project),
// and running containers.
func (m AppModel) homeStats() (groups, services, running int) {
	if m.config.configProject != nil {
		groups = len(m.allGroupNames())
		services = len(m.config.configProject.Services)
	}
	running = m.containers.runningCount
	return
}

// broadcastHomeStats returns a SetHomeStats command for the home page,
// or nil if the active page isn't Home. Call this whenever the underlying
// data changes so the status header stays in sync.
func (m AppModel) broadcastHomeStats() tea.Cmd {
	if m.activePage != "Home" {
		return nil
	}
	groups, services, running := m.homeStats()
	return cmds.SetHomeStats(groups, services, running)
}

func (m *AppModel) UpdateInnerComponent(activePage string, msg tea.Msg) tea.Cmd {
	var finalCmds []tea.Cmd

	innerComponents, ok := m.pages[activePage]

	if ok {
		for idx, _ := range innerComponents {
			var componentCmd tea.Cmd
			m.pages[activePage][idx], componentCmd = m.pages[activePage][idx].Update(msg)
			finalCmds = append(finalCmds, componentCmd)
		}
	}

	return tea.Batch(finalCmds...)
}

// shouldForwardToComponents reports whether a message should be passed to
// the active page's components. GetRunningContainersMsg is skipped while
// waiting for stats to avoid a flicker where stats disappear for one
// render cycle.
func (m AppModel) shouldForwardToComponents(msg tea.Msg) bool {
	if m.waitingForStats {
		if _, ok := msg.(cmds.GetRunningContainersMsg); ok {
			return false
		}
	}
	return true
}

// GetInitialModel builds the app's starting state. source is what the -f/-d
// flags resolved to; the zero value means "the compose file in the current
// directory", which is what a bare run gets.
func GetInitialModel(source utils.ComposeSource) AppModel {
	pages := make(map[string][]tea.Model)

	pages["Home"] = []tea.Model{
		components.GroupsList([]string{}, 0, 0),
		components.GroupDetailsPanel(),
	}

	pages["Services"] = []tea.Model{
		components.ServicesList([]types.ServiceConfig{}, 0, 0),
		components.DetailsPanel(nil),
	}

	// Every page in apptypes.PageTitles needs an entry here. A page missing
	// from this map renders an empty body, which used to drop the app out of
	// the alternate screen and look like a crash.
	pages["Compose Files"] = []tea.Model{
		components.ComposeFilePanel(),
	}

	return AppModel{
		containers: containersModel{
			runningContainers: []list.Item{},
		},
		config: configModel{
			source:         source,
			configFileName: "",
			configProject:  nil,
		},
		components: Components{
			MainMenu:      components.MainMenu(),
			KeybindingBar: components.KeybindingBar(),
		},
		pages: pages,
		// Page activation sends this focus to the active page's components.
		// Keeping the model in the same initial state makes the first Tab move
		// to the details panel rather than appearing to do nothing.
		focusedComponent: constants.COMPONENT_BODY_LIST,
	}
}
