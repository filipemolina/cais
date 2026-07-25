package components

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"stack-stitcher/src/appstyles"
	"stack-stitcher/src/cmds"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type createStep int

const (
	stepFilename createStep = iota
	stepAddServicePrompt
	stepServiceFields
)

type CreateComposeFileModalModel struct {
	step        createStep
	filename    textinput.Model
	serviceName textinput.Model
	image       textinput.Model
	errMsg      string
}

func (m CreateComposeFileModalModel) Init() tea.Cmd {
	return nil
}

func (m CreateComposeFileModalModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		switch m.step {
		case stepFilename:
			return m.updateFilename(keyMsg)
		case stepAddServicePrompt:
			return m.updateAddServicePrompt(keyMsg)
		case stepServiceFields:
			return m.updateServiceFields(keyMsg)
		}
	}

	// Forward non-key messages (e.g. WindowSizeMsg) to the active input so
	// the cursor keeps blinking and resize still flows.
	switch m.step {
	case stepFilename:
		var cmd tea.Cmd
		m.filename, cmd = m.filename.Update(msg)
		return m, cmd
	case stepServiceFields:
		if m.serviceName.Focused() {
			var cmd tea.Cmd
			m.serviceName, cmd = m.serviceName.Update(msg)
			return m, cmd
		}
		var cmd tea.Cmd
		m.image, cmd = m.image.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m CreateComposeFileModalModel) updateFilename(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return m, cmds.CloseModal(nil)
	case "enter":
		name := strings.TrimSpace(m.filename.Value())
		if name == "" {
			m.errMsg = "Filename can't be empty"
			return m, nil
		}
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			m.errMsg = "Filename must end in .yaml or .yml"
			return m, nil
		}
		if _, err := os.Stat(name); err == nil {
			m.errMsg = fmt.Sprintf("%s already exists in this directory", name)
			return m, nil
		} else if !os.IsNotExist(err) {
			m.errMsg = fmt.Sprintf("Can't stat %s: %v", name, err)
			return m, nil
		}
		m.errMsg = ""
		m.step = stepAddServicePrompt
		return m, nil
	}

	var cmd tea.Cmd
	m.filename, cmd = m.filename.Update(msg)
	return m, cmd
}

func (m CreateComposeFileModalModel) updateAddServicePrompt(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return m, cmds.CloseModal(nil)
	case "y", "Y":
		m.errMsg = ""
		m.step = stepServiceFields
		return m, m.serviceName.Focus()
	case "n", "N":
		return m, cmds.CloseModal(cmds.CreateComposeFile(strings.TrimSpace(m.filename.Value()), "", ""))
	}

	return m, nil
}

func (m CreateComposeFileModalModel) updateServiceFields(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return m, cmds.CloseModal(nil)
	case "tab":
		if m.serviceName.Focused() {
			m.serviceName.Blur()
			return m, m.image.Focus()
		}
		m.image.Blur()
		return m, m.serviceName.Focus()
	case "enter":
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
		if !isValidServiceName(name) {
			m.errMsg = fmt.Sprintf("%q is not a valid service name", name)
			return m, nil
		}
		return m, cmds.CloseModal(cmds.CreateComposeFile(strings.TrimSpace(m.filename.Value()), name, image))
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

func isValidServiceName(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '-' || r == '_':
		default:
			return false
		}
	}
	return true
}

func (m CreateComposeFileModalModel) View() tea.View {
	errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#B33A3A"))
	var lines []string

	switch m.step {
	case stepFilename:
		lines = []string{
			"New compose file",
			"Filename (in the current directory):",
			m.filename.View(),
		}
	case stepAddServicePrompt:
		lines = []string{
			fmt.Sprintf("Creating %s", filepath.Base(strings.TrimSpace(m.filename.Value()))),
			"Add a first service? (y/n)",
		}
	case stepServiceFields:
		lines = []string{
			fmt.Sprintf("Creating %s", filepath.Base(strings.TrimSpace(m.filename.Value()))),
			"Service name:",
			m.serviceName.View(),
			"Image:",
			m.image.View(),
		}
	}

	if m.errMsg != "" {
		lines = append(lines, errStyle.Render(m.errMsg))
	}

	return tea.NewView(modalSurface(
		appstyles.PanelBackgroundColor,
		lipgloss.JoinVertical(lipgloss.Left, lines...),
	))
}

// CreateComposeFileModal walks the user through creating a brand-new compose
// file in the current directory: a filename (with a sane default and basic
// validation) and an optional one-service seed. Esc cancels the whole flow
// at any point - the file is never half-created.
func CreateComposeFileModal() tea.Model {
	filename := textinput.New()
	filename.Placeholder = "compose.yaml"
	filename.SetWidth(40)
	filename.SetValue("compose.yaml")
	filename.CursorEnd()
	filename.Focus()

	serviceName := textinput.New()
	serviceName.Placeholder = "e.g. web"
	serviceName.SetWidth(30)
	serviceName.Focus()

	image := textinput.New()
	image.Placeholder = "e.g. nginx:alpine"
	image.SetWidth(30)

	return CreateComposeFileModalModel{
		step:        stepFilename,
		filename:    filename,
		serviceName: serviceName,
		image:       image,
	}
}
