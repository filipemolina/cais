import Config from "./config";
import Containers from "./containers";
import Navigation from "./navigation";

export default class AppState {
  // Store the single instance of the class
  private static singleton: AppState | null = null;

  // Prevent the use of `new AppState`
  private constructor() {}

  // Provide a way to get the singleton instance
  public static getState(): AppState {
    if (!AppState.singleton) {
      AppState.singleton = new AppState();
    }

    return AppState.singleton;
  }

  // Main state properties
  config: Config = new Config();
  navigation: Navigation = new Navigation();
  containers: Containers = new Containers(this.config);
}
