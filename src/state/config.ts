import type { ComposeFileName } from "../consts/COMPOSE_FILE_NAMES";
import lookForComposeFile from "../utils/lookForComposeFile";

export default class Config {
  fileName: ComposeFileName | null = null;
  file: Bun.BunFile | null = null;
  fileContents: string = "";
  object: unknown | null = null;

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
        this.object = Bun.YAML.parse(this.fileContents);
        return true;
      } catch (error) {
        console.error("Failed to parse YAML:", error);
      }
    }

    return false;
  }
}
