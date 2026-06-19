// TODO: Work with override files (compose.override.yml)

// The order here matters. This is the order of priority in which
// Docker compose loads the files.
const COMPOSE_FILE_NAMES = [
  "compose.yaml",
  "compose.yml",
  "docker-compose.yaml",
  "docker-compose.yml",
] as const;

type ComposeFileName = (typeof COMPOSE_FILE_NAMES)[number];

export { COMPOSE_FILE_NAMES, type ComposeFileName };
