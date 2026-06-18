import {
  ASCIIFontRenderable,
  BoxRenderable,
  type CliRenderer,
} from "@opentui/core";
import theme from "../theme";

function DashboardHeader(renderer: CliRenderer): BoxRenderable {
  const header = new BoxRenderable(renderer, {
    id: "header",
    borderColor: theme.colors.primary,
    borderStyle: "rounded",
    paddingY: 1,
    paddingX: 2,
    height: "auto",
    flexShrink: 0,
  });

  const title = new ASCIIFontRenderable(renderer, {
    id: "title",
    text: "STACK STITCHER",
    font: "tiny",
    color: "#ADADAD",
  });

  header.add(title);

  return header;
}

export default DashboardHeader;
