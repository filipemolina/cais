import {
  ASCIIFontRenderable,
  BoxRenderable,
  type RenderContext,
} from "@opentui/core";
import theme from "../theme";

function MainTitle(renderer: RenderContext) {
  const wrapper = new BoxRenderable(renderer, {
    width: "100%",
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "center",
  });

  const title = new ASCIIFontRenderable(renderer, {
    id: "title",
    text: "STACK STITCHER",
    font: "tiny",
    color: theme.colors.primary,
  });

  wrapper.add(title);

  return wrapper;
}

export default MainTitle;
