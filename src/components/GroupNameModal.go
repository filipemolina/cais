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
	// isRename switches step 1 of the create flow into the whole rename
	// flow: Enter writes the rename instead of advancing to the service
	// checklist, and uniqueness is judged against the other groups - the
	// name being renamed is not a collision.
	isRename bool
	// currentName is the group being renamed; the input is pre-filled with
	// it and a submit that leaves it unchanged is refused.
	currentName string
	// termHeight is carried rather than used here: this modal is one text
	// input tall, but step 2 is a list of every service and has to be sized
	// to the screen. Update builds it directly, so the height has to arrive
	// with the flow rather than from AppModel at that point.
	termHeight int
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

			switch {
			case name == "":
				m.errMsg = "Group name can't be empty"
				return m, nil

			case m.isRename && name == m.currentName:
				// The same name would still rewrite the whole file (closing
				// blank lines - see README's YAML caveat), so refuse it as a
				// no-op rather than doing the write.
				m.errMsg = fmt.Sprintf("Group is already named %q", name)
				return m, nil

			case slices.Contains(m.existingGroups, name):
				// For a rename, the group being renamed is itself in
				// existingGroups; the currentName guard above already
				// rejected it, so this only fires for a genuine collision.
				m.errMsg = fmt.Sprintf("Group %q already exists", name)
				return m, nil

			case m.isRename:
				return m, cmds.CloseModal(cmds.RequestRenameGroup(m.currentName, name))
			}

			return ServiceChecklistModal(name, m.serviceNames, m.termHeight), nil
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)

	return m, cmd
}

func (m GroupNameModalModel) View() tea.View {
	title := "New group"
	submitDesc := "next"
	if m.isRename {
		title = "Rename group"
		submitDesc = "rename"
	}

	lines := []string{modalTitle(title), "Group name:", m.input.View()}
	if m.errMsg != "" {
		errStyle := lipgloss.NewStyle().Foreground(appstyles.Active.Danger)
		lines = append(lines, errStyle.Render(m.errMsg))
	}

	// Enter is "next" on the create flow (step 1 of two, handing off to
	// the service checklist rather than writing anything) and "rename" on
	// the rename flow (the only step, which writes).
	lines = append(lines, "", modalHints(
		hintAs(keys.Overlay.Submit, submitDesc),
		hintFor(keys.Overlay.Cancel),
	))

	return tea.NewView(modalSurface(
		appstyles.Active.ModalBg,
		lipgloss.JoinVertical(lipgloss.Left, lines...),
	))
}

// GroupNameModal is step 1 of the create-group flow: prompt for a new,
// unique group name. Enter with a valid name advances to
// ServiceChecklistModal; Esc cancels the whole flow.
func GroupNameModal(existingGroups []string, serviceNames []string, termHeight int) tea.Model {
	input := textinput.New()
	input.Placeholder = "e.g. core"
	input.SetWidth(30)
	input.Focus()

	return GroupNameModalModel{
		input:          input,
		existingGroups: existingGroups,
		serviceNames:   serviceNames,
		termHeight:     termHeight,
	}
}

// GroupNameModalForRename is the rename flow: prompt for the group's new
// name, pre-filled with the current one (cursor at end; ctrl+u clears it
// wholesale). Enter writes the rename via RequestRenameGroup; Esc cancels.
// Uniqueness excludes the current name, so renaming core to core gets its
// own message rather than "already exists". No termHeight: unlike the
// create flow there is no step-2 checklist to size to the screen.
func GroupNameModalForRename(currentName string, existingGroups []string) tea.Model {
	input := textinput.New()
	input.Placeholder = "e.g. core"
	input.SetWidth(30)
	input.SetValue(currentName)
	input.Focus()

	return GroupNameModalModel{
		input:          input,
		existingGroups: existingGroups,
		isRename:       true,
		currentName:    currentName,
	}
}
