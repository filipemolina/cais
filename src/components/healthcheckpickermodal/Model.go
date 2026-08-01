// Package healthcheckpickermodal is the h entry point on the Services page:
// a list of healthcheck templates relevant to the selected service, with an
// inline port field that appears only while the generic HTTP template is
// highlighted. See docs/plans/healthcheck-insertion.md's "UX: the modal"
// for why this is one modal rather than a two-step handoff.
package healthcheckpickermodal

import (
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/compose-spec/compose-go/v2/types"
	"github.com/filipemolina/stack-stitcher/src/components/chrome"
	"github.com/filipemolina/stack-stitcher/src/utils"
)

// templateItem wraps a catalog entry as a list.Item. FilterValue is unused
// - the list's own filtering is off, since a picker of five rows has
// nothing worth filtering and "/" would otherwise fight the port field for
// keystrokes.
type templateItem struct {
	template utils.HealthcheckTemplate
}

func (t templateItem) FilterValue() string { return t.template.Name }

type Model struct {
	serviceName string
	// replacing is true when the service already carries a healthcheck: -
	// the picker labels its submit hint "replace" rather than "add" so a
	// user is not surprised that choosing a template overwrites, rather
	// than merges with, whatever is already there.
	replacing bool
	list      list.Model
	portInput textinput.Model
	errMsg    string
}

func (m Model) Init() tea.Cmd { return nil }

// selectedTemplate returns the catalog entry the cursor is on.
func (m Model) selectedTemplate() (utils.HealthcheckTemplate, bool) {
	item, ok := m.list.SelectedItem().(templateItem)
	if !ok {
		return utils.HealthcheckTemplate{}, false
	}
	return item.template, true
}

// New builds the picker for svc: utils.TemplatesFor orders image-matched
// templates first, the generic HTTP fallback last. The port field is
// prefilled from the service's first published port's target (else 80),
// so the common case - an unrecognised image - is h, Enter: one modal, one
// keystroke.
func New(serviceName string, svc types.ServiceConfig, termHeight int) tea.Model {
	templates := utils.TemplatesFor(svc.Image)

	items := make([]list.Item, len(templates))
	for i, t := range templates {
		items[i] = templateItem{template: t}
	}

	visible := chrome.ModalListHeight(len(items), termHeight)
	cl := list.New(items, templateDelegate{}, 40, visible)
	// Filtering disabled first: list.updatePagination's row budget reserves
	// a title/filter-input row whenever showFilter && filteringEnabled,
	// regardless of showTitle, and SetFilteringEnabled alone does not
	// retrigger that calculation the way every SetShowX call does. Left
	// last, it silently starves the list to one visible row forever - the
	// catalog has as few as two entries, so that bug is not a hairline
	// case here the way it might be for a longer list.
	cl.SetFilteringEnabled(false)
	cl.SetShowTitle(false)
	cl.SetShowHelp(false)
	cl.SetShowStatusBar(false)
	cl.SetShowPagination(visible < len(items))

	port := textinput.New()
	port.Placeholder = "80"
	port.SetWidth(6)
	port.SetValue(utils.DefaultGenericPort(svc))
	port.Focus()

	return Model{
		serviceName: serviceName,
		replacing:   svc.HealthCheck != nil,
		list:        cl,
		portInput:   port,
	}
}
