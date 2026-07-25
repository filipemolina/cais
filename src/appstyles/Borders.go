package appstyles

import "charm.land/lipgloss/v2"

var ActiveTabBorder = lipgloss.NewStyle().
	BorderStyle(lipgloss.RoundedBorder()).
	BorderForeground(BorderFocus)

var InactiveTabBorder = lipgloss.NewStyle().
	BorderStyle(lipgloss.RoundedBorder()).
	BorderForeground(BorderDefault)
