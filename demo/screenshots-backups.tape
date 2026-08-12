# VHS tape for the README "Backups" screenshot.
# Requires the binary on PATH: `make build` installs it to $(go env GOPATH)/bin.
#
# Record from the repo root with:  vhs demo/screenshots-backups.tape
# The recording seeds demo backups via demo/seed-backups.sh (run first, or
# let this tape's Hide block do it) so the page shows real history and a
# populated preview, then captures demo/screenshot-backups.png.
#
# VHS Screenshot takes relative paths only — the PNG lands under the directory
# VHS was invoked from (the repo root). Re-run if the file is missing.

Output demo/screenshots-backups.gif

Set Shell "bash"
Set FontSize 16
Set Width 1440
Set Height 720
Set Theme "Dracula"
Set CursorBlink false
Set TypingSpeed 60ms

# Record against a throwaway config dir holding the theme these shots are
# framed for, so the recording neither depends on nor overwrites whatever
# theme the person recording happens to use. Seed the demo backup store so
# the Backups page (tab 4) has history to show.
Hide
Type "mkdir -p /tmp/cais-demo-config/cais && printf 'theme: cais-dark\\n' > /tmp/cais-demo-config/cais/config.yaml && export XDG_CONFIG_HOME=/tmp/cais-demo-config && ./demo/seed-backups.sh && clear"
Enter
Sleep 1s
Show

Type "cd demo/fixtures && cais"
Enter
Sleep 3s

# --- Backups: the merged list of stored compose + .env copies -----------
# The newest copy is selected by default and its preview is shown beside the
# list. Hold a beat so the preview viewport settles, then capture.
Type "4"
Sleep 3s
Screenshot demo/screenshot-backups.png

Type "q"
Sleep 1s
