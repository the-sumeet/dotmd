// Thin wrapper over the generated Go bindings for the notes folder, so the
// rest of the app does not depend on binding paths or nullable slices.
import { NotesService } from "../../../bindings/dotmd/internal/notes";
import type { Node, Tag as TagBinding } from "../../../bindings/dotmd/internal/notes";

export type { Node };
export type EntryKind = "note" | "folder";
export type Tag = Omit<TagBinding, "notes"> & { notes: string[] };

export async function loadTree(): Promise<Node[]> {
  return (await NotesService.Tree()) ?? [];
}

export async function loadTags(): Promise<Tag[]> {
  return ((await NotesService.Tags()) ?? []).map((t) => ({ ...t, notes: t.notes ?? [] }));
}

/** Frontmatter tags of a note's content (parsed in Go, no disk access). */
export async function tagsOf(content: string): Promise<string[]> {
  return (await NotesService.TagsOf(content)) ?? [];
}

const byLower = (a: string, b: string) => {
  const x = a.toLowerCase();
  const y = b.toLowerCase();
  return x < y ? -1 : x > y ? 1 : 0;
};

/**
 * The tag list after `path` was saved with tags `names`: the note is removed
 * from tags it no longer has and added to new ones; empty tags are dropped.
 * Matching is case-insensitive, like the Go side.
 */
export function withNoteTags(tags: Tag[], path: string, names: string[]): Tag[] {
  const byName = new Map<string, Tag>();
  for (const t of tags) {
    byName.set(t.name.toLowerCase(), { name: t.name, notes: t.notes.filter((p) => p !== path) });
  }
  for (const name of names) {
    const key = name.toLowerCase();
    let tag = byName.get(key);
    if (!tag) byName.set(key, (tag = { name, notes: [] }));
    if (!tag.notes.includes(path)) tag.notes = [...tag.notes, path].sort(byLower);
  }
  return [...byName.values()]
    .filter((t) => t.notes.length)
    .sort((a, b) => byLower(a.name, b.name));
}

export function readNote(path: string): Promise<string> {
  return NotesService.Read(path);
}

export function writeNote(path: string, content: string): Promise<void> {
  return NotesService.Write(path, content);
}

/** Create an empty note; ".md" is added if missing. Resolves to its path. */
export function createNote(path: string): Promise<string> {
  return NotesService.CreateNote(path);
}

/** Create a folder (and missing parents). Resolves to its path. */
export function createFolder(path: string): Promise<string> {
  return NotesService.CreateFolder(path);
}

export function notesRoot(): Promise<string> {
  return NotesService.Root();
}

/** First note in display order (depth-first), or null if there are none. */
export function firstNote(nodes: Node[]): string | null {
  for (const node of nodes) {
    if (!node.isDir) return node.path;
    const found = firstNote(node.children ?? []);
    if (found) return found;
  }
  return null;
}

/** Display name for a note: file name without its markdown extension. */
export function noteTitle(path: string): string {
  const name = path.split("/").pop() ?? path;
  return name.replace(/\.(md|markdown)$/i, "");
}
