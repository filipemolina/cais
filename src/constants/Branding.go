package constants

import "strings"

var lines = []string{
	"[38;2;95;184;175m   _________    _________[0m",
	"[38;2;125;187;168m  / ____/   |  /  _/ ___/[0m",
	"[38;2;155;191;160m / /   / /| |  / / \\__ \\ [0m",
	"[38;2;185;194;153m/ /___/ ___ |_/ / ___/ / [0m",
	"[38;2;215;198;145m\\____/_/  |_/___//____/  [0m",
	"[38;2;245;201;138m                         [0m",}

// LOGO is the large ASCII brand mark. Reserved for a future About modal;
// it is no longer rendered by PanelFrame.
var LOGO = strings.Join(lines, "\n")

// WORDMARK is the one-line badge rendered in the main menu bar.
var WORDMARK = "▌ Cais"

var SLOGAN = "Nothing to sew here."
var APP_NAME = "Cais"
