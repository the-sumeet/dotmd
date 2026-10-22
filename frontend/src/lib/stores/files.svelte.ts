import { SvelteSet } from "svelte/reactivity";
import { FileService } from "../../../bindings/changeme";
import type { FileNode } from "../../../bindings/changeme/models.js";
import { editor } from "./editor.svelte.js";

/** Whether `path` is `root` or inside the folder `root`. */
export function within(path: string | null, root: string): path is string {
  return path !== null && (path === root || path.startsWith(root + "/"));
}

function parent(path: string): string {
  const i = path.lastIndexOf("/");
  return i === -1 ? "" : path.slice(0, i);
}

class FilesStore {
  tree = $state<FileNode[]>([]);
  /** Paths of the folders expanded in the explorer. */
  expanded = new SvelteSet<string>();

  async refresh() {
    try {
      this.tree = (await FileService.ListTree()) ?? [];
    } catch (err) {
      console.error("Failed to load data directory", err);
    }
  }

  find(path: string): FileNode | undefined {
    let nodes = this.tree;
    let node: FileNode | undefined;
    for (const name of path.split("/")) {
      node = nodes.find((n) => n.name === name);
      if (!node) return undefined;
      nodes = node.children ?? [];
    }
    return node;
  }

  /** Expands every folder above `path`. */
  reveal(path: string) {
    for (let dir = parent(path); dir !== ""; dir = parent(dir)) this.expanded.add(dir);
  }

  setExpanded(path: string, open: boolean) {
    if (open) this.expanded.add(path);
    else this.expanded.delete(path);
  }

  /** Renames the entry at `from` to `name`, keeping it in the same folder. */
  async rename(from: string, name: string) {
    const dir = parent(from);
    const to = dir === "" ? name : `${dir}/${name}`;
    if (to === from) return;
    const rename = async () => {
      await FileService.Rename(from, to);
      editor.moved(from, to);
    };
    // Hold the editor so a save can't recreate the file at its old path.
    await (within(editor.path, from) ? editor.hold(rename) : rename());
    for (const path of [...this.expanded]) {
      if (!within(path, from)) continue;
      this.expanded.delete(path);
      this.expanded.add(to + path.slice(from.length));
    }
    await this.refresh();
  }

  /** Copies the entry at `path` next to itself and returns the copy's path. */
  async duplicate(path: string): Promise<string> {
    if (within(editor.path, path)) await editor.flush();
    const copy = await FileService.Duplicate(path);
    await this.refresh();
    return copy;
  }

  async delete(path: string) {
    if (within(editor.path, path)) {
      // Save first: if the delete fails, the edits are on disk rather than discarded.
      await editor.hold(async () => {
        await FileService.Delete(path);
        await editor.close();
      });
    } else {
      await FileService.Delete(path);
    }
    for (const p of [...this.expanded]) if (within(p, path)) this.expanded.delete(p);
    await this.refresh();
  }
}

export const files = new FilesStore();
