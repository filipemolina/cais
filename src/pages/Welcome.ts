import { BoxRenderable, CliRenderer } from "@opentui/core";

import MainTitle from "../components/MainTitle";
import ComposeFileInput from "../components/ComposeFileInput";
import theme from "../theme";

function Welcome(renderer: CliRenderer): BoxRenderable {
  const pageWrapper = new BoxRenderable(renderer, {
    id: "pageWrapper",
    borderColor: theme.colors.primary,
    borderStyle: "rounded",
    flexDirection: "column",
    justifyContent: "center",
    alignItems: "center",
    flexGrow: 1,
    width: "100%",
    gap: 1,
  });

  const mainTitle = MainTitle(renderer);
  const composeFileInput = ComposeFileInput(renderer);

  pageWrapper.add(mainTitle);
  pageWrapper.add(composeFileInput);

  composeFileInput.getRenderable("input")?.focus();

  return pageWrapper;
}

export default Welcome;
