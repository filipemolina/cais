package apptypes

type ContainerListItem DockerContainer

func (i ContainerListItem) Title() string       { return i.Names }
func (i ContainerListItem) Description() string { return i.State }
func (i ContainerListItem) FilterValue() string { return i.Names }
