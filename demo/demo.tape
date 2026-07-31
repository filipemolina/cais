# VHS tape for the Stack Stitcher README demo.
# Install VHS first: https://github.com/charmbracelet/vhs (needs ttyd + ffmpeg too).
# Requires the binary on PATH: `make build` installs it to $(go env GOPATH)/bin.
# Record from the repo root with:  vhs demo/demo.tape
#
# Then shrink it. VHS writes every frame in full at 25fps, and this tape comes
# out around 12MB raw — unacceptable on a file the README loads on every visit.
# Three passes together bring it to ~2.4MB with the text still legible: drop
# duplicate frames (mpdecimate), halve the rate to 10fps, and render at 900px
# wide, which is roughly what GitHub displays a README image at anyway:
#
#   FILT="mpdecimate,fps=10,scale=900:-1:flags=lanczos"
#   ffmpeg -i demo/demo.gif -vf "$FILT,palettegen=max_colors=48" /tmp/pal.png
#   ffmpeg -i demo/demo.gif -i /tmp/pal.png \
#     -lavfi "$FILT[x];[x][1:v]paletteuse=dither=none" \
#     -fps_mode vfr /tmp/demo.gif && mv /tmp/demo.gif demo/demo.gif
#
# Recording larger and scaling down beats recording at 900px directly: the
# keybinding bar wraps below roughly 130 columns (a known open item), and at
# 900px of terminal it wraps in every frame.
#
# This starts real containers from demo/fixtures/compose.yaml, so Docker has to
# be running. The tape stops them again before it quits; anything it misses is
# cleaned up by:  cd demo/fixtures && docker compose down -v

Output demo/demo.gif

Set Shell "bash"
Set FontSize 15
Set Width 1200
Set Height 660
Set Theme "Dracula"
# A blinking cursor changes pixels twice a second whether or not anything
# happened, and every one of those ticks is a frame the GIF has to carry.
# Holding it still is worth several megabytes on a recording this long.
Set CursorBlink false
Set TypingSpeed 60ms

# Same throwaway config dir the screenshots use: the recording should not
# depend on, or overwrite, the recorder's own theme.
Hide
Type "mkdir -p /tmp/stitch-demo-config/stack-stitcher && printf 'theme: stitcher-dark\n' > /tmp/stitch-demo-config/stack-stitcher/config.yaml && export XDG_CONFIG_HOME=/tmp/stitch-demo-config && clear"
Enter
Sleep 1s
Show

Type "cd demo/fixtures && stitch"
Enter
Sleep 2500ms

# Groups page: the cursor lands on "downloads" — four *arr services that
# belong together. The details panel follows the cursor, no select step.
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

# Edit the service's compose fragment in place — real YAML, validated live.
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

# Back to the group, and stop all four again.
Type "1"
Sleep 800ms
Tab
Sleep 500ms
Type "t"
Sleep 6s

# Every key, on one screen.
Type "?"
Sleep 2500ms
Escape
Sleep 500ms

Type "q"
Sleep 1s
