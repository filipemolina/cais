package groupnamemodal

import (
	"fmt"
	"slices"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/filipemolina/cais/src/apptypes"
	"github.com/filipemolina/cais/src/cmds"
	"github.com/filipemolina/cais/src/components/servicechecklistmodal"
	"github.com/filipemolina/cais/src/keys"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

			case name == apptypes.UngroupedGroup:
				// Reserved: cais shows a group of this name for every service
				// with no profiles: key. A real group by the same name would
				// collide with it in the list and in every membership check.
				m.errMsg = fmt.Sprintf("%q is reserved for services with no group", name)
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

			return servicechecklistmodal.New(name, m.serviceNames, m.termHeight), nil
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)

	return m, cmd
}
