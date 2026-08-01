package createcomposefilemodal

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/filipemolina/stack-stitcher/src/cmds"
	"github.com/filipemolina/stack-stitcher/src/keys"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

func (m Model) updateFilename(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Overlay.Cancel):
		return m, cmds.CloseModal(nil)
	case key.Matches(msg, keys.Overlay.Submit):
		name := strings.TrimSpace(m.filename.Value())
		if name == "" {
			m.errMsg = "Filename can't be empty"
			return m, nil
		}
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			m.errMsg = "Filename must end in .yaml or .yml"
			return m, nil
		}
		if _, err := os.Stat(filepath.Join(m.dir, name)); err == nil {
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

func (m Model) updateAddServicePrompt(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Overlay.Cancel):
		return m, cmds.CloseModal(nil)
	case key.Matches(msg, keys.Overlay.Yes):
		m.errMsg = ""
		m.step = stepServiceFields
		return m, m.serviceName.Focus()
	case key.Matches(msg, keys.Overlay.No):
		return m, cmds.CloseModal(cmds.CreateComposeFile(m.path(), "", ""))
	}

	return m, nil
}

func (m Model) updateServiceFields(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Overlay.Cancel):
		return m, cmds.CloseModal(nil)
	case key.Matches(msg, keys.Overlay.NextField):
		if m.serviceName.Focused() {
			m.serviceName.Blur()
			return m, m.image.Focus()
		}
		m.image.Blur()
		return m, m.serviceName.Focus()
	case key.Matches(msg, keys.Overlay.Submit):
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
		return m, cmds.CloseModal(cmds.CreateComposeFile(m.path(), name, image))
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
