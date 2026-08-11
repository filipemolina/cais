package createcomposefilemodal

import (
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/filipemolina/cais/src/components/servicefieldsstep"
)

type createStep int

const (
	stepFilename createStep = iota
	stepAddServicePrompt
	stepServiceFields
)

type Model struct {
	step createStep
	// dir is the directory the app was told to look in (--dir), so the file
	// is created where it will then be found. Empty is the current directory.
	dir         string
	filename    textinput.Model
	serviceStep servicefieldsstep.Model
	errMsg      string
}

// path is the file this modal would create: the typed name, in the directory
// the app is working in.
func (m Model) path() string {
	return filepath.Join(m.dir, strings.TrimSpace(m.filename.Value()))
}

func (m Model) Init() tea.Cmd {
	return nil
}

// New walks the user through creating a brand-new compose file: a filename
// (with a sane default and basic validation) and an optional one-service
// seed. Esc cancels the whole flow at any point - the file is never
// half-created.
//
// This is the bootstrap flow for a directory with no compose file in it. dir
// is the directory to create it in - the same one the app resolved in and
// found nothing, so the file it writes is the file the reload afterwards
// picks up.
func New(dir string) tea.Model {
	filename := textinput.New()
	filename.Placeholder = "compose.yaml"
	filename.SetWidth(40)
	filename.SetValue("compose.yaml")
	filename.CursorEnd()
	filename.Focus()

	return Model{
		step:     stepFilename,
		dir:      dir,
		filename: filename,
	}
}
