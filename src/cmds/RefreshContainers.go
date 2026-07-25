package cmds

import (
	"time"

	tea "charm.land/bubbletea/v2"
)

// ContainerRefreshInterval is how often the app re-polls `docker compose ps`
// in the background, so panel statuses stay current when containers change
// outside the app (another terminal, a restart policy, ...).
const ContainerRefreshInterval = 5 * time.Second

type RefreshContainersTickMsg time.Time

// RefreshContainersTick fires once after ContainerRefreshInterval. AppModel
// re-issues it on every RefreshContainersTickMsg, which is what makes the
// poll recurring.
func RefreshContainersTick() tea.Cmd {
	return tea.Tick(ContainerRefreshInterval, func(t time.Time) tea.Msg {
		return RefreshContainersTickMsg(t)
	})
}
