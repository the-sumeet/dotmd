// What kind of target a link's href is: a note path, or an address outside
// the notes. Shared by the editor (autolinking, the link input) and by
// following links, so they always agree.
import { test as isWebAddress } from "linkifyjs";

/** True for hrefs with a scheme (`https:`, `mailto:`, …), which point outside the notes. */
export function isExternalLink(href: string): boolean {
  return /^[a-z][a-z0-9+.-]*:/i.test(href);
}

/**
 * Schemes the app hands to the system. Others (`file:`, app-specific schemes)
 * could launch local programs from a note someone else wrote, so they are refused.
 */
export function canOpenExternally(href: string): boolean {
  return /^(https?|mailto|tel):/i.test(href);
}

/**
 * True for text like "Plan.md" or "../A.md". Linkify reads these as web
 * addresses (".md" is Moldova's domain); an explicit "https://site.md" is still a URL.
 */
export function looksLikeNoteFile(text: string): boolean {
  if (isExternalLink(text)) return false;
  const host = text.split(/[/?#]/)[0];
  return /\.(md|markdown)$/i.test(host);
}

/**
 * The href meant by what the user typed: a bare web address ("example.com",
 * "www.github.com/x") gets "https://"; note paths and everything else are kept.
 */
export function normalizeHref(href: string): string {
  if (isExternalLink(href) || looksLikeNoteFile(href) || /^[./#]/.test(href)) return href;
  return isWebAddress(href, "url") ? `https://${href}` : href;
}
