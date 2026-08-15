# VHS tape for the Cais landing page demo (light theme).
# Light counterpart to demo/demo.tape: same choreography, but recorded in the
# cais-day app palette on a warm terminal, natively 16:9, and emitted as video
# directly instead of a GIF. demo/demo.tape stays the dark README version.
# Install VHS first: https://github.com/charmbracelet/vhs (needs ttyd + ffmpeg too).
# Requires the binary on PATH: `make build` installs it to $(go env GOPATH)/bin.
#
# Pre-start the stack before recording:
#   cd demo/fixtures && docker compose --profile media --profile home --profile infra up -d
#
# Record from the repo root with:  vhs demo/landing-demo.tape
# The tape hides itself during setup, Show only when the TUI is on screen.
# Recording stops before the quit key, so the return to terminal isn't captured.
#
# This tape writes .mp4 and .webm straight from VHS. No GIF: H.264 costs
# roughly a third of the bytes for better quality, and the landing page embeds
# an HTML video element, so none of demo.tape's mpdecimate / fps / palettegen
# shrink pipeline is needed here.
#
# WARNING: this tape MUTATES DOCKER STATE. It presses "s" to start the home
# group and later "t" to stop all three of its services, so when it finishes
# those services are STOPPED. Record the still screenshots from
# demo/landing.tape BEFORE running this tape, or bring the stack back up
# first, otherwise the stills capture stopped services and red STOPPED badges.
#
# Compressing the master: the raw VHS output is a MASTER, not a deliverable.
# VHS's own encoder writes demo/landing-demo.mp4 at about 3.1 MB and
# demo/landing-demo.webm at about 2.4 MB for this 41.5s recording, far too
# heavy to ship: the landing page has a hard budget of under one second to
# load and a 100/100 Lighthouse requirement, with about 1.6 MB of total
# referenced page weight. Run this pass before copying the file over:
#
#   ffmpeg -y -i demo/landing-demo.mp4 \
#     -vf "fps=12,scale=1152:-2:flags=lanczos" \
#     -c:v libx264 -crf 33 -preset slow -tune stillimage \
#     -pix_fmt yuv420p -an -movflags +faststart \
#     demo/landing-demo-web.mp4
#
# The three choices that are not obvious: fps=12 because a terminal recording
# is mostly duplicate frames and that is where most of the saving comes from;
# scale=1152 because it is the widest the landing page's video container ever
# renders, so anything larger is wasted bytes; -tune stillimage because the
# content is static text.
#
# Measured on this exact recording, so the numbers do not need re-deriving:
#
#   raw VHS mp4                                   3.1  MB
#   raw VHS webm (VP9)                            2.4  MB
#   scale 1152, crf 26, 25fps                     1392 KB
#   scale 1152, crf 30, 25fps                      989 KB
#   scale 1152, crf 34, 25fps                      682 KB
#   VP9, scale 1152, crf 40, 12fps                 793 KB
#   fps 12, scale 1152, crf 33  <-- CHOSEN         569 KB
#
# The chosen setting is weight-neutral against the dark demo.mp4 the page
# ships today (566 KB) while being higher resolution and light themed. VP9 /
# webm was measured and REJECTED: at 793 KB it is larger than the 569 KB H.264
# file, so this pipeline ships H.264 only. The tape's second Output line
# exists only to produce a webm master; the webm is not shipped.
#
# The poster frame for the landing page's video element is the FIRST frame of
# this recording, which deliberately shows the stack fully running. Extract
# it with:
#
#   ffmpeg -y -ss 0 -i demo/landing-demo-web.mp4 -frames:v 1 demo/landing-demo-poster.png

Output demo/landing-demo.mp4
Output demo/landing-demo.webm

Set Shell "bash"
Set FontSize 16
Set Width 1440
Set Height 810
Set Padding 24
Set Margin 0
Set Theme { "name": "cais-day", "background": "#F1ECE2", "foreground": "#1B3232", "cursor": "#1F5E5E", "selection": "#D9D0C0", "black": "#1B3232", "red": "#B33A3A", "green": "#1E7F4E", "yellow": "#B07933", "blue": "#1F5E5E", "magenta": "#C0243F", "cyan": "#2A7C7C", "white": "#F1ECE2", "brightBlack": "#5A6F75", "brightRed": "#C0243F", "brightGreen": "#2A9A62", "brightYellow": "#C98F45", "brightBlue": "#2A7C7C", "brightMagenta": "#D0405A", "brightCyan": "#3FA9A0", "brightWhite": "#FBF7EF" }
Set CursorBlink false
Set TypingSpeed 60ms

# Same throwaway config dir the screenshots use: the recording should not
# depend on, or overwrite, the recorder's own theme. Echo writes the value
# directly, no escapes, so config.yaml parses cleanly.
Hide
Type "mkdir -p /tmp/cais-landing-config/cais && echo 'theme: cais-day' > /tmp/cais-landing-config/cais/config.yaml && export XDG_CONFIG_HOME=/tmp/cais-landing-config && clear"
Enter
Sleep 1s

# Launch the TUI - this is the first frame we show.
Type "cd demo/fixtures && cais"
Enter
Sleep 2500ms
Show

# Groups page: the cursor lands on "home" (first group, alphabetical):
# three home-management services that belong together. The details panel
# follows the cursor, no select step.
Sleep 1500ms
Down
Sleep 1200ms
Down
Sleep 1500ms
Up
Up
Sleep 1200ms

# Start the whole group with one key.
Tab
Sleep 500ms
Type "s"
Sleep 8s

# Tail every service in the group at once.
Type "l"
Sleep 3500ms
Escape
Sleep 800ms

# Digits switch pages: 2 is Services.
Type "2"
Sleep 1200ms
Down
Down
Down
Down
Down
Down
Down
Down
Down
Sleep 2500ms

# Edit the service's compose fragment in place: real YAML, validated live.
Tab
Sleep 500ms
Type "e"
Sleep 1500ms
Down
Down
Down
Down
Sleep 800ms
Escape
Sleep 1s

# Back to the group, and stop all three again.
Type "1"
Sleep 800ms
Tab
Sleep 500ms
Type "t"
Sleep 6s

# Open the Backups page (tab 4): the merged list of every stored copy of the
# compose file and the .env, newest first, with a live preview beside it.
Type "4"
Sleep 2000ms
# Walk down the list so the preview follows, showing the copy beneath the
# newest rather than only the top row.
Down
Sleep 1500ms
Down
Sleep 1500ms
Up
Up
Sleep 1200ms

# Every key, on one screen.
Type "?"
Sleep 2500ms
Escape
Sleep 500ms
