import { createCliRenderer } from "@opentui/core";
import lookForComposeFile from "./src/utils/lookForComposeFile";
import App from "./src/App";

const renderer = await createCliRenderer({
  exitOnCtrlC: true,
});

const app = await App(renderer);

renderer.root.add(app);
