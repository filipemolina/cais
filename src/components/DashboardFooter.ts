import { BoxRenderable, type CliRenderer } from "@opentui/core";
import theme from "../theme";

function DashboardFooter(renderer: CliRenderer): BoxRenderable {
  const footer = new BoxRenderable(renderer, {
    id: "footer",
    borderColor: theme.colors.primary,
    flexDirection: "row",
    width: "100%",
  });

  return footer;
}

export default DashboardFooter;
