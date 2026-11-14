/** True if every ")" in `text` closes an earlier "(". */
function balancedParens(text: string): boolean {
  let depth = 0;
  for (const ch of text) {
    if (ch === "(") depth++;
    else if (ch === ")" && --depth < 0) return false;
  }
  return depth === 0;
}

/**
 * Markdown link destination for `href`. One with spaces, angle brackets or
 * unbalanced parentheses (a note named "My Note.md") is wrapped in `<…>`,
 * or it would not parse as a link.
 */
export function markdownHref(href: string): string {
  if (!/[\s<>]/.test(href) && balancedParens(href)) return href;
  return `<${href.replace(/</g, "%3C").replace(/>/g, "%3E")}>`;
}
