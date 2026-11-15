// Thin wrapper over the generated Go bindings for the notes folder, so the
// rest of the app does not depend on binding paths or nullable slices.
import { NotesService } from "../../../bindings/dotmd/internal/notes";
import type { Node, Tag as TagBinding } from "../../../bindings/dotmd/internal/notes";

export type { Node };
export type EntryKind = "note" | "folder";
/** What a Files tree row stands for. */
export type FileKind = EntryKind | "image";

export function kindOf(node: Node): FileKind {
  return node.isDir ? "folder" : node.isImage ? "image" : "note";
}
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

/**
 * Give a note, folder or image a new name in the same folder; a note or image
 * keeps its extension if `name` leaves it out. Resolves to the new path.
 */
export function renameEntry(path: string, name: string): Promise<string> {
  return NotesService.Rename(path, name);
}

/** Move a note, folder or image to the system Trash. */
export function trashEntry(path: string): Promise<void> {
  return NotesService.Delete(path);
}

/** True when `path` is `base` itself or lies inside the folder `base`. */
export function isWithin(path: string, base: string): boolean {
  return path === base || path.startsWith(base + "/");
}

/**
 * Where `path` is after `from` was renamed to `to` (`from` may be `path`
 * itself or a folder above it), or null if the rename didn't move it.
 */
export function movedPath(path: string, from: string, to: string): string | null {
  return isWithin(path, from) ? to + path.slice(from.length) : null;
}

/**
 * Root-relative path of the note a link in note `from` points to. Hrefs are
 * relative to `from`'s folder ("/" = notes root). Rejects if there is no such note.
 */
export function resolveLink(from: string, href: string): Promise<string> {
  return NotesService.ResolveLink(from, href);
}

/** Href that links note `from` to note `to`, relative to `from`'s folder. */
export function linkPath(from: string, to: string): Promise<string> {
  return NotesService.LinkPath(from, to);
}

export function notesRoot(): Promise<string> {
  return NotesService.Root();
}

/** Every note (not folder or image) in display order, depth-first. */
export function notePaths(nodes: Node[]): string[] {
  const out: string[] = [];
  const walk = (list: Node[]) => {
    for (const n of list) {
      if (n.isDir) walk(n.children ?? []);
      else if (!n.isImage) out.push(n.path);
    }
  };
  walk(nodes);
  return out;
}

/** Display name for a note: file name without its markdown extension. */
export function noteTitle(path: string): string {
  const name = path.split("/").pop() ?? path;
  return name.replace(/\.(md|markdown)$/i, "");
}
