import type COMPOSE_FILE_NAMES from "../consts/COMPOSE_FILE_NAMES";
import lookForComposeFile from "../utils/lookForComposeFile";
import Navigation from "./navigation";

type Config = {
  fileName: (typeof COMPOSE_FILE_NAMES)[number] | null;
  file: Bun.BunFile | null;
  fileContents: string;
  object: unknown | null;
  wasLoaded: boolean;
};

export default class AppState {
  // Store the single instance of the class
  private static instance: AppState | null = null;

  config: Config = {
    fileName: null,
    file: null,
    fileContents: "",
    object: null,
    wasLoaded: false,
  };

  navigation: Navigation = new Navigation();

  // Prevent the use of `new AppState`
  private constructor() {}

  // Provide a way to get the singleton instance
  public static getState(): AppState {
    if (!AppState.instance) {
      AppState.instance = new AppState();
    }

    return AppState.instance;
  }

  /**
   * Looks for the compose file in the current directory and tries
   * to parse it and store it on the properties of this class.
   *
   * Returns true if all the steps were successful, false otherwise.
   * @returns {Promise<boolean>}
   */
  async loadConfiguration(): Promise<boolean> {
    const { file, fileName } = await lookForComposeFile();

    this.config.file = file;
    this.config.fileName = fileName;

    if (this.config.file) {
      this.config.fileContents = await this.config.file.text();

      try {
        this.config.object = Bun.YAML.parse(this.config.fileContents);
        this.config.wasLoaded = true;

        return true;
      } catch (error) {
        console.error("Failed to parse YAML:", error);
      }
    }

    return false;
  }
}
