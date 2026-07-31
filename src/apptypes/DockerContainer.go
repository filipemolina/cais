package apptypes

type DockerContainer struct {
	Command   string
	CreatedAt string
	// docker compose ps names this field "Health". Without the tag, the
	// field never binds and the HEALTH column reads "-" for every
	// container, healthy ones included.
	HealthStatus string `json:"Health"`
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

	// Runtime stats from `docker stats --no-stream`. These are populated
	// by GetContainerStats, not by docker compose ps.
	MemPerc  string // e.g. "0.07%"
	MemUsage string // e.g. "21.71MiB / 31.02GiB"
	NetIO    string // e.g. "3.22MB / 4.7kB"
	BlockIO  string // e.g. "70.3MB / 43.7MB"
	CPUPerc  string // e.g. "0.00%"
	PIDs     string // e.g. "19"
}
