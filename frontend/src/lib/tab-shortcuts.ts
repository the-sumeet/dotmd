export type TabAction =
  | { type: "step"; step: 1 | -1 }
  | { type: "select"; index: number };

/** Maps a keydown to the tab action it triggers, or null if it isn't a tab shortcut. */
export function tabShortcut(e: KeyboardEvent): TabAction | null {
  const mod = e.metaKey || e.ctrlKey;

  // Ctrl+Tab / Ctrl+Shift+Tab, as in browsers and VS Code.
  if (e.ctrlKey && !e.metaKey && !e.altKey && e.key === "Tab") {
    return { type: "step", step: e.shiftKey ? -1 : 1 };
  }
  // Ctrl+PageDown / Ctrl+PageUp.
  if (e.ctrlKey && !e.metaKey && !e.altKey && !e.shiftKey) {
    if (e.key === "PageDown") return { type: "step", step: 1 };
    if (e.key === "PageUp") return { type: "step", step: -1 };
  }
  // Cmd+Shift+] / Cmd+Shift+[ (macOS). Shift changes `key`, so match the physical key.
  if (e.metaKey && e.shiftKey && !e.ctrlKey && !e.altKey) {
    if (e.code === "BracketRight") return { type: "step", step: 1 };
    if (e.code === "BracketLeft") return { type: "step", step: -1 };
  }
  // Cmd/Ctrl+1…8 pick a tab, 9 the last one.
  if (mod && !e.shiftKey && !e.altKey && /^[1-9]$/.test(e.key)) {
    const n = Number(e.key);
    return { type: "select", index: n === 9 ? -1 : n - 1 };
  }
  return null;
}
