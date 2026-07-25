package apptypes

// PageTitles is the ordered list of page IDs used for navigation state.
var PageTitles = []string{
	"Home",
	"Dashboard",
	"Compose Files",
	"Settings",
}

// PageLabels maps page IDs to their display labels in the main menu.
// Page IDs are preserved for all state comparisons; only the label changes.
var PageLabels = map[string]string{
	"Home":          "Groups",
	"Dashboard":     "Dashboard",
	"Compose Files": "Files",
	"Settings":      "Settings",
}
