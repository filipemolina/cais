package apptypes

type CurrentPage int

const (
	DashboardPage CurrentPage = iota
	GroupsPage
	ComposeFilesPage
	SettingsPage
)

var currentPageTitles = map[CurrentPage]string{
	DashboardPage:    "Dashboard",
	GroupsPage:       "Groups",
	ComposeFilesPage: "Compose Files",
	SettingsPage:     "Settings",
}

func (c CurrentPage) String() string {
	return currentPageTitles[c]
}
