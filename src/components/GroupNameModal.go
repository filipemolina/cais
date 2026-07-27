package components

import (
	"fmt"
	"slices"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/filipemolina/stack-stitcher/src/appstyles"
	"github.com/filipemolina/stack-stitcher/src/cmds"
	"github.com/filipemolina/stack-stitcher/src/keys"
)

type GroupNameModalModel struct {
	input          textinput.Model
	existingGroups []string
	serviceNames   []string
	errMsg         string
}

func (m GroupNameModalModel) Init() tea.Cmd {
	return nil
}

func (m GroupNameModalModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		switch {
		case key.Matches(keyMsg, keys.Overlay.Cancel):
			return m, cmds.CloseModal(nil)

		case key.Matches(keyMsg, keys.Overlay.Submit):
			name := m.input.Value()

			if name == "" {
				m.errMsg = "Group name can't be empty"
				return m, nil
			}

			if slices.Contains(m.existingGroups, name) {
				m.errMsg = fmt.Sprintf("Group %q already exists", name)
				return m, nil
			}

			return ServiceChecklistModal(name, m.serviceNames), nil
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)

	return m, cmd
}

func (m GroupNameModalModel) View() tea.View {
	lines := []string{"New group name:", m.input.View()}
	if m.errMsg != "" {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#B33A3A"))
		lines = append(lines, errStyle.Render(m.errMsg))
	}

	return tea.NewView(modalSurface(
		appstyles.Active.ModalBg,
		lipgloss.JoinVertical(lipgloss.Left, lines...),
	))
}

// GroupNameModal is step 1 of the create-group flow: prompt for a new,
// unique group name. Enter with a valid name advances to
// ServiceChecklistModal; Esc cancels the whole flow.
func GroupNameModal(existingGroups []string, serviceNames []string) tea.Model {
	input := textinput.New()
	input.Placeholder = "e.g. core"
	input.SetWidth(30)
	input.Focus()

	return GroupNameModalModel{
		input:          input,
		existingGroups: existingGroups,
		serviceNames:   serviceNames,
	}
}
