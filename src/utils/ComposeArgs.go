package utils

// ComposeFileArgs starts the argument list for a `docker compose` invocation,
// pinning it to composeFile when one is known.
//
// Every docker call in the app goes through this so the UI and the commands
// can never disagree about which file is in play: the panel shows the file the
// app resolved, and `--file` makes docker act on that same one. Before this,
// each side resolved independently - identically, but only because both looked
// in the current directory. A `--file` flag pointing somewhere else would have
// split them, which is why this landed first.
//
// An empty composeFile leaves the flag off and lets docker resolve the file
// itself. That is the bootstrap state (no file loaded yet), where there is
// nothing to pin to and no panel making a claim to contradict.
//
// --file is compose-level, so it belongs between `compose` and the
// subcommand: `docker compose --file X up -d`, never `docker compose up
// --file X`.
func ComposeFileArgs(composeFile string) []string {
	if composeFile == "" {
		return []string{"compose"}
	}

	return []string{"compose", "--file", composeFile}
}
