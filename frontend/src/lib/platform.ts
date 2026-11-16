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

/** True for Cmd (macOS) / Ctrl (elsewhere) with no Alt, on a key or a click. */
export function isMod(e: KeyboardEvent | MouseEvent): boolean {
  return (isMac ? e.metaKey : e.ctrlKey) && !e.altKey;
}

/**
 * The key a shortcut was pressed with, lowercased: "p", "/", ",", … A typed
 * ASCII character is used as is, so ⌘P follows the user's layout (Dvorak,
 * AZERTY) and ⌘/ works where "/" needs Shift (German). Only layouts that
 * type something else (Cyrillic, Greek, …) fall back to the physical US key,
 * so no Latin layout gets a shortcut on two keys.
 */
export function shortcutKey(e: KeyboardEvent): string {
  const key = e.key.toLowerCase();
  if (/^[\x20-\x7e]$/.test(key)) return key;
  if (/^Key[A-Z]$/.test(e.code)) return e.code.slice(3).toLowerCase();
  if (e.code === "Slash") return "/";
  if (e.code === "Comma") return ",";
  return key;
}

/** True while a modal dialog (not the phone sidebar sheet) is open. */
export function dialogOpen(): boolean {
  return document.querySelector('[data-slot="dialog-content"]') !== null;
}
