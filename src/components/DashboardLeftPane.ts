import { BoxRenderable, type CliRenderer } from "@opentui/core";
import theme from "../theme";
import AppState from "../state/appState";
import { MAIN_SECTIONS } from "../consts/MAIN_SECTIONS";
import NavListItem from "./NavListItem";

function DashboardLeftPane(renderer: CliRenderer): BoxRenderable {
  const state = AppState.getState();

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

  MAIN_SECTIONS.forEach((section) =>
    leftPane.add(
      NavListItem(renderer, {
        text: section,
        isActive: state.navigation.getActiveSectionName() === section,
      }),
    ),
  );

  return leftPane;
}

export default DashboardLeftPane;
