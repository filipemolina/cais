package cmds

import tea "charm.land/bubbletea/v2"

// SetHomeStatsMsg carries the counts shown in the home page status header.
// Groups is the number of distinct groups (Compose profiles) in the loaded
// project. Services is the total number of services (across all profiles).
// Running is the number of containers currently in the "running" state.
// Ungrouped is the number of services that carry no profile tag at all -
// Compose starts these alongside *every* group regardless of which one was
// asked for, so they are worth a standing count rather than something the
// user only discovers by tracing a failed start back to a service they
// never selected.
type SetHomeStatsMsg struct {
	Groups    int
	Services  int
	Running   int
	Ungrouped int
}

func SetHomeStats(groups, services, running, ungrouped int) tea.Cmd {
	return func() tea.Msg {
		return SetHomeStatsMsg{Groups: groups, Services: services, Running: running, Ungrouped: ungrouped}
	}
}
