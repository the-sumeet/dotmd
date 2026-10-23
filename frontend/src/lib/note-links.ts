/**
 * Resolves a link target against the note at `from` (both relative to the data directory).
 * Returns `null` for links that aren't notes: URLs with a scheme, `//host` links and `#anchors`.
 */
export function resolveNoteLink(from: string, href: string): string | null {
  const path = resolveRelative(from, href);
  if (path === null) return null;
  return path.split("/").pop()!.includes(".") ? path : path + ".md";
}

/**
 * The URL the app serves the file `src` points to, for a note at `from`, so images next to a note can be shown.
 * Returns `null` for remote sources (a scheme or `//host`) and anything that isn't a path.
 */
export function resolveAssetUrl(from: string, src: string): string | null {
  const path = resolveRelative(from, src);
  return path === null ? null : "/files/" + encodePath(path);
}

/** Resolves `href` against the note at `from`; `null` if it isn't a relative path to a file. */
function resolveRelative(from: string, href: string): string | null {
  if (/^[a-z][a-z\d+.-]*:/i.test(href) || href.startsWith("//") || href.startsWith("#")) return null;
  // `..` can't climb above the URL root, so the result stays inside the data directory.
  let path: string;
  try {
    path = decodeURIComponent(new URL(href, "file:///" + encodePath(from)).pathname.slice(1));
  } catch {
    return null;
  }
  return path === "" || path.endsWith("/") ? null : path;
}

/** Percent-encodes each segment of a path, so spaces, `#`, `%` and parentheses survive as a link destination. */
function encodePath(path: string): string {
  return path
    .split("/")
    .map((s) => encodeURIComponent(s).replace(/[()]/g, (c) => "%" + c.charCodeAt(0).toString(16).toUpperCase()))
    .join("/");
}

/**
 * The link to `to` from the note at `from` (both relative to the data directory),
 * encoded so it can be pasted straight into `[text](…)`.
 */
export function relativeNotePath(from: string, to: string): string {
  const dir = from.split("/").slice(0, -1);
  const target = to.split("/");
  let common = 0;
  while (common < dir.length && common < target.length - 1 && dir[common] === target[common]) common++;
  return encodePath([...Array(dir.length - common).fill(".."), ...target.slice(common)].join("/"));
}
