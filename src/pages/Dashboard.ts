import { BoxRenderable, CliRenderer, KeyEvent } from "@opentui/core";
import DashboardHeader from "../components/DashboardHeader";
import DashboardBody from "../components/DashboardBody";
import DashboardFooter from "../components/DashboardFooter";
import AppState from "../state/appState";

function renderDashboard(renderer: CliRenderer, wrapper: BoxRenderable) {
  for (const child of wrapper.getChildren()) {
    if (child.id) child.destroyRecursively();
  }

  const header = DashboardHeader(renderer);
  const body = DashboardBody(renderer);
  const footer = DashboardFooter(renderer);

  wrapper.add(header);
  wrapper.add(body);
  wrapper.add(footer);
}

function registerKeyEvents(renderer: CliRenderer, wrapper: BoxRenderable) {
  const state = AppState.getState();

  renderer.keyInput.on("keypress", (key: KeyEvent) => {
    switch (key.name) {
      case "up":
        state.navigation.navigateUp();
        renderDashboard(renderer, wrapper);
        break;
      case "down":
        state.navigation.navigateDown();
        renderDashboard(renderer, wrapper);
        break;
      default:
    }
  });
}

function Dashboard(renderer: CliRenderer): BoxRenderable {
  const wrapper = new BoxRenderable(renderer, {
    id: "wrapper",
  });

  renderDashboard(renderer, wrapper);
  registerKeyEvents(renderer, wrapper);

  return wrapper;
}

export default Dashboard;
