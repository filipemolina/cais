import {
  BoxRenderable,
  InputRenderable,
  TextRenderable,
  type RenderContext,
} from "@opentui/core";

function ComposeFileInput(renderer: RenderContext): BoxRenderable {
  const wrapper = new BoxRenderable(renderer, {
    width: "100%",
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "center",
    gap: 1,
  });

  const label = new TextRenderable(renderer, {
    content: "Compose File Path:",
    fg: "#FFFFFF",
  });

  const input = new InputRenderable(renderer, {
    id: "input",
    width: 30,
    backgroundColor: "#49244F",
    textColor: "#FFFFFF",
  });

  wrapper.add(label);
  wrapper.add(input);

  return wrapper;
}

export default ComposeFileInput;
