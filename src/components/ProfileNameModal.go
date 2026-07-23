package components

import (
	"fmt"
	"slices"
	"stack-stitcher/src/appstyles"
	"stack-stitcher/src/cmds"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type ProfileNameModalModel struct {
	input            textinput.Model
	existingProfiles []string
	serviceNames     []string
	errMsg           string
}

func (m ProfileNameModalModel) Init() tea.Cmd {
	return nil
}

func (m ProfileNameModalModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		switch keyMsg.String() {
		case "esc":
			return m, cmds.CloseModal(nil)

		case "enter":
			name := m.input.Value()

			if name == "" {
				m.errMsg = "Profile name can't be empty"
				return m, nil
			}

			if slices.Contains(m.existingProfiles, name) {
				m.errMsg = fmt.Sprintf("Profile %q already exists", name)
				return m, nil
			}

			return ServiceChecklistModal(name, m.serviceNames), nil
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)

	return m, cmd
}

func (m ProfileNameModalModel) View() tea.View {
	style := lipgloss.NewStyle().
		Padding(1, 2).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(appstyles.PrimaryColor).
		Background(appstyles.PanelBackgroundColor)

	lines := []string{"New profile name:", m.input.View()}
	if m.errMsg != "" {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#B33A3A"))
		lines = append(lines, errStyle.Render(m.errMsg))
	}

	return tea.NewView(style.Render(lipgloss.JoinVertical(lipgloss.Left, lines...)))
}

// ProfileNameModal is step 1 of the create-profile flow: prompt for a new,
// unique profile name. Enter with a valid name advances to
// ServiceChecklistModal; Esc cancels the whole flow.
func ProfileNameModal(existingProfiles []string, serviceNames []string) tea.Model {
	input := textinput.New()
	input.Placeholder = "e.g. core"
	input.SetWidth(30)
	input.Focus()

	return ProfileNameModalModel{
		input:            input,
		existingProfiles: existingProfiles,
		serviceNames:     serviceNames,
	}
}
