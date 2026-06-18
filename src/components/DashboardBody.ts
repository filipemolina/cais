import { BoxRenderable, type CliRenderer } from "@opentui/core";
import DashboardLeftPane from "./DashboardLeftPane";
import DashboardRightPane from "./DashboardRightPane";

function DashboardBody(renderer: CliRenderer): BoxRenderable {
  const body = new BoxRenderable(renderer, {
    id: "body",
    flexDirection: "row",
    height: "100%",
  });

  const leftPane = DashboardLeftPane(renderer);
  const rightPane = DashboardRightPane(renderer);

  body.add(leftPane);
  body.add(rightPane);

  return body;
}

export default DashboardBody;
