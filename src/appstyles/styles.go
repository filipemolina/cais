package appstyles

import (
	"charm.land/lipgloss/v2"
)

const primaryColor = "#7D56F4"
const primaryFontColor = "#FAFAFA"

var DocStyle = lipgloss.NewStyle().MarginBottom(1)

var BackgroundColor = lipgloss.Darken(lipgloss.Color(primaryColor), 0.5)
var FocusedBackgroundColor = lipgloss.Color(primaryColor)
var PrimaryFontColor = lipgloss.Color(primaryFontColor)
