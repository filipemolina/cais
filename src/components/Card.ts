import { BoxRenderable, TextRenderable, type CliRenderer } from "@opentui/core";
import theme from "../theme";

type CardProps = {
  title: string;
  description?: string;
  footerTitle?: string;
};

function Card(
  renderer: CliRenderer,
  { title, description, footerTitle }: CardProps,
): BoxRenderable {
  const card = new BoxRenderable(renderer, {
    title: ` ${title} `,
    titleColor: theme.colors.text.primary,
    bottomTitle: ` ${footerTitle} `,
    bottomTitleAlignment: "right",
    flexDirection: "column",
    borderColor: theme.colors.primary,
    borderStyle: "rounded",
    width: 40,
    height: 10,
    margin: 1,
  });

  card.add(
    new TextRenderable(renderer, {
      content: `${description}`,
      fg: theme.colors.text.secondary,
    }),
  );

  return card;
}

export default Card;
