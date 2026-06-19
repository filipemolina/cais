import { BoxRenderable, TextRenderable, type CliRenderer } from "@opentui/core";
import AppState from "../state/appState";
import theme from "../theme";
import Card from "./Card";

async function DashboardRightPane(
  renderer: CliRenderer,
): Promise<BoxRenderable> {
  const state = AppState.getState();

  const rightPane = new BoxRenderable(renderer, {
    id: "rightPane",
    title: " Right Pane (r) ",
    titleAlignment: "left",
    borderColor: theme.colors.primary,
    borderStyle: "rounded",
    flexDirection: "row",
    width: "100%",
    flexWrap: "wrap",
    paddingX: 1,
  });

  const containers = await state.containers.getRunningContainers();

  for (const container of containers) {
    rightPane.add(
      Card(renderer, {
        title: container.Names,
        description: "",
        footerTitle: container.Status,
      }),
    );
  }

  return rightPane;
}

export default DashboardRightPane;
