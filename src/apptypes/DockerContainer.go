package apptypes

type DockerContainer struct {
	Command      string
	CreatedAt    string
	HealthStatus string
	ID           string
	Image        string
	Labels       string
	LocalVolumes string
	Mounts       string
	Names        string
	Networks     string
	Platforms    struct {
		architecture string
		os           string
	}
	Ports      string
	RunningFor string
	Service    string
	Size       string
	State      string
	Status     string
}
