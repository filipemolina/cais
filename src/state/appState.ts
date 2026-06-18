import type COMPOSE_FILE_NAMES from "../consts/COMPOSE_FILE_NAMES";
import lookForComposeFile from "../utils/lookForComposeFile";

export default class AppState {
  // Store the single instance of the class
  private static instance: AppState | null = null;

  configFileName: (typeof COMPOSE_FILE_NAMES)[number] | null = null;
  configFile: Bun.BunFile | null = null;
  configFileContents: string = "";
  configObject: unknown | null = null;
  wasConfigObjectLoaded: boolean = false;

  // Prevent the use of `new AppState`
  private constructor() {}

  // Provide a way to get the single instance
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

    this.configFile = file;
    this.configFileName = fileName;

    if (this.configFile) {
      this.configFileContents = await this.configFile.text();

      try {
        this.configObject = Bun.YAML.parse(this.configFileContents);
        this.wasConfigObjectLoaded = true;

        return true;
      } catch (error) {
        console.error("Failed to parse YAML:", error);
      }
    }

    return false;
  }
}
