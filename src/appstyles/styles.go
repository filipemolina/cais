package appstyles

import (
	"charm.land/lipgloss/v2"
)

const primaryColor = "#7D56F4"
const primaryFontColor = "#FAFAFA"

var DocStyle = lipgloss.NewStyle()

var PrimaryColor = lipgloss.Color(primaryColor)
var BackgroundColor = lipgloss.Darken(lipgloss.Color(primaryColor), 0.5)
var FocusedBackgroundColor = PrimaryColor
var PrimaryFontColor = lipgloss.Color(primaryFontColor)
var SecondaryFontColor = lipgloss.Darken(PrimaryFontColor, 0.2)
