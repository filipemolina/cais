import { BoxRenderable, type CliRenderer } from "@opentui/core";
import DashboardLeftPane from "./DashboardLeftPane";
import DashboardRightPane from "./DashboardRightPane";

async function DashboardBody(renderer: CliRenderer): Promise<BoxRenderable> {
  const body = new BoxRenderable(renderer, {
    id: "body",
    flexDirection: "row",
    height: "100%",
    gap: 2,
  });

  const leftPane = DashboardLeftPane(renderer);
  const rightPane = await DashboardRightPane(renderer);

  body.add(leftPane);
  body.add(rightPane);

  return body;
}

export default DashboardBody;
