package model

import (
	"stack-stitcher/src/appstyles"

	tea "charm.land/bubbletea/v2"
)

func (m AppModel) View() tea.View {
	doc := appstyles.DocStyle.Render(
		m.containers.runningContainers.View(),
	)

	v := tea.NewView(doc)
	v.AltScreen = true

	return v
}
