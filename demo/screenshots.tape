# VHS tape for the README screenshots.
# Install VHS first: https://github.com/charmbracelet/vhs (needs ttyd + ffmpeg too).
# Requires the binary on PATH: `make build` installs it to $(go env GOPATH)/bin.
#
# The stack in demo/fixtures has to be running first, so the panels have real
# state to show (statuses, health, uptime, live stats):
#
#   cd demo/fixtures && docker compose --profile media --profile home --profile infra up -d
#
# Then record from the repo root with:  vhs demo/screenshots.tape
# VHS sometimes drops the last screenshot of a tape — re-run if a file is
# missing. The .gif this writes is a byproduct; only the .png files are used.

Output demo/screenshots.gif

Set Shell "bash"
Set FontSize 16
Set Width 1440
Set Height 720
Set Theme "Dracula"
Set CursorBlink false
Set TypingSpeed 60ms

# Record against a throwaway config dir holding the theme these shots are
# framed for, so the recording neither depends on nor overwrites whatever
# theme the person recording happens to use.
Hide
Type "mkdir -p /tmp/stitch-demo-config/stack-stitcher && printf 'theme: cais-dark\\n' > /tmp/stitch-demo-config/stack-stitcher/config.yaml && export XDG_CONFIG_HOME=/tmp/stitch-demo-config && clear"
Enter
Sleep 1s
Show

Type "cd demo/fixtures && cais"
Enter
Sleep 3s

# --- Groups: the media group, running, with its member table --------------
Down
Down
Sleep 2s
Screenshot demo/screenshot-groups.png

# --- Services: navidrome's config + its live runtime stats ----------------
# (Alphabetical list: audiobookshelf, cache, calibre, db, ha, kavita, navidrome, …)
Type "2"
Sleep 1s
Down
Down
Down
Down
Sleep 3s
Screenshot demo/screenshot-service.png

# --- The inline YAML editor -----------------------------------------------
Tab
Sleep 500ms
Type "e"
Sleep 2s
Down
Down
Down
Down
Down
Down
Down
Sleep 1s
Screenshot demo/screenshot-editor.png
Escape
Sleep 1s

# --- Live logs -------------------------------------------------------------
Type "l"
Sleep 4s
Screenshot demo/screenshot-logs.png
Escape
Sleep 1s

# --- Files: the compose file, highlighted ----------------------------------
Type "3"
Sleep 1500ms
Down
Down
Down
Down
Sleep 1s
Screenshot demo/screenshot-files.png

# --- The theme picker ------------------------------------------------------
Type "T"
Sleep 1500ms
Down
Down
Sleep 1500ms
Screenshot demo/screenshot-themes.png
Escape
Sleep 1s

# The `?` overlay is deliberately not captured here: it is taller than the
# 720px this tape records at, so the shot would show it clipped. It needs its
# own tape at a taller Height if it is ever wanted.

Type "q"
Sleep 1s
