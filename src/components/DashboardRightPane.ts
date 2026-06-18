import { BoxRenderable, TextRenderable, type CliRenderer } from "@opentui/core";
import AppState from "../state/appState";
import theme from "../theme";

function DashboardRightPane(renderer: CliRenderer): BoxRenderable {
  const state = AppState.getState();

  const rightPane = new BoxRenderable(renderer, {
    id: "rightPane",
    title: " Right Pane (r) ",
    titleAlignment: "left",
    borderColor: theme.colors.primary,
    borderStyle: "rounded",
    flexDirection: "column",
    width: "100%",
  });

  const configContent = new TextRenderable(renderer, {
    content: state.config.fileContents,
  });

  rightPane.add(configContent);

  return rightPane;
}

export default DashboardRightPane;
