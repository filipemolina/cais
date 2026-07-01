package appstyles

import (
	"charm.land/lipgloss/v2"
)

// const primaryColor = "62"
const primaryColor = "#BC3FBC"
const primaryFontColor = "#FAFAFA"
const paneColor = "#151520"
const panelBackgroundColor = "#3F3F3F"

var lightDark = lipgloss.LightDark(false)

var DocStyle = lipgloss.NewStyle()

var PrimaryColor = lipgloss.Color(primaryColor)
var ComplementaryColor = lipgloss.Complementary(PrimaryColor)
var PaneColor = lipgloss.Color(paneColor)
var PanelBackgroundColor = lipgloss.Color(panelBackgroundColor)
var SelectedPaneColor = lipgloss.Darken(PrimaryColor, 0.5)
var FocusedPaneColor = lipgloss.Darken(PrimaryColor, 0.7)
var BackgroundColor = lipgloss.Darken(PrimaryColor, 0.5)
var FocusedBackgroundColor = PrimaryColor
var PrimaryFontColor = lipgloss.Color(primaryFontColor)
var SecondaryFontColor = lipgloss.Darken(PrimaryFontColor, 0.2)

var NormalTitle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#dddddd")).
	Background(PrimaryColor).
	Padding(0, 1).
	MarginLeft(2)

var NormalDesc = NormalTitle.
	Foreground(lightDark(lipgloss.Color("#A49FA5"), lipgloss.Color("#777777")))

var SelectedTitle = lipgloss.NewStyle().
	Border(lipgloss.NormalBorder(), false, false, false, true).
	BorderForeground(lightDark(lipgloss.Color("#F793FF"), lipgloss.Color("#AD58B4"))).
	Foreground(lightDark(lipgloss.Color("#EE6FF8"), lipgloss.Color("#EE6FF8"))).
	Padding(0, 0, 0, 1)

var SelectedDesc = SelectedTitle.
	Foreground(lightDark(lipgloss.Color("#F793FF"), lipgloss.Color("#AD58B4")))

var DimmedTitle = lipgloss.NewStyle().
	Foreground(lightDark(lipgloss.Color("#A49FA5"), lipgloss.Color("#777777"))).
	Padding(0, 0, 0, 2) //nolint:mnd

var DimmedDesc = DimmedTitle.
	Foreground(lightDark(lipgloss.Color("#C2B8C2"), lipgloss.Color("#4D4D4D")))
