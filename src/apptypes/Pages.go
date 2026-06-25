package apptypes

type Page int

const (
	DashboardPage Page = iota
	GroupsPage
	ComposeFilesPage
	SettingsPage
)

var PageTitles = map[Page]string{
	DashboardPage:    "Dashboard",
	GroupsPage:       "Groups",
	ComposeFilesPage: "Compose Files",
	SettingsPage:     "Settings",
}

func (c Page) String() string {
	return PageTitles[c]
}
