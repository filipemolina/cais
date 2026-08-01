package cmds

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/filipemolina/stack-stitcher/src/utils"

	tea "charm.land/bubbletea/v2"
	"github.com/compose-spec/compose-go/v2/types"
)

// OpenServiceEditorMsg asks AppModel to open one service for editing in the
// user's $EDITOR. The panel emits this rather than the command itself,
// because which compose file is loaded is AppModel's business.
type OpenServiceEditorMsg struct {
	ServiceName string
}

// RequestInlineEditMsg asks AppModel to prepare the YAML fragment for one
// service so the panel can edit it inline. AppModel answers with
// InlineEditReadyMsg.
type RequestInlineEditMsg struct {
	ServiceName string
}

// InlineEditReadyMsg carries the YAML fragment the panel should put into the
// inline editor. Err means the fragment could not be extracted; the panel
// is not yet in edit mode and should not enter it.
//
// Select, when set, is the service the panel should adopt as m.service
// before entering edit mode. nil for the ordinary path ('e' on an
// already-selected service, where the panel's current selection is already
// right); set when the caller cannot assume a prior SetSelectedServiceMsg
// has already reached the panel - a service AddServiceModal just created,
// before the reload it also kicks off can be guaranteed to have landed:
// tea.Batch makes no ordering promises between sibling commands, so
// selection and edit-readiness have to travel in the same message to be
// atomic.
type InlineEditReadyMsg struct {
	ServiceName string
	Fragment    []byte
	Err         error
	Select      *types.ServiceConfig
}

// RequestSaveServiceMsg asks AppModel to save an edited service fragment
// back to the compose file. AppModel applies it and answers with
// ServiceSavedMsg.
type RequestSaveServiceMsg struct {
	ServiceName string
	Fragment    []byte
}

// ServiceSavedMsg reports the outcome of an inline save. The panel is the
// one that keeps the editor open, so the error belongs inline there; the
// banner is not set for this message.
type ServiceSavedMsg struct {
	ServiceName string
	Err         error
}

// CancelInlineEditMsg tells the panel to abandon inline editing without
// saving. Emitted by the follow-up of the discard-changes confirmation.
type CancelInlineEditMsg struct{}

// ServiceEditedMsg reports the outcome of the $EDITOR path. Err covers
// everything that can go wrong between opening the editor and writing the
// file; the compose file is untouched whenever it is set.
type ServiceEditedMsg struct {
	ServiceName string
	Err         error
}

// OpenServiceEditor asks AppModel to open serviceName for editing in $EDITOR.
func OpenServiceEditor(serviceName string) tea.Cmd {
	return func() tea.Msg {
		return OpenServiceEditorMsg{ServiceName: serviceName}
	}
}

// RequestInlineEdit asks AppModel to prepare a service fragment for inline
// editing.
func RequestInlineEdit(serviceName string) tea.Cmd {
	return func() tea.Msg {
		return RequestInlineEditMsg{ServiceName: serviceName}
	}
}

// RequestSaveService asks AppModel to save an inline-edited service fragment.
func RequestSaveService(serviceName string, fragment []byte) tea.Cmd {
	return func() tea.Msg {
		return RequestSaveServiceMsg{ServiceName: serviceName, Fragment: fragment}
	}
}

// CancelInlineEdit is the follow-up command for the discard-changes confirm
// modal; the panel exits edit mode when it receives the message.
func CancelInlineEdit() tea.Cmd {
	return func() tea.Msg {
		return CancelInlineEditMsg{}
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
