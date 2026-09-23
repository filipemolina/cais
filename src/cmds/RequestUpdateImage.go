package cmds

import (
	tea "charm.land/bubbletea/v2"

	"github.com/filipemolina/cais/src/utils"
)

// RequestUpdateImageMsg asks AppModel to open the update confirm for one
// service. Panels emit this rather than the command itself for the same
// reason they emit RunDockerActionMsg: the write and the compose calls act on
// the file the app resolved, and that is AppModel's state, not a panel's.
type RequestUpdateImageMsg struct {
	Service string
}

// RequestUpdateImage is the panel-side entry to the update action, usable
// from a key handler.
func RequestUpdateImage(service string) tea.Cmd {
	return func() tea.Msg {
		return RequestUpdateImageMsg{Service: service}
	}
}

// ImageUpdatedMsg is the update action's terminal message. Err carries the
// failure of the pin write or of the pull/up — both are foreground errors.
// A nil Err means the pin is in the file, the container is running the new
// image, and the glyph can clear from the locally re-resolved digest.
type ImageUpdatedMsg struct {
	Service string
	Err     error
}

// UpdateImage is the confirm's follow-up command: the digest pin, then the
// pull and the up -d against the service named — the same verbs the action
// keys use, through the same composeActionArgs plumbing, so the two compose
// calls are scoped exactly the way every other action is.
//
// Every step stops at the first failure: a refused pin must not pull, and a
// failed pull or up must not pretend the update happened. Each failure is
// the terminal message's Err, which the foreground error path renders; a
// failed pull leaves the pin in the file — that is honest state, and U (or
// p) retries it. The glyph is not touched here: the model clears it from its
// own map on success, no network call — the pinned reference now resolves
// locally to exactly the digest it was pinned to.
func UpdateImage(fileName string, service string, digest string) tea.Cmd {
	return func() tea.Msg {
		if err := utils.PinImageDigest(fileName, service, digest); err != nil {
			return ImageUpdatedMsg{Service: service, Err: err}
		}

		if err := utils.RunDockerCompose("pull", service, false, fileName, nil); err != nil {
			return ImageUpdatedMsg{Service: service, Err: err}
		}

		if err := utils.RunDockerCompose("start", service, false, fileName, nil); err != nil {
			return ImageUpdatedMsg{Service: service, Err: err}
		}

		return ImageUpdatedMsg{Service: service}
	}
}
