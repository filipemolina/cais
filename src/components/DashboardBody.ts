import { BoxRenderable, TextRenderable, type CliRenderer } from "@opentui/core";
import theme from "../theme";
import AppState from "../state/appState";

function DashboardBody(renderer: CliRenderer): BoxRenderable {
  const state = AppState.getState();

  const body = new BoxRenderable(renderer, {
    id: "body",
    flexDirection: "row",
    height: "100%",
  });

  const leftPane = new BoxRenderable(renderer, {
    id: "leftPane",
    title: " Left Pane (a) ",
    titleAlignment: "left",
    borderColor: theme.colors.primary,
    borderStyle: "rounded",
    flexDirection: "column",
    width: 30,
    height: "100%",
  });

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
    content: state.configFileContents,
  });

  rightPane.add(configContent);

  body.add(leftPane);
  body.add(rightPane);

  return body;
}

export default DashboardBody;
