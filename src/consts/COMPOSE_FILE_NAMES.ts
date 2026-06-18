// TODO: Work with override files (compose.override.yml)

// The order here matters. This is the order of priority in which
// Docker compose loads the files.
export default [
  "compose.yaml",
  "compose.yml",
  "docker-compose.yaml",
  "docker-compose.yml",
] as const;
