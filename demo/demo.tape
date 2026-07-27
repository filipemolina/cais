# VHS tape for the Stack Stitcher README demo.
# Install VHS first: https://github.com/charmbracelet/vhs (needs ttyd + ffmpeg too).
# Record from the repo root with:  vhs demo/demo.tape
# Requires the binary on PATH: `make build` installs it to $(go env GOPATH)/bin.
#
# This starts real containers from demo/fixtures/compose.yaml, so Docker has to
# be running and port 8085 free. The tape stops them again before it quits.

Output demo/demo.gif

Set Shell "bash"
Set FontSize 18
Set Width 1200
Set Height 700
Set Theme "Dracula"
# 20fps rather than the default 50: the app repaints on keypresses and poll
# ticks, not continuously, so the extra frames are duplicates that only cost
# the README bytes.
Set Framerate 20
Set TypingSpeed 60ms

Type "cd demo/fixtures && stack-stitcher"
Enter
Sleep 2s

# Groups page: focus starts on the groups list. Select "core".
Space
Sleep 1500ms

# Tab onto the details panel and start the whole group together.
Tab
Sleep 500ms
Type "s"
Sleep 4s

# Live logs for every service in the group, then back.
Type "l"
Sleep 2s
Escape
Sleep 1s

# Stop the group again.
Type "t"
Sleep 3s

# Digits switch pages: 2 is Services.
Type "2"
Sleep 1500ms

# Select "app", tab onto its details panel, and start just that one service.
Space
Sleep 500ms
Tab
Sleep 500ms
Type "s"
Sleep 3s
Type "t"
Sleep 2s

# Every key, on one screen.
Type "?"
Sleep 2s
Escape
Sleep 500ms

Type "q"
Sleep 1s
