package cmds

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/filipemolina/stack-stitcher/src/utils"

	tea "charm.land/bubbletea/v2"
)

// OpenServiceEditorMsg asks AppModel to open one service for editing. The
// panel emits this rather than the command itself, because which compose
// file is loaded is AppModel's business.
type OpenServiceEditorMsg struct {
	ServiceName string
}

// ServiceEditedMsg reports the outcome of an edit. Err covers everything
// that can go wrong between opening the editor and writing the file; the
// compose file is untouched whenever it is set.
type ServiceEditedMsg struct {
	ServiceName string
	Err         error
}

// OpenServiceEditor asks AppModel to open serviceName for editing.
func OpenServiceEditor(serviceName string) tea.Cmd {
	return func() tea.Msg {
		return OpenServiceEditorMsg{ServiceName: serviceName}
	}
}

// EditService writes the service's YAML to a scratch file, hands it to the
// user's editor, and splices whatever comes back into the compose file.
//
// A failure at any point leaves the compose file exactly as it was. There
// is deliberately no retry loop: the error is reported, the app is in an
// ordinary state, and pressing the key again is the retry. Being unable to
// leave without first fixing your text is worse than losing it.
func EditService(fileName string, serviceName string) tea.Cmd {
	fragment, err := utils.ExtractServiceFragment(fileName, serviceName)
	if err != nil {
		return failedEdit(serviceName, err)
	}

	// Beside the compose file rather than in a system temp directory: the
	// editor shows the user the path it is editing, and one next to their
	// project is far less alarming than one in /tmp. It is removed on every
	// path out.
	scratch, err := os.CreateTemp(filepath.Dir(fileName), fmt.Sprintf(".%s.*.yaml", serviceName))
	if err != nil {
		return failedEdit(serviceName, fmt.Errorf("failed creating a file to edit: %w", err))
	}

	scratchName := scratch.Name()
	if _, err := scratch.Write(fragment); err != nil {
		scratch.Close()
		os.Remove(scratchName)

		return failedEdit(serviceName, fmt.Errorf("failed writing the file to edit: %w", err))
	}
	if err := scratch.Close(); err != nil {
		os.Remove(scratchName)

		return failedEdit(serviceName, fmt.Errorf("failed writing the file to edit: %w", err))
	}

	return tea.ExecProcess(utils.EditorCommand(scratchName), func(execErr error) tea.Msg {
		defer os.Remove(scratchName)

		if execErr != nil {
			return ServiceEditedMsg{ServiceName: serviceName, Err: execErr}
		}

		edited, err := os.ReadFile(scratchName)
		if err != nil {
			return ServiceEditedMsg{ServiceName: serviceName, Err: fmt.Errorf("failed reading the edited service: %w", err)}
		}

		// Quitting the editor without saving is the cancel, so an unchanged
		// file must write nothing - not even a semantically identical
		// rewrite, which would still touch the compose file's mtime.
		if bytes.Equal(edited, fragment) || len(bytes.TrimSpace(edited)) == 0 {
			return ServiceEditedMsg{ServiceName: serviceName}
		}

		return ServiceEditedMsg{
			ServiceName: serviceName,
			Err:         utils.ApplyServiceFragment(fileName, serviceName, edited),
		}
	})
}

func failedEdit(serviceName string, err error) tea.Cmd {
	return func() tea.Msg {
		return ServiceEditedMsg{ServiceName: serviceName, Err: err}
	}
}
