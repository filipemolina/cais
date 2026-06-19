import type { ComposeFileName } from "../consts/COMPOSE_FILE_NAMES";
import type { DockerComposeFile } from "../types/common";
import getEmptyConfigObject from "../utils/getEmptyConfigObject";
import lookForComposeFile from "../utils/lookForComposeFile";

export default class Config {
  private fileName: ComposeFileName | null = null;
  private file: Bun.BunFile | null = null;
  private fileContents: string = "";
  private configObject: DockerComposeFile = getEmptyConfigObject();

  /**
   * Looks for the compose file in the current directory and tries
   * to parse it and store it on the properties of this class.
   *
   * Returns true if all the steps were successful, false otherwise.
   * @returns {Promise<boolean>}
   */
  async loadConfiguration(): Promise<boolean> {
    const { file, fileName } = await lookForComposeFile();

    this.file = file;
    this.fileName = fileName;

    if (this.file) {
      this.fileContents = await this.file.text();

      try {
        this.configObject = Bun.YAML.parse(
          this.fileContents,
        ) as DockerComposeFile;
        return true;
      } catch (error) {
        console.error("Failed to parse YAML:", error);
      }
    }

    return false;
  }

  getConfigObject() {
    return this.configObject;
  }

  getConfigFileContents() {
    return this.fileContents;
  }

  getConfigFile() {
    return this.file;
  }

  getConfigFileName() {
    return this.fileName;
  }

  getServiceNames() {
    return Object.keys(this.configObject.services);
  }
}
