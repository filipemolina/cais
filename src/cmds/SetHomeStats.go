package cmds

import tea "charm.land/bubbletea/v2"

// SetHomeStatsMsg carries the counts shown in the home page status header.
// Groups is the number of distinct groups (Compose profiles) in the loaded
// project. Services is the total number of services (across all profiles).
// Running is the number of containers currently in the "running" state.
type SetHomeStatsMsg struct {
	Groups   int
	Services int
	Running  int
}

func SetHomeStats(groups, services, running int) tea.Cmd {
	return func() tea.Msg {
		return SetHomeStatsMsg{Groups: groups, Services: services, Running: running}
	}
}
