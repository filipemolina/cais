# VHS tape for the Stack Stitcher README demo.
# Install VHS first: https://github.com/charmbracelet/vhs (needs ttyd + ffmpeg too).
# Record from the repo root with:  vhs demo/demo.tape
# Requires the binary on PATH: make build && sudo mv dist/stack-stitcher /usr/local/bin/

Output demo/demo.gif

Set Shell "bash"
Set FontSize 18
Set Width 1200
Set Height 700
Set Theme "Dracula"
Set TypingSpeed 60ms

Type "cd demo/fixtures && stack-stitcher"
Enter
Sleep 1s

# Home page: focus starts on the profiles list. Select "core".
Space
Sleep 1s

# Tab onto the details panel and start the whole profile together.
Tab
Sleep 500ms
Type "s"
Sleep 4s

# Stop it again.
Type "t"
Sleep 1s

# Tab onto the Main Menu, switch to the Dashboard page.
Tab
Sleep 500ms
Right
Sleep 500ms

# Tab onto the services list, select "app", tab onto its details panel,
# and start just that one service.
Tab
Sleep 500ms
Space
Sleep 500ms
Tab
Sleep 500ms
Type "s"
Sleep 2s

Type "q"
Sleep 1s
