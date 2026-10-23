import { SvelteSet } from "svelte/reactivity";
import { toast } from "svelte-sonner";
import { FileService } from "../../../bindings/changeme";
import type { FileNode } from "../../../bindings/changeme/models.js";
import { editor } from "./editor.svelte.js";

/** Whether `path` is `root` or inside the folder `root`. */
export function within(path: string | null, root: string): path is string {
  return path !== null && (path === root || path.startsWith(root + "/"));
}

/** The folder containing `path` ("" for the root). */
export function parent(path: string): string {
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
    for (let dir = parent(path); dir !== ""; dir = parent(dir))
      this.expanded.add(dir);
  }

  setExpanded(path: string, open: boolean) {
    if (open) this.expanded.add(path);
    else this.expanded.delete(path);
  }

  /** Renames the entry at `from` to `name`, keeping it in the same folder. */
  async rename(from: string, name: string) {
    const dir = parent(from);
    await this.moveTo(from, dir === "" ? name : `${dir}/${name}`);
  }

  /** Moves the entry at `from` into the folder `destDir` ("" for the root), keeping its name. */
  async move(from: string, destDir: string) {
    if (within(destDir, from)) return;
    const name = from.slice(from.lastIndexOf("/") + 1);
    await this.moveTo(from, destDir === "" ? name : `${destDir}/${name}`);
    if (destDir !== "") this.expanded.add(destDir);
  }

  private async moveTo(from: string, to: string) {
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

  /** Creates an empty note in the folder `dir` ("" for the root), opens it and returns its path. */
  async createFile(dir: string, name: string): Promise<string> {
    const path = await FileService.CreateFile(dir, name);
    if (dir !== "") this.expanded.add(dir);
    await this.refresh();
    // The file exists now, so a failure to open it must not read as a failure to create it.
    try {
      await editor.open(path);
    } catch (err) {
      console.error("Failed to open new file", path, err);
      toast.error("Created the file but couldn't open it", {
        description: path,
      });
    }
    return path;
  }

  /** Creates an empty folder in the folder `dir` ("" for the root) and returns its path. */
  async createFolder(dir: string, name: string): Promise<string> {
    const path = await FileService.CreateFolder(dir, name);
    if (dir !== "") this.expanded.add(dir);
    await this.refresh();
    return path;
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
    for (const p of [...this.expanded])
      if (within(p, path)) this.expanded.delete(p);
    await this.refresh();
  }
}

export const files = new FilesStore();
