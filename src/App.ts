import { type CliRenderer } from "@opentui/core";
import Welcome from "./pages/Welcome";
import Dashboard from "./pages/Dashboard";

import AppState from "./state/appState";

async function App(renderer: CliRenderer) {
  const state = AppState.getState();

  const wasComposeFileLoaded = await state.loadConfiguration();

  if (wasComposeFileLoaded) {
    renderer.root.add(Dashboard(renderer));
  } else {
    renderer.root.add(Welcome(renderer));
  }
}

export default App;
