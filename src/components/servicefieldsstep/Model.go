// Package servicefieldsstep is the two-field step every "make a service"
// flow needs: a name and an image, then out to the existing YAML editor
// rather than a form with a field for every compose key -
// docs/DESIGN.md §Editing services rejects that outright.
//
// It is shared, not copied, between createcomposefilemodal (the bootstrap
// flow's optional first service) and addservicemodal (`n` on the Services
// page) precisely so a later feature added here - Docker Hub search, a tag
// picker - lands in both flows at once instead of leaving the bootstrap
// flow behind. See D2 in docs/plans/image-search.md.
package servicefieldsstep

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/filipemolina/stack-stitcher/src/appstyles"
	"github.com/filipemolina/stack-stitcher/src/cmds"
	"github.com/filipemolina/stack-stitcher/src/components/chrome"
	"github.com/filipemolina/stack-stitcher/src/keys"
	"github.com/filipemolina/stack-stitcher/src/utils"
)

// Model is not a tea.Model in its own right - it has no Init and its
// Update/View take and return the concrete type, not the interface - because
// it is always embedded inside a modal that owns the surrounding chrome
// (title text, what else the modal shows before or after this step) rather
// than run standalone.
type Model struct {
	title       string
	onSubmit    func(name, image string) tea.Cmd
	serviceName textinput.Model
	image       textinput.Model
	errMsg      string
}

// New builds the step. title is shown as the step's heading; onSubmit turns
// the validated (name, image) pair into the command the embedding modal
// should run once it closes - cmds.CreateComposeFile for the bootstrap flow,
// cmds.AddService for the Services page. That is deliberately the whole
// contract: a third parameter is the signal to stop sharing this component
// and copy the step instead (D2).
//
// The returned command focuses the service-name field; embed it in whatever
// command the caller already returns from the keypress that opens this step.
func New(title string, onSubmit func(name, image string) tea.Cmd) (Model, tea.Cmd) {
	serviceName := textinput.New()
	serviceName.Placeholder = "e.g. web"
	serviceName.SetWidth(30)

	image := textinput.New()
	image.Placeholder = "e.g. nginx:alpine"
	image.SetWidth(30)

	cmd := serviceName.Focus()

	return Model{
		title:       title,
		onSubmit:    onSubmit,
		serviceName: serviceName,
		image:       image,
	}, cmd
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
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

func (m Model) submit() (Model, tea.Cmd) {
	name := strings.TrimSpace(m.serviceName.Value())
	image := strings.TrimSpace(m.image.Value())

	if name == "" {
		m.errMsg = "Service name can't be empty"
		return m, nil
	}
	if image == "" {
		m.errMsg = "Image can't be empty (e.g. nginx:alpine)"
		return m, nil
	}
	if !utils.IsValidServiceName(name) {
		m.errMsg = fmt.Sprintf("%q is not a valid service name", name)
		return m, nil
	}

	return m, cmds.CloseModal(m.onSubmit(name, image))
}

// View renders the step's body - title, both fields, an error line when one
// of the validations above tripped, and its own hint line, since the modal
// embedding it is not expected to know this step's keys beyond Cancel. The
// caller wraps the result in chrome.ModalSurface.
func (m Model) View() string {
	lines := []string{
		chrome.ModalTitle(m.title),
		"Service name:",
		m.serviceName.View(),
		"Image:",
		m.image.View(),
	}

	if m.errMsg != "" {
		lines = append(lines, lipgloss.NewStyle().Foreground(appstyles.Active.Danger).Render(m.errMsg))
	}

	lines = append(lines, "", chrome.ModalHints(
		chrome.HintFor(keys.Overlay.NextField),
		chrome.HintFor(keys.Overlay.Submit),
		chrome.HintFor(keys.Overlay.Cancel),
	))

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}
