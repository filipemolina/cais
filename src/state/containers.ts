import { $ } from "bun";
import type Config from "./config";
import type { DockerContainerInfo } from "../types/common";
import { CliRenderer, TextRenderable, type BoxRenderable } from "@opentui/core";

export default class Containers {
  private config: Config;

  constructor(config: Config) {
    this.config = config;
  }

  async getRunningContainers() {
    const dockerJson = await $`docker ps --format json | jq -s`.text();

    try {
      // All containers running on this Docker instance
      const runningContainers = JSON.parse(dockerJson) as DockerContainerInfo[];

      // The service names defined on the compose.yaml file
      const configServiceNames = this.config.getServiceNames();

      return configServiceNames.reduce((acc, serviceName) => {
        const runningServiceIndex = runningContainers.findIndex(
          (container) => container.Names === serviceName,
        );

        if (
          runningServiceIndex > -1 &&
          runningContainers[runningServiceIndex]
        ) {
          acc.push(runningContainers[runningServiceIndex]);
        }

        return acc;
      }, [] as DockerContainerInfo[]);
    } catch (error) {
      console.error(error);

      return [];
    }
  }
}
