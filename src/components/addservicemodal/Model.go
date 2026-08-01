// Package addservicemodal is the n entry point on the Services page: a
// search-first modal (Spotlight/Telescope-style live results table) that
// hands off to a confirm stage, then to the existing, unchanged write path
// (cmds.AddService -> inline editor). See docs/plans/image-search.md D2/D3.
package addservicemodal

import (
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/filipemolina/stack-stitcher/src/appstyles"
	"github.com/filipemolina/stack-stitcher/src/components/chrome"
	"github.com/filipemolina/stack-stitcher/src/keys"
)

type addStep int

const (
	stepSearch addStep = iota
	stepConfirm
)

// searchListHeight is how many result rows the search stage's list shows.
// Fixed, not derived from the terminal: New has no termHeight parameter
// (unlike healthcheckpickermodal.New) and the plan keeps it that way so the
// call site in src/model/Update.go does not change. At 10 rows the whole
// modal is ~21 rows tall - it fits a 24-row terminal, the smallest this
// app's chrome meaningfully renders in.
const searchListHeight = 10

type Model struct {
	fileName             string
	existingServiceNames []string

	step addStep

	// stepSearch fields. query doubles as the image field once a result is
	// picked with no highlighted row (D3) - there is deliberately no
	// separate "image" input at this stage.
	query      textinput.Model
	results    list.Model
	generation int    // bumped on every keystroke and every fired search (D3)
	searching  bool   // true between a fired search and its result/staleness
	searchErr  string // set on a search failure; cleared on the next successful one

	// stepConfirm fields - a small, deliberately un-shared copy of
	// servicefieldsstep's two-input shape (D7).
	serviceName textinput.Model
	image       textinput.Model
	confirmErr  string
}

func (m Model) Init() tea.Cmd { return nil }

// New builds the search-first modal. fileName is the compose file the new
// service is written into; existingServiceNames is what is already there,
// used by the confirm stage's inline collision check (D7).
func New(fileName string, existingServiceNames []string) tea.Model {
	query := textinput.New()
	query.Placeholder = "search Docker Hub"
	query.SetWidth(40)
	query.Focus()

	cl := list.New(nil, resultsDelegate{width: 40}, 40, searchListHeight)
	// Filtering disabled first: list.updatePagination's row budget reserves
	// a title/filter-input row whenever showFilter && filteringEnabled,
	// regardless of showTitle - the same gotcha healthcheckpickermodal
	// documents and that this list shares. Its "filtering" is the network
	// search itself, not list.Model's local substring filter (D8).
	cl.SetFilteringEnabled(false)
	cl.SetShowTitle(false)
	cl.SetShowHelp(false)
	cl.SetShowStatusBar(false)
	cl.SetShowPagination(searchListHeight < len(cl.Items()))

	return Model{
		fileName:             fileName,
		existingServiceNames: existingServiceNames,
		step:                 stepSearch,
		query:                query,
		results:              cl,
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.query, cmd = m.query.Update(msg)
	return m, cmd
}

func (m Model) View() tea.View {
	sections := []string{
		chrome.ModalTitle("Search Docker Hub"),
		m.query.View(),
	}

	sections = append(sections, "", chrome.ModalHints(
		chrome.HintFor(keys.Overlay.Submit),
		chrome.HintFor(keys.Overlay.Cancel),
	))

	return tea.NewView(chrome.ModalSurface(
		appstyles.Active.ModalBg,
		lipgloss.JoinVertical(lipgloss.Left, sections...),
	))
}
