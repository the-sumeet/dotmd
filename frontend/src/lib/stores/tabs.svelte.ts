import { editor } from "./editor.svelte.js";
import { files, within } from "./files.svelte.js";

const STORAGE_KEY = "dotmd:tabs";

type Saved = { paths: string[]; active: string | null };

/**
 * The notes open in tabs, in display order; the active one is `editor.path`.
 * Background tabs are just paths: the editor saves before it switches away.
 */
class TabsStore {
  paths = $state<string[]>([]);
  #restored = false;
  /** The tab last requested by a shortcut, so rapid presses step from it rather than from the still-open note. */
  #target: string | null = null;

  /** Opens `path` in a tab (next to the active one), or focuses its tab. */
  async open(path: string) {
    const from = editor.path;
    if (!(await editor.open(path))) return;
    // The user has opened something, so saving can no longer clobber an unrestored session.
    this.#restored = true;
    if (this.paths.includes(path)) return;
    const at = from === null ? -1 : this.paths.indexOf(from);
    this.paths.splice(at === -1 ? this.paths.length : at + 1, 0, path);
  }

  /** Closes the tab for `path`; closing the active tab opens its neighbour. */
  async close(path: string) {
    const i = this.paths.indexOf(path);
    if (i === -1) return;
    if (editor.path === path) {
      const next = this.paths[i + 1] ?? this.paths[i - 1];
      if (next === undefined) {
        if (!(await editor.save())) return;
        await editor.close();
      } else if (!(await editor.open(next))) {
        return;
      }
    }
    this.paths = this.paths.filter((p) => p !== path);
  }

  /** Moves `step` tabs from the active one, wrapping around. Returns whether there was a tab to move to. */
  step(step: number): boolean {
    const from = this.#target ?? editor.path;
    const i = from === null ? -1 : this.paths.indexOf(from);
    if (i === -1 || this.paths.length < 2) return false;
    const n = this.paths.length;
    this.#go(this.paths[(((i + step) % n) + n) % n]);
    return true;
  }

  /** Activates the tab at `index` (0-based); negative counts from the end. Returns whether it exists. */
  select(index: number): boolean {
    const path = this.paths.at(index);
    if (path === undefined) return false;
    this.#go(path);
    return true;
  }

  #go(path: string) {
    this.#target = path;
    void editor.open(path).finally(() => {
      if (this.#target === path) this.#target = null;
    });
  }

  /** Follow a rename of a note, or of a folder containing open notes. */
  moved(from: string, to: string) {
    this.paths = this.paths.map((p) =>
      within(p, from) ? to + p.slice(from.length) : p,
    );
  }

  /** Closes the tabs of notes deleted with `path`, focusing a survivor if the active one went. */
  async dropWithin(path: string) {
    const first = this.paths.findIndex((p) => within(p, path));
    if (first === -1) return;
    this.paths = this.paths.filter((p) => !within(p, path));
    if (editor.path !== null) return;
    const next = this.paths[Math.min(first, this.paths.length - 1)];
    if (next !== undefined) await editor.open(next);
  }

  /** Reopens the tabs of the last session, skipping notes that no longer exist. */
  async restore() {
    let saved: Saved | null = null;
    try {
      saved = JSON.parse(localStorage.getItem(STORAGE_KEY) ?? "null");
    } catch {
      // Unreadable storage just means a fresh start.
    }
    const savedPaths = Array.isArray(saved?.paths) ? saved.paths : [];
    try {
      if (savedPaths.length === 0) return;
      await files.refresh();
      const paths = savedPaths.filter((p) => {
        const node = typeof p === "string" ? files.find(p) : undefined;
        return node !== undefined && !node.isDir;
      });
      const active =
        saved?.active != null && paths.includes(saved.active)
          ? saved.active
          : paths.at(-1);
      // The user may have opened a note while the tree loaded; keep it.
      const userOpened = editor.path !== null;
      if (active !== undefined && (userOpened || (await editor.open(active)))) {
        this.paths = [...paths, ...this.paths.filter((p) => !paths.includes(p))];
      }
    } finally {
      // A session that failed to restore stays on disk until the user opens a note.
      if (savedPaths.length === 0 || this.paths.length > 0) this.#restored = true;
    }
  }

  save() {
    // Reads are tracked even when gated, so the caller's effect re-runs on change.
    const saved: Saved = { paths: [...this.paths], active: editor.path };
    if (!this.#restored) return;
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(saved));
    } catch {
      // Persistence is a convenience; ignore blocked storage.
    }
  }
}

export const tabs = new TabsStore();
