/**
 * macOS desktop (not an iPad, which also reports "Mac"). Used to label
 * shortcuts with ⌘ and to clear the window's traffic-light buttons.
 */
export const isMac =
  typeof navigator !== "undefined" &&
  /Mac/.test(navigator.userAgent) &&
  navigator.maxTouchPoints < 2;

/** Label for the primary modifier key: "⌘" on macOS, "Ctrl+" elsewhere. */
export const modKey = isMac ? "⌘" : "Ctrl+";

/** True for Cmd (macOS) / Ctrl (elsewhere) with no Alt. */
export function isMod(e: KeyboardEvent): boolean {
  return (isMac ? e.metaKey : e.ctrlKey) && !e.altKey;
}

/**
 * The key a shortcut was pressed with, lowercased: "p", "/", … The typed
 * character is used when it is a Latin letter or "/", so ⌘P follows the
 * user's layout (Dvorak, AZERTY) and ⌘/ works where "/" needs Shift (German).
 * Otherwise (Cyrillic, Greek, …) it falls back to the physical US key.
 */
export function shortcutKey(e: KeyboardEvent): string {
  const key = e.key.toLowerCase();
  if (/^[a-z/]$/.test(key)) return key;
  if (/^Key[A-Z]$/.test(e.code)) return e.code.slice(3).toLowerCase();
  return e.code === "Slash" ? "/" : key;
}

/** True while a modal dialog (not the phone sidebar sheet) is open. */
export function dialogOpen(): boolean {
  return document.querySelector('[data-slot="dialog-content"]') !== null;
}
