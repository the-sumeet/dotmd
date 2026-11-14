// YAML frontmatter is not part of the Tiptap document. It is split off before
// parsing and re-attached verbatim on serialization so notes round-trip losslessly.

// Frontmatter must open the file with `---`, be either empty or start with a
// `key:` line, and close with `---`. Requiring the key line keeps a note that
// merely begins with a horizontal rule from being mistaken for frontmatter.
const FRONTMATTER_RE =
  /^---[ \t]*\r?\n(?:[A-Za-z0-9_][\w-]*[ \t]*:.*\r?\n(?:.*\r?\n)*?)?---[ \t]*(?:\r?\n|$)/;

export interface SplitMarkdown {
  frontmatter: string;
  body: string;
}

export function splitFrontmatter(markdown: string): SplitMarkdown {
  const match = markdown.match(FRONTMATTER_RE);
  if (!match) return { frontmatter: "", body: markdown };
  return {
    frontmatter: match[0].replace(/\r?\n$/, ""),
    body: markdown.slice(match[0].length).replace(/^\r?\n/, ""),
  };
}

export function joinFrontmatter(frontmatter: string, body: string): string {
  const text = body.endsWith("\n") ? body : `${body}\n`;
  return frontmatter ? `${frontmatter}\n\n${text}` : text;
}
