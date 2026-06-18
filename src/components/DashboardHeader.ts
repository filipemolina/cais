import {
  ASCIIFontRenderable,
  BoxRenderable,
  type CliRenderer,
} from "@opentui/core";
import theme from "../theme";

function DashboardHeader(renderer: CliRenderer): BoxRenderable {
  const header = new BoxRenderable(renderer, {
    id: "header",
    paddingY: 1,
    paddingX: 2,
    height: "auto",
    flexShrink: 0,
    borderColor: theme.colors.primary,
    borderStyle: "rounded",
  });

  const title = new ASCIIFontRenderable(renderer, {
    id: "title",
    text: "STACK STITCHER",
    font: "tiny",
    color: theme.colors.primary,
  });

  header.add(title);

  return header;
}

export default DashboardHeader;
