# VHS tape for the new 3-tier background + underline-only nav design.
# Run from the repo root with:  vhs demo/design-preview.tape
# Outputs demo/design-preview.gif and demo/screenshot-*.png

Output demo/design-preview.gif

Set Shell "bash"
Set FontSize 18
Set Width 1200
Set Height 700
Set Theme "Dracula"
Set TypingSpeed 60ms

# Set TERM explicitly (bubble tea needs it for the alternate-screen
# buffer) and run the app with absolute paths. The && chain makes the
# whole thing one typed command so vhs doesn't need a complex setup.
Type "cd /home/filipe/Documents/projects/tui/demo/fixtures && export TERM=xterm-256color && /home/filipe/go/bin/stitch"
Enter
Sleep 4s

# Capture a screenshot of the home page (initial state, focus on
# Groups list - left panel = tier 4, right details = tier 3).
Screenshot demo/screenshot-home.png

# Tab to move focus to the Group Details panel (right = tier 4).
Tab
Sleep 1500ms
Screenshot demo/screenshot-focus-right.png

# Tab back to the Groups list.
Tab
Sleep 1500ms

Type "q"
Sleep 500ms
