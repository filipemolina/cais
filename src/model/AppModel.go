package model

import (
	"slices"
	"stack-stitcher/src/cmds"
	"stack-stitcher/src/components"
	"stack-stitcher/src/constants"
	"stack-stitcher/src/utils"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/compose-spec/compose-go/v2/types"
)

type navigationModel struct {
	currentPage string
}

type configModel struct {
	configFileName string
	configProject  *types.Project
	terminalWidht  int
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

func GetInitialModel() AppModel {
	pages := make(map[string][]tea.Model)

	pages["Home"] = []tea.Model{
		components.GroupsList([]string{}, 0, 0),
		components.GroupDetailsPanel(),
	}

	pages["Dashboard"] = []tea.Model{
		components.ServicesList([]types.ServiceConfig{}, 0, 0),
		components.DetailsPanel(nil),
	}

	// Every page in apptypes.PageTitles needs an entry here. A page missing
	// from this map renders an empty body, which used to drop the app out of
	// the alternate screen and look like a crash.
	pages["Compose Files"] = []tea.Model{
		components.PlaceholderPanel("Files",
			"Browsing and editing compose files from here is not built yet. For now, Stack Stitcher reads the compose file in the directory it was started from."),
	}

	pages["Settings"] = []tea.Model{
		components.PlaceholderPanel("Settings",
			"There is nothing to configure yet. Colors, key bindings and the default compose file will live here."),
	}

	return AppModel{
		containers: containersModel{
			runningContainers: []list.Item{},
		},
		config: configModel{
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
