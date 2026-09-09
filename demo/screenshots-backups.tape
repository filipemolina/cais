# VHS tape for the README "Backups" screenshot.
# Requires the binary on PATH: `make build` installs it to $(go env GOPATH)/bin.
#
# Record from the repo root with:  vhs demo/screenshots-backups.tape
# The recording seeds demo backups via demo/seed-backups.sh (run first, or
# let this tape's Hide block do it) so the page shows real history, then
# captures demo/screenshot-backups.png.
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

# --- Backups: the list panel and the diff preview ------------------------
# The page opens on the newest copy, which is the live file verbatim: its
# row carries the (current) marker and the preview answers with the
# "Identical to the live file" card. Hold a beat so that state is on the
# recording, then walk down past the live .env copy to the September draft,
# whose preview is a git-style diff against the live file — red for what
# restoring would remove, green for what it would add. The screenshot is
# the diff frame: the diff is the page's answer to "what would restoring
# change?", and the (current) marker still visible on the rows above says
# which copy is which.
Type "4"
Sleep 3s
Down
Sleep 1200ms
Down
Sleep 3s
Screenshot demo/screenshot-backups.png

Type "q"
Sleep 1s
