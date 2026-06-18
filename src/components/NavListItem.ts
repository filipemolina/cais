import {
  BoxRenderable,
  TextAttributes,
  TextRenderable,
  type CliRenderer,
} from "@opentui/core";
import theme from "../theme";

type Props = {
  text: string;
  isActive?: boolean;
};

function NavListItem(
  renderer: CliRenderer,
  { text, isActive }: Props,
): BoxRenderable {
  const wrapper = new BoxRenderable(renderer, {
    width: "100%",
    paddingX: 2,
    paddingY: 0,
    backgroundColor: isActive ? theme.colors.primary : undefined,
  });

  const item = new TextRenderable(renderer, {
    content: text,
    attributes: isActive ? TextAttributes.BOLD : undefined,
  });

  wrapper.add(item);

  return wrapper;
}

export default NavListItem;
