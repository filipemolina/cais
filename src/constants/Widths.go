package constants

// LEFT_PANEL_WIDTH is set to lazydocker's default sidebar width (one-third of
// the terminal) so the service/group list is readable without overwhelming the
// details panel.
const LEFT_PANEL_WIDTH float32 = 0.33

// RIGHT_PANEL_WIDTH is kept for reference only. The right panel is sized as
// the remainder of the body row (see AppModel.calculateBodyLayout) rather than
// as an independent fraction of the terminal, so that
// left + gutter + right == terminal width exactly.
const RIGHT_PANEL_WIDTH float32 = 1 - LEFT_PANEL_WIDTH

const MIN_PANEL_WIDTH = 30

// BODY_GUTTER_WIDTH is the blank tier-2 column rendered between the two body
// panels so they don't touch. It is subtracted from the row before the panels
// are sized, so the gutter never pushes the layout past the terminal width.
const BODY_GUTTER_WIDTH = 2
