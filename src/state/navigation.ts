import type { CliRenderer } from "@opentui/core";
import { MAIN_SECTIONS, type Section } from "../consts/MAIN_SECTIONS";

class Navigation {
  activeSectionIndex: number = 0;

  getActiveSectionName(): Section {
    return MAIN_SECTIONS[this.activeSectionIndex] as Section;
  }

  navigateUp(): void {
    if (this.activeSectionIndex > 0) {
      this.activeSectionIndex -= 1;
    }
  }

  navigateDown(): void {
    if (this.activeSectionIndex < MAIN_SECTIONS.length - 1) {
      this.activeSectionIndex += 1;
    }
  }
}

export default Navigation;
