// Package addservicemodal is the n entry point on the Services page: a
// search-first modal (Spotlight/Telescope-style live results table) that
// hands off to a confirm stage, then to the existing, unchanged write path
// (cmds.AddService -> inline editor). See docs/plans/image-search.md D2/D3.
package addservicemodal

import (
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/filipemolina/stack-stitcher/src/appstyles"
	"github.com/filipemolina/stack-stitcher/src/cmds"
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
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok && m.step == stepSearch {
		switch {
		case keyMsg.Code == tea.KeyUp:
			m.results.CursorUp()
			return m, nil
		case keyMsg.Code == tea.KeyDown:
			m.results.CursorDown()
			return m, nil
		case key.Matches(keyMsg, keys.Overlay.Cancel):
			return m, cmds.CloseModal(nil)
		case key.Matches(keyMsg, keys.Overlay.Submit):
			return m.advanceToConfirm()
		}

		// Every other key, including every letter list.DefaultKeyMap would
		// otherwise claim, goes to the query input (D3 - stricter than
		// healthcheckpickermodal's port field, because this input is never
		// not focused).
		var cmd tea.Cmd
		m.query, cmd = m.query.Update(keyMsg)
		m.generation++
		gen := m.generation
		m.results.SetItems(nil) // clear stale results immediately, don't wait for the debounce
		m.searchErr = ""
		if len(strings.TrimSpace(m.query.Value())) < 2 {
			return m, cmd // too short to search - stay in the empty state, no timer armed
		}
		return m, tea.Batch(cmd, cmds.SearchDebounce(gen))
	}

	switch msg := msg.(type) {
	case cmds.SearchDebounceMsg:
		if msg.Generation != m.generation {
			return m, nil // superseded by a later keystroke - do nothing
		}
		m.searching = true
		return m, cmds.SearchImages(strings.TrimSpace(m.query.Value()), 20, m.generation)

	case cmds.SearchImagesMsg:
		if msg.Generation != m.generation {
			return m, nil // a stale result from an earlier keystroke - discard (D3)
		}
		m.searching = false
		if msg.Err != nil {
			m.searchErr = "image search unavailable — type the full image reference and press enter"
			return m, nil
		}
		items := make([]list.Item, len(msg.Results))
		for i, r := range msg.Results {
			items[i] = searchResultItem{result: r}
		}
		m.results.SetItems(items)
		if len(items) == 0 {
			m.searchErr = "no images matched"
		}
		return m, nil
	}

	// Non-key messages (cursor blink ticks, window size) still need to
	// reach both sub-components.
	var listCmd, inputCmd tea.Cmd
	m.results, listCmd = m.results.Update(msg)
	m.query, inputCmd = m.query.Update(msg)
	return m, tea.Batch(listCmd, inputCmd)
}

// advanceToConfirm moves the modal from the search stage to the confirm
// stage, picking the highlighted result or, with nothing highlighted, the
// typed text (D3). Filled in by step 7 of docs/plans/image-search.md Phase
// 2A.
func (m Model) advanceToConfirm() (Model, tea.Cmd) { return m, nil }

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
