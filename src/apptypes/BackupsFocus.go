package apptypes

// BackupsFocus names which half of the Backups page the arrows are driving.
//
// It lives here rather than in either panel because four packages have to
// agree on it: AppModel owns the value and tab moves it, cmds carries it to
// the panels, keys reads it to decide whether the footer says "navigate" or
// "scroll", and the two panels route their own keys by it.
//
// The Backups page is the only page with focus. Home and Services gave theirs
// up because their two panels are two views of one selection (see
// docs/DESIGN.md); here the panels are a cursor over metadata and a scrolling
// file, which are genuinely two things to drive.
type BackupsFocus int

const (
	// BackupsList is the zero value because the page opens on the list: it is
	// the panel with the cursor, and the preview has nothing in it until the
	// cursor picks a row.
	BackupsList BackupsFocus = iota
	BackupsPreview
)

// Toggled is the other half. With exactly two stops, tab and shift+tab are
// the same move, so both call this.
func (f BackupsFocus) Toggled() BackupsFocus {
	if f == BackupsList {
		return BackupsPreview
	}

	return BackupsList
}
