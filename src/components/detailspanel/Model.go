package detailspanel

import (
	"github.com/filipemolina/cais/src/apptypes"
	"github.com/filipemolina/cais/src/components/chrome"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"github.com/compose-spec/compose-go/v2/types"
)

type Model struct {
	service     *types.ServiceConfig
	panelWidth  int
	panelHeight int

	// editing is true while the inline YAML editor is open.
	editing bool
	// editor is the textarea used in inline edit mode.
	editor textarea.Model
	// originalFragment is the fragment the editor started from, used to
	// detect unsaved changes.
	originalFragment []byte
	// validationError is the last YAML syntax error from the live parse.
	validationError string
	// saveError is the last compose-level error from a failed save.
	saveError string

	pendingAction *chrome.PendingAction
	spinner       spinner.Model

	// containers is the latest known container list, used to derive the
	// RUNNING/STOPPED status pill in the panel title row.
	containers []apptypes.DockerContainer

	// host is the address utils.ResolveURL builds every service URL
	// against - resolved once, at startup, since it cannot change during a
	// run (see utils.URLHost).
	host string
	// urlMessage is a status-line confirmation ("copied http://…") set by
	// the y key, cleared on the next keypress or selection change the same
	// way saveError/validationError are.
	urlMessage string
	// applyHint tells the user a just-saved healthcheck needs a restart to
	// take effect - restart (r) reuses the container's existing config,
	// only start (s) re-reads compose (docs/plans/healthcheck-insertion.md,
	// "The apply gap"). Set when AddHealthcheckMsg succeeds for the
	// selected, running service; cleared on the next docker action request
	// or selection change.
	applyHint string
}

func (m Model) Init() tea.Cmd {
	return nil
}

// EditorValue returns the editor's current contents. Exported for the model
// tests, which drive paste and indentation through the whole message path and
// need to see what landed in the buffer.
func (m Model) EditorValue() string {
	return m.editor.Value()
}

// EditorCursor returns the editor's current row and column. Exported for the
// model tests covering tab/outdent/backspace, which pin cursor position, not
// just buffer text - that is where the SetValue-resets-the-cursor bug would
// show up.
func (m Model) EditorCursor() (int, int) {
	return m.editor.Line(), m.editor.Column()
}

// OwnsKeyboard reports whether the panel is holding the whole keyboard. This
// is the same contract the filtered lists use: while true, AppModel stands
// down from its own keys so letters are letters, not commands. The editor
// is always focused when it is open, so it does not need to check focus.
func (m Model) OwnsKeyboard() bool {
	return m.editing
}

func New(service *types.ServiceConfig, host string) tea.Model {
	return Model{
		service: service,
		host:    host,
		spinner: chrome.NewSpinner(),
	}
}
