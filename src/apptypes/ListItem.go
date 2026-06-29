package apptypes

type ListItem struct {
	ItemTitle, ItemDesc string
}

func (i ListItem) Title() string       { return i.ItemTitle }
func (i ListItem) Description() string { return i.ItemDesc }
func (i ListItem) FilterValue() string { return i.ItemTitle }
