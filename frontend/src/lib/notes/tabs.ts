// Open tabs: each tab is a note's root-relative path, and a note is open in
// at most one tab. App.svelte keeps the list; these helpers stay pure.
import { noteTitle } from "./notes";

const TABS_KEY = "dotmd:tabs";

export interface SavedTabs {
  paths: string[];
  /** The active tab, or null if no note was open. */
  active: string | null;
}

/**
 * `tabs` with `path` opened next to the tab at index `at`: replacing it, or
 * inserted after it (at the end when `at` is -1). A note already open keeps
 * its tab, so the list comes back unchanged.
 */
export function withTab(tabs: string[], path: string, at: number, replace: boolean): string[] {
  if (tabs.includes(path)) return tabs;
  if (at < 0 || at >= tabs.length) return [...tabs, path];
  const next = [...tabs];
  next.splice(replace ? at : at + 1, replace ? 1 : 0, path);
  return next;
}

/**
 * The tab to show once `path` closes: the nearest one to its right, else to
 * its left, that isn't closing too (`closing` says which do; just `path` by
 * default).
 */
export function neighborAfterClose(
  tabs: string[],
  path: string,
  closing: (tab: string) => boolean = (tab) => tab === path,
): string | null {
  const i = tabs.indexOf(path);
  if (i < 0) return null;
  const open = (tab: string) => !closing(tab);
  return tabs.slice(i + 1).find(open) ?? tabs.slice(0, i).findLast(open) ?? null;
}

/** `tabs` with the tab at `from` moved to index `to`. */
export function movedTab(tabs: string[], from: number, to: number): string[] {
  const next = [...tabs];
  const [tab] = next.splice(from, 1);
  next.splice(to, 0, tab);
  return next;
}

/**
 * Label for each tab: the note's title, plus its folder when another open
 * note has the same title ("Plan — work").
 */
export function tabLabels(paths: string[]): { title: string; folder: string }[] {
  const counts = new Map<string, number>();
  for (const p of paths) counts.set(noteTitle(p), (counts.get(noteTitle(p)) ?? 0) + 1);
  return paths.map((p) => {
    const title = noteTitle(p);
    const shared = counts.get(title)! > 1;
    // A note at the top of the notes folder has no folder name; say "Notes".
    const folder = shared ? p.split("/").slice(0, -1).pop() || "Notes" : "";
    return { title, folder };
  });
}

export function loadTabs(): SavedTabs {
  try {
    const v = JSON.parse(localStorage.getItem(TABS_KEY) ?? "null");
    const paths = Array.isArray(v?.paths)
      ? v.paths.filter((p: unknown): p is string => typeof p === "string")
      : [];
    const active = typeof v?.active === "string" ? v.active : null;
    return { paths, active };
  } catch {
    return { paths: [], active: null };
  }
}

export function saveTabs(tabs: SavedTabs) {
  try {
    localStorage.setItem(TABS_KEY, JSON.stringify(tabs));
  } catch {
    // Remembered tabs are a convenience; private storage just means none are kept.
  }
}
