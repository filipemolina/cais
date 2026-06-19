type DockerComposeFile = {
  name?: string;
  services: Dict<Dict<object>>;
};

const CONTAINER_HEALTH_STATUSES = ["starting", "healthy", "unhealthy"] as const;
type DockerContainerHealthStatus = (typeof CONTAINER_HEALTH_STATUSES)[number];

const CONTAINER_STATES = [
  "created",
  "running",
  "paused",
  "restarting",
  "exited",
  "removing",
  "dead",
] as const;
type DockerContainerState = (typeof CONTAINER_STATES)[number];

type DockerContainerInfo = {
  Command: string;
  CreatedAt: string;
  HealthStatus: DockerContainerHealthStatus;
  ID: string;
  Image: string;
  Labels: string;
  LocalVolumes: string;
  Mounts: string;
  Names: string;
  Networks: string;
  Platform: {
    architecture: string;
    os: string;
  };
  Ports: string;
  RunningFor: string;
  Size: string;
  State: DockerContainerState;
  Status: string;
};

export {
  CONTAINER_HEALTH_STATUSES,
  CONTAINER_STATES,
  type DockerComposeFile,
  type DockerContainerInfo,
  type DockerContainerHealthStatus,
  type DockerContainerState,
};
