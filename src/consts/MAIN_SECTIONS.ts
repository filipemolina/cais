const MAIN_SECTIONS = ["Home", "Groups"] as const;

type Section = (typeof MAIN_SECTIONS)[number];

export { MAIN_SECTIONS, type Section };
