package cmds

// TagLookupMsg carries the result of a background best-tag lookup on the
// image the confirm stage was pre-filled with (Phase 2B of
// docs/plans/image-search.md D4). It is only ever applied if the image
// field still holds exactly the pre-filled value - if the user has typed
// since, the result is stale and dropped silently.
type TagLookupMsg struct {
	Repo    string
	BestTag string
	Err     error
}
