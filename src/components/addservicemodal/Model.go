// Package addservicemodal is the n entry point on the Services page: a thin
// wrapper around servicefieldsstep, the name+image step shared with
// createcomposefilemodal's optional first service (D2 in
// docs/plans/image-search.md).
package addservicemodal

import (
	"fmt"
	"slices"

	tea "charm.land/bubbletea/v2"
	"github.com/filipemolina/stack-stitcher/src/appstyles"
	"github.com/filipemolina/stack-stitcher/src/cmds"
	"github.com/filipemolina/stack-stitcher/src/components/chrome"
	"github.com/filipemolina/stack-stitcher/src/components/servicefieldsstep"
)

type Model struct {
	step servicefieldsstep.Model
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.step, cmd = m.step.Update(msg)
	return m, cmd
}

func (m Model) View() tea.View {
	return tea.NewView(chrome.ModalSurface(appstyles.Active.ModalBg, m.step.View()))
}

// New builds the "add a service" modal. fileName is the compose file the new
// service is written into; existingServiceNames is what is already there.
//
// The collision check runs before cmds.AddService is ever dispatched - the
// same synchronous check groupnamemodal makes against its own
// existingGroups - so a name that is already taken reopens as an error
// instead of the user waiting on a write to disk that was always going to
// fail. It lives here, in the closure New builds for servicefieldsstep,
// rather than as a third parameter on that shared component (D2): the
// bootstrap flow's equivalent step can never collide (there is nothing in
// the file yet), so the check is specific to this caller.
func New(fileName string, existingServiceNames []string) tea.Model {
	step, _ := servicefieldsstep.New("New service", func(name, image string) tea.Cmd {
		if slices.Contains(existingServiceNames, name) {
			return cmds.OpenErrorModal(fmt.Sprintf("Service %q already exists", name))
		}
		return cmds.AddService(fileName, name, image)
	})

	return Model{step: step}
}
