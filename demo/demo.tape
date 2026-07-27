# VHS tape for the Stack Stitcher README demo.
# Install VHS first: https://github.com/charmbracelet/vhs (needs ttyd + ffmpeg too).
# Requires the binary on PATH: `make build` installs it to $(go env GOPATH)/bin.
# Record from the repo root with:  vhs demo/demo.tape
#
# Then shrink it — VHS writes every frame in full, and at 25fps most of them
# are identical to the one before, which is worth several megabytes on a file
# the README loads on every visit:
#
#   ffmpeg -i demo/demo.gif -vf "mpdecimate,palettegen=max_colors=64" /tmp/pal.png
#   ffmpeg -i demo/demo.gif -i /tmp/pal.png \
#     -lavfi "mpdecimate[x];[x][1:v]paletteuse=dither=none" \
#     -fps_mode vfr /tmp/demo.gif && mv /tmp/demo.gif demo/demo.gif
#
# This starts real containers from demo/fixtures/compose.yaml, so Docker has to
# be running and port 8085 free. The tape stops them again before it quits.

Output demo/demo.gif

Set Shell "bash"
Set FontSize 15
Set Width 960
Set Height 580
Set Theme "Dracula"
# A blinking cursor changes pixels twice a second whether or not anything
# happened, and every one of those ticks is a frame the GIF has to carry.
# Holding it still is worth several megabytes on a recording this long.
Set CursorBlink false
Set TypingSpeed 60ms

Type "cd demo/fixtures && stack-stitcher"
Enter
Sleep 2s

# Groups page: focus starts on the groups list. Select "core".
Space
Sleep 1s

# Tab onto the details panel and start the whole group together.
Tab
Sleep 500ms
Type "s"
Sleep 3500ms

# Live logs for every service in the group, then back.
Type "l"
Sleep 2s
Escape
Sleep 500ms

# Stop the group again.
Type "t"
Sleep 2500ms

# Digits switch pages: 2 is Services.
Type "2"
Sleep 1s

# Select "app", tab onto its details panel, and start just that one service.
Space
Sleep 500ms
Tab
Sleep 500ms
Type "s"
Sleep 2500ms

# Every key, on one screen.
Type "?"
Sleep 2s
Escape
Sleep 500ms

Type "q"
Sleep 1s
