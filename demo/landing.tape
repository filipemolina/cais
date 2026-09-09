# VHS tape for the Cais landing page screenshots (light theme).
# Install VHS first: https://github.com/charmbracelet/vhs (needs ttyd + ffmpeg too).
# Requires the binary on PATH: `make build` installs it to $(go env GOPATH)/bin.
#
# The stack in demo/fixtures has to be fully up first, so the panels show real
# live state (statuses, health, uptime, stats) instead of "stopped" badges:
#
#   cd demo/fixtures && docker compose --profile media --profile home --profile infra up -d
#
# Then record from the repo root with:  vhs demo/landing.tape
# VHS sometimes drops the last screenshot of a tape - re-run if
# demo/landing-backups.png is missing. The .gif this writes is a byproduct;
# only the .png files are used. The terminal theme mirrors the app's cais-day
# palette, and the 1440x810 frames (native 16:9, no window bar) match the
# landing page's own cropping and chrome.

Output demo/landing.gif

Set Shell "bash"
Set FontSize 16
Set Width 1440
Set Height 810
Set Padding 24
Set Margin 0
Set Theme { "name": "cais-day", "background": "#F1ECE2", "foreground": "#1B3232", "cursor": "#1F5E5E", "selection": "#D9D0C0", "black": "#1B3232", "red": "#B33A3A", "green": "#1E7F4E", "yellow": "#B07933", "blue": "#1F5E5E", "magenta": "#C0243F", "cyan": "#2A7C7C", "white": "#F1ECE2", "brightBlack": "#5A6F75", "brightRed": "#C0243F", "brightGreen": "#2A9A62", "brightYellow": "#C98F45", "brightBlue": "#2A7C7C", "brightMagenta": "#D0405A", "brightCyan": "#3FA9A0", "brightWhite": "#FBF7EF" }
Set CursorBlink false
Set TypingSpeed 60ms

# Record against a throwaway config dir holding the theme these shots are
# framed for, so the recording neither depends on nor overwrites whatever
# theme the person recording happens to use. Seed the demo backup store so
# the Backups page (tab 4) has history to show.
Hide
Type "mkdir -p /tmp/cais-landing-config/cais && echo 'theme: cais-day' > /tmp/cais-landing-config/cais/config.yaml && export XDG_CONFIG_HOME=/tmp/cais-landing-config && ./demo/seed-backups.sh && clear"
Enter
Sleep 2s
Show

Type "cd demo/fixtures && cais"
Enter
Sleep 3s

# --- Hero: the overview with the whole group list, everything green --------
Sleep 1s
Screenshot demo/landing-hero.png

# --- Services: homeassistant's config + its live runtime stats --------------
# (Four Downs select homeassistant on page 1. The sidebar paginates at 7
# entries, and staying on page 1 is deliberate so the shot shows the full
# service list instead of an empty sidebar.)
Type "2"
Sleep 1s
Down
Down
Down
Down
Sleep 3s
Screenshot demo/landing-service.png

# --- The inline YAML editor: navidrome's fragment --------------------------
# (Two Downs after the service shot move homeassistant on to kavita and then
# navidrome while the sidebar still has focus. The editor opens with the
# cursor at the end of the fragment; an arrow key while it is open would move
# the list cursor and abandon the edit, so there is no cursor walk here.)
Down
Down
Sleep 1s
Type "e"
Sleep 2s
Screenshot demo/landing-editor.png
Escape
Sleep 1s

# --- Backups: stored compose + .env copies, and the diff -------------------
# The page opens on the newest copy, which is the live file verbatim: its row
# carries the (current) marker and the preview answers with the "Identical to
# the live file" card. Hold a beat, then walk down to the draft beneath, and
# capture there: the shot should show the feature, so the preview in frame is
# the git-style diff against the live file — red for what restoring would
# remove, green for what it would add — with the (current) markers still
# visible on the rows above it.
Type "4"
Sleep 2s
Down
Sleep 1s
Down
Sleep 2500ms
Screenshot demo/landing-backups.png

Type "q"
Sleep 1s
