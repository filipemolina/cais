import type { DockerComposeFile } from "../types/common";

export default function getEmptyConfigObject(): DockerComposeFile {
  return {
    name: "",
    services: {},
  };
}
