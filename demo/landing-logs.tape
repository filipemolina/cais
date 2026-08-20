# VHS tape for the Cais landing page logs screenshot (light theme).
# Same framing as demo/landing.tape: 1440x810, padding 24, cais-day theme.
#
# The stack in demo/fixtures has to be fully up first, so the logs overlay
# shows real output:
#
#   cd demo/fixtures && docker compose --profile media --profile home --profile infra up -d
#
# Then record from the repo root with:  vhs demo/landing-logs.tape
# The .gif this writes is a byproduct; only the .png is used.

Output demo/landing-logs.gif

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

# --- Logs overlay: a running service's live docker compose output ----------
# (Four Downs select homeassistant on page 1, same as the service shot in
# landing.tape. The sidebar paginates at 7 entries, and staying on page 1 is
# deliberate so the shot shows the full service list.)
Type "2"
Sleep 1s
Down
Down
Down
Down
Sleep 2s
Type "l"
Sleep 3s
Screenshot demo/landing-logs.png
Escape
Sleep 500ms

Type "q"
Sleep 1s