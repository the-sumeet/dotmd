// The user's settings, kept by the Go settings service (internal/settings)
// in a file in the OS config folder. This module mirrors them as reactive
// state and applies the theme to the page.
import { Service, type State } from "../../bindings/dotmd/internal/settings";

export type Theme = "system" | "light" | "dark";

/** The theme is copied here so it can be applied before Go answers. */
const THEME_KEY = "dotmd:theme";

const darkQuery = window.matchMedia("(prefers-color-scheme: dark)");

function isTheme(v: unknown): v is Theme {
  return v === "system" || v === "light" || v === "dark";
}

function storedTheme(): Theme {
  try {
    const v = localStorage.getItem(THEME_KEY);
    return isTheme(v) ? v : "system";
  } catch {
    return "system";
  }
}

function followSystem() {
  document.documentElement.classList.toggle("dark", darkQuery.matches);
}

/** Show the page in `theme`; "system" follows the OS, live. */
function applyTheme(theme: Theme) {
  darkQuery.removeEventListener("change", followSystem);
  if (theme === "system") {
    followSystem();
    darkQuery.addEventListener("change", followSystem);
  } else {
    document.documentElement.classList.toggle("dark", theme === "dark");
  }
  try {
    localStorage.setItem(THEME_KEY, theme);
  } catch {
    // Only costs a flash of the OS theme on the next launch.
  }
}

/** Apply the last theme used, and follow the OS if it is "System". */
export function applyStoredTheme() {
  applyTheme(storedTheme());
}

class Settings {
  theme = $state<Theme>(storedTheme());
  wideLayout = $state(false);
  /** Absolute path of the notes folder in use. */
  notesFolder = $state("");
  /** False on phones and tablets, where apps can't pick a folder. */
  canChooseFolder = $state(false);
  /**
   * True while notes are in the default folder (~/Documents/dotmd), the only
   * one the Welcome note is added to. False until Go says so, so a failed
   * load never writes into a folder the user picked.
   */
  defaultFolder = $state(false);
  /** The saved notes folder, when it wasn't found at startup. */
  missingFolder = $state("");

  private apply(s: State) {
    this.theme = isTheme(s.theme) ? s.theme : "system";
    this.wideLayout = s.wideLayout;
    this.notesFolder = s.notesFolder;
    this.canChooseFolder = s.canChooseFolder;
    this.defaultFolder = s.defaultFolder;
    this.missingFolder = s.missingFolder;
    applyTheme(this.theme);
  }

  /** Read the settings from Go; on failure the defaults stay. */
  async load() {
    try {
      this.apply(await Service.Get());
    } catch (err) {
      console.error("Failed to load settings", err);
    }
  }

  async setTheme(theme: Theme) {
    await Service.SetTheme(theme);
    this.theme = theme;
    applyTheme(theme);
  }

  async setWideLayout(on: boolean) {
    await Service.SetWideLayout(on);
    this.wideLayout = on;
  }

  /**
   * Ask for a new notes folder and switch to it. Resolves to false if the
   * user cancelled. Go has switched folders by the time this resolves, so
   * callers must drop state from the old folder right away, with no await
   * in between.
   */
  async chooseNotesFolder(): Promise<boolean> {
    const s = await Service.ChooseNotesFolder();
    if (!s) return false;
    this.apply(s);
    return true;
  }
}

export const settings = new Settings();
