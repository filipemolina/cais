package cmds

import (
	"fmt"

	"github.com/filipemolina/cais/src/utils"

	tea "charm.land/bubbletea/v2"
	"gopkg.in/yaml.v3"
)

// AddServiceMsg reports the result of writing a new service into the
// compose file. Image and Fragment (set on success) are what AppModel needs
// to open the inline editor on the new service without waiting on a
// separate reload's selection broadcast to land first - see the Select
// field on InlineEditReadyMsg.
type AddServiceMsg struct {
	ServiceName string
	Image       string
	Fragment    []byte
	Err         error
}

// AddService writes serviceName/image as a new, minimal two-line service
// into the compose file at fileName - the whole feature minus the network
// (docs/plans/image-search.md Phase 1). It lands as the smallest fragment
// that exists, on purpose: the inline editor opens on it next so the user
// adds ports, volumes and everything else in the same YAML they would have
// hand-written, rather than this modal growing a field for each of them
// (docs/DESIGN.md §Editing services).
//
// fileName is supplied by AppModel, not typed by the user - the same split
// RunDockerActionMsg uses for the file the docker calls act on.
func AddService(fileName, serviceName, image string) tea.Cmd {
	return func() tea.Msg {
		// Marshalled rather than hand-formatted so an image reference with a
		// YAML-significant character (a leading `*`, `&`, `!`, quoting a
		// digest's `@sha256:...`) is quoted correctly instead of producing a
		// fragment that fails to parse.
		fragment, err := yaml.Marshal(map[string]map[string]string{
			serviceName: {"image": image},
		})
		if err != nil {
			return AddServiceMsg{ServiceName: serviceName, Err: fmt.Errorf("building the new service: %w", err)}
		}

		return AddServiceMsg{
			ServiceName: serviceName,
			Image:       image,
			Fragment:    fragment,
			Err:         utils.AddServiceFragment(fileName, serviceName, fragment),
		}
	}
}
