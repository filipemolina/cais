// Package addservicemodal is the n entry point on the Services page: a
// search-first modal (Spotlight/Telescope-style live results table) that
// hands off to a confirm stage, then to the existing, unchanged write path
// (cmds.AddService -> inline editor). See docs/plans/image-search.md D2/D3.
package addservicemodal

import (
	"fmt"
	"slices"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/filipemolina/stack-stitcher/src/appstyles"
	"github.com/filipemolina/stack-stitcher/src/cmds"
	"github.com/filipemolina/stack-stitcher/src/components/chrome"
	"github.com/filipemolina/stack-stitcher/src/keys"
	"github.com/filipemolina/stack-stitcher/src/utils"
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
	spinner    spinner.Model
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
		spinner:              chrome.NewSpinner(),
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		if m.step == stepSearch {
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

		// stepConfirm: the same two-field interaction servicefieldsstep has,
		// deliberately not the same type (D7). Esc closes the whole modal -
		// no "go back to search", the only modal in this app that would.
		switch {
		case key.Matches(keyMsg, keys.Overlay.Cancel):
			return m, cmds.CloseModal(nil)
		case key.Matches(keyMsg, keys.Overlay.NextField):
			if m.serviceName.Focused() {
				m.serviceName.Blur()
				return m, m.image.Focus()
			}
			m.image.Blur()
			return m, m.serviceName.Focus()
		case key.Matches(keyMsg, keys.Overlay.Submit):
			return m.submit()
		}

		if m.serviceName.Focused() {
			var cmd tea.Cmd
			m.serviceName, cmd = m.serviceName.Update(keyMsg)
			return m, cmd
		}
		var cmd tea.Cmd
		m.image, cmd = m.image.Update(keyMsg)
		return m, cmd
	}

	switch msg := msg.(type) {
	case cmds.SearchDebounceMsg:
		if msg.Generation != m.generation {
			return m, nil // superseded by a later keystroke - do nothing
		}
		m.searching = true
		// Arm the spinner the way detailspanel does when a pending action
		// starts: Update with its own TickMsg returns the next tick command,
		// which this case returns alongside the search itself.
		var spCmd tea.Cmd
		m.spinner, spCmd = m.spinner.Update(m.spinner.Tick())
		return m, tea.Batch(cmds.SearchImages(strings.TrimSpace(m.query.Value()), 20, m.generation), spCmd)

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

	case spinner.TickMsg:
		if !m.searching {
			return m, nil // search resolved - let the tick chain die
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	// Non-key messages (cursor blink ticks, window size) still need to
	// reach the sub-components live for the current stage.
	if m.step == stepSearch {
		var listCmd, inputCmd tea.Cmd
		m.results, listCmd = m.results.Update(msg)
		m.query, inputCmd = m.query.Update(msg)
		return m, tea.Batch(listCmd, inputCmd)
	}

	if m.serviceName.Focused() {
		var cmd tea.Cmd
		m.serviceName, cmd = m.serviceName.Update(msg)
		return m, cmd
	}
	var cmd tea.Cmd
	m.image, cmd = m.image.Update(msg)
	return m, cmd
}

// submit validates the confirm stage's two fields and hands off to the
// unchanged write path: cmds.AddService -> the inline editor. The same
// validation servicefieldsstep applies, plus the collision check re-run
// here because the user may have edited the name field since the stage
// rendered (D7).
func (m Model) submit() (Model, tea.Cmd) {
	name := strings.TrimSpace(m.serviceName.Value())
	image := strings.TrimSpace(m.image.Value())

	if name == "" {
		m.confirmErr = "Service name can't be empty"
		return m, nil
	}
	if image == "" {
		m.confirmErr = "Image can't be empty (e.g. nginx:alpine)"
		return m, nil
	}
	if !utils.IsValidServiceName(name) {
		m.confirmErr = fmt.Sprintf("%q is not a valid service name", name)
		return m, nil
	}
	if slices.Contains(m.existingServiceNames, name) {
		m.confirmErr = fmt.Sprintf("Service %q already exists", name)
		return m, nil
	}

	return m, cmds.CloseModal(cmds.AddService(m.fileName, name, image))
}

// advanceToConfirm moves the modal from the search stage to the confirm
// stage. Enter's rule, precisely (D3): a highlighted row's image wins; with
// nothing highlighted - short query, no matches, or a search error - the
// literal text in the input is used as the image, verbatim, no validation
// beyond "not empty." Either way the free-text escape hatch for non-Hub
// registries stays open: an empty results table means the same thing
// whether its cause is a non-Hub path, zero matches, or a search failure.
func (m Model) advanceToConfirm() (Model, tea.Cmd) {
	image := ""
	if item, ok := m.results.SelectedItem().(searchResultItem); ok {
		image = item.result.Name
	} else {
		image = strings.TrimSpace(m.query.Value())
	}
	if image == "" {
		return m, nil // nothing highlighted and nothing typed - do nothing
	}

	m.step = stepConfirm
	m.serviceName = textinput.New()
	m.serviceName.SetValue(deriveServiceName(image))
	m.serviceName.SetWidth(30)
	m.image = textinput.New()
	m.image.SetValue(image)
	m.image.SetWidth(30)
	cmd := m.serviceName.Focus()

	// The collision check runs the moment the confirm stage renders, not
	// only on submit (D7) - a genuine improvement this redesign gets for
	// free because the confirm stage is now a distinct moment.
	if slices.Contains(m.existingServiceNames, m.serviceName.Value()) {
		m.confirmErr = fmt.Sprintf("Service %q already exists", m.serviceName.Value())
	}

	return m, cmd
}

// deriveServiceName assumes the image's own name as the service name (the
// refinement request's literal ask): strip a :tag or @digest suffix, then
// take the substring after the last "/". "nginx" stays "nginx";
// "linuxserver/sonarr" becomes "sonarr". Never auto-sanitized if the
// result fails utils.IsValidServiceName - the confirm stage's existing
// validation explains the problem and the user fixes it (D7).
func deriveServiceName(image string) string {
	name := image
	if i := strings.IndexAny(name, "@:"); i != -1 {
		name = name[:i]
	}
	if i := strings.LastIndex(name, "/"); i != -1 {
		name = name[i+1:]
	}
	return name
}

func (m Model) View() tea.View {
	if m.step == stepConfirm {
		lines := []string{
			chrome.ModalTitle("New service"),
			"Service name:",
			m.serviceName.View(),
			"Image:",
			m.image.View(),
		}

		if m.confirmErr != "" {
			lines = append(lines, lipgloss.NewStyle().Foreground(appstyles.Active.Danger).Render(m.confirmErr))
		}

		lines = append(lines, "", chrome.ModalHints(
			chrome.HintFor(keys.Overlay.NextField),
			chrome.HintFor(keys.Overlay.Submit),
			chrome.HintFor(keys.Overlay.Cancel),
		))

		return tea.NewView(chrome.ModalSurface(
			appstyles.Active.ModalBg,
			lipgloss.JoinVertical(lipgloss.Left, lines...),
		))
	}

	sections := []string{
		chrome.ModalTitle("Search Docker Hub"),
		m.query.View(),
	}

	// Four body states below the input, first match wins (the plan's
	// Step 9 order): in-flight search, quiet error, not-yet-searched hint,
	// then the results table. The spinner is armed in Update on
	// SearchDebounceMsg and ticks only while m.searching.
	switch {
	case m.searching:
		spinnerLine := m.spinner.View() + " searching…"
		sections = append(sections, lipgloss.NewStyle().Foreground(appstyles.Active.TextMuted).Render(spinnerLine))
	case m.searchErr != "":
		// TextMuted, deliberately not Danger: this is a quiet degradation
		// (D6a) - the user can still type a full reference and continue.
		sections = append(sections, lipgloss.NewStyle().Foreground(appstyles.Active.TextMuted).Render(m.searchErr))
	case len(strings.TrimSpace(m.query.Value())) < 2:
		sections = append(sections, lipgloss.NewStyle().Foreground(appstyles.Active.TextMuted).Render("Type to search Docker Hub"))
	default:
		sections = append(sections, m.results.View())
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
