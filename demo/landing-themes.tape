# VHS tape for the Cais landing page theme-picker screenshot (light theme).
# Light counterpart to demo/screenshots.tape's theme-picker shot. Same framing
# as demo/landing.tape: 1440x810, padding 24, cais-day theme, no window bar.
#
# The stack in demo/fixtures has to be running first:
#
#   cd demo/fixtures && docker compose --profile media --profile home --profile infra up -d
#
# Then record from the repo root with:  vhs demo/landing-themes.tape
# The .gif this writes is a byproduct; only the .png is used.

Output demo/landing-themes.gif

Set Shell "bash"
Set FontSize 16
Set Width 1440
Set Height 810
Set Padding 24
Set Margin 0
Set Theme { "name": "cais-day", "background": "#F1ECE2", "foreground": "#1B3232", "cursor": "#1F5E5E", "selection": "#D9D0C0", "black": "#1B3232", "red": "#B33A3A", "green": "#1E7F4E", "yellow": "#B07933", "blue": "#1F5E5E", "magenta": "#C0243F", "cyan": "#2A7C7C", "white": "#F1ECE2", "brightBlack": "#5A6F75", "brightRed": "#C0243F", "brightGreen": "#2A9A62", "brightYellow": "#C98F45", "brightBlue": "#2A7C7C", "brightMagenta": "#D0405A", "brightCyan": "#3FA9A0", "brightWhite": "#FBF7EF" }
Set CursorBlink false
Set TypingSpeed 60ms

# Same throwaway config dir the other landing tapes use: the recording should
# not depend on, or overwrite, the recorder's own theme.
Hide
Type "mkdir -p /tmp/cais-landing-config/cais && echo 'theme: cais-day' > /tmp/cais-landing-config/cais/config.yaml && export XDG_CONFIG_HOME=/tmp/cais-landing-config && clear"
Enter
Sleep 1s

Type "cd demo/fixtures && cais"
Enter
Sleep 3s
Show

# --- Files: same page shown behind the theme picker in the dark version ----
Type "3"
Sleep 1500ms
Down
Down
Down
Down
Sleep 1s

# --- The theme picker --------------------------------------------------
# Capture on the cursor's initial row (the active theme, cais-day) - same
# reasoning as screenshots.tape: moving the cursor repaints the Files page
# behind the modal in the hovered theme's palette.
Type "T"
Sleep 1500ms
Screenshot demo/landing-themes.png
Escape
Sleep 500ms
