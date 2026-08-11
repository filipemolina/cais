package createcomposefilemodal

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/filipemolina/cais/src/cmds"
	"github.com/filipemolina/cais/src/components/servicefieldsstep"
	"github.com/filipemolina/cais/src/keys"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.step == stepServiceFields {
		var cmd tea.Cmd
		m.serviceStep, cmd = m.serviceStep.Update(msg)
		return m, cmd
	}

	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		switch m.step {
		case stepFilename:
			return m.updateFilename(keyMsg)
		case stepAddServicePrompt:
			return m.updateAddServicePrompt(keyMsg)
		}
	}

	// Forward non-key messages (e.g. WindowSizeMsg) to the active input so
	// the cursor keeps blinking and resize still flows.
	if m.step == stepFilename {
		var cmd tea.Cmd
		m.filename, cmd = m.filename.Update(msg)
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
		path := m.path()
		var cmd tea.Cmd
		m.serviceStep, cmd = servicefieldsstep.New(
			fmt.Sprintf("Creating %s", filepath.Base(strings.TrimSpace(m.filename.Value()))),
			func(name, image string) tea.Cmd { return cmds.CreateComposeFile(path, name, image) },
		)
		return m, cmd
	case key.Matches(msg, keys.Overlay.No):
		return m, cmds.CloseModal(cmds.CreateComposeFile(m.path(), "", ""))
	}

	return m, nil
}
