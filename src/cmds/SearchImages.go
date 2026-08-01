package cmds

import (
	"github.com/filipemolina/stack-stitcher/src/utils"

	tea "charm.land/bubbletea/v2"
)

// SearchImagesMsg carries a search result back to addservicemodal.
// Generation must match the component's currently-in-flight generation
// counter or the message is stale and must be discarded silently (D3) -
// the component checks this, not this file.
type SearchImagesMsg struct {
	Generation int
	Results    []utils.ImageResult
	Err        error
}

// SearchImages runs a Docker Hub search tagged with generation, so the
// caller can tell a late-arriving result from the current one apart from
// every earlier, superseded search (D3).
func SearchImages(term string, limit int, generation int) tea.Cmd {
	return func() tea.Msg {
		results, err := utils.SearchImages(term, limit)
		return SearchImagesMsg{Generation: generation, Results: results, Err: err}
	}
}
