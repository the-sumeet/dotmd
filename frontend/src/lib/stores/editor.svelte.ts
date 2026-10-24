import { toast } from "svelte-sonner";
import { FileService } from "../../../bindings/changeme";

const DEBOUNCE_MS = 1000;
const MAX_INTERVAL_MS = 5000;

class EditorStore {
  path = $state<string | null>(null);
  content = $state("");
  dirty = $state(false);
  loading = $state(false);
  /** Changes whenever a different document is loaded; unlike `path`, not on rename. */
  docId = $state(0);

  #debounce: ReturnType<typeof setTimeout> | undefined;
  #interval: ReturnType<typeof setInterval> | undefined;
  #openSeq = 0;
  /** Whether an `open` is in flight, i.e. one that re-selecting the current note should cancel. */
  #opening = false;
  #saving: Promise<void> = Promise.resolve();

  /** Flush the current file, then load `path` into the editor. Resolves to whether `path` is now open. */
  async open(path: string): Promise<boolean> {
    if (path === this.path) {
      // Re-selecting the open note cancels a switch still in flight.
      if (this.#opening) {
        ++this.#openSeq;
        this.#opening = false;
        this.loading = false;
      }
      return true;
    }
    const seq = ++this.#openSeq;
    this.#opening = true;
    try {
      // Never replace text that couldn't be written to disk.
      if (!(await this.save()) || seq !== this.#openSeq) return false;
      this.loading = true;
      const text = await FileService.ReadFile(path);
      if (seq !== this.#openSeq) return false;
      this.path = path;
      this.content = text;
      this.dirty = false;
      this.docId++;
      return true;
    } catch (err) {
      console.error("Failed to read file", path, err);
      toast.error("Couldn't open note", { description: path });
      return false;
    } finally {
      if (seq === this.#openSeq) {
        this.loading = false;
        this.#opening = false;
      }
    }
  }

  /** Close the current file, discarding unsaved changes. Resolves once any in-flight write has finished. */
  close(): Promise<void> {
    ++this.#openSeq;
    this.#opening = false;
    clearTimeout(this.#debounce);
    clearInterval(this.#interval);
    this.#interval = undefined;
    this.path = null;
    this.content = "";
    this.dirty = false;
    this.loading = false;
    this.docId++;
    return this.#saving;
  }

  /**
   * Save, then run `fn` with the editor read-only, so no edit or save can race it.
   * For file operations that move or remove the current file.
   */
  async hold<T>(fn: () => Promise<T>): Promise<T> {
    ++this.#openSeq; // cancel an in-flight open, which could land on a stale path
    this.#opening = false;
    this.loading = true;
    try {
      await this.flush();
      return await fn();
    } finally {
      this.loading = false;
    }
  }

  /** Follow a rename of the current file, or of a folder containing it. */
  moved(from: string, to: string) {
    if (this.path === from || this.path?.startsWith(from + "/")) {
      this.path = to + this.path.slice(from.length);
    }
  }

  /** Called on user edits. */
  edit = (value: string) => {
    if (this.path === null || this.loading) return;
    this.content = value;
    this.dirty = true;
    clearTimeout(this.#debounce);
    this.#debounce = setTimeout(() => void this.flush(), DEBOUNCE_MS);
    // Guarantees a save during uninterrupted typing.
    this.#interval ??= setInterval(() => void this.flush(), MAX_INTERVAL_MS);
  };

  /** Like `flush`, but resolves to false (with a toast) if the changes still couldn't be written. */
  async save(): Promise<boolean> {
    await this.flush();
    if (!this.dirty) return true;
    toast.error("Couldn't save note", { description: this.path ?? undefined });
    return false;
  }

  /** Write pending changes to disk. Resolves once the write has finished. */
  flush(): Promise<void> {
    clearTimeout(this.#debounce);
    clearInterval(this.#interval);
    this.#interval = undefined;
    this.#saving = this.#saving.then(() => this.#write());
    return this.#saving;
  }

  async #write() {
    const path = this.path;
    if (path === null || !this.dirty) return;
    const content = this.content;
    try {
      await FileService.WriteFile(path, content);
      // Edits made during the write keep the file dirty.
      if (this.path === path && this.content === content) this.dirty = false;
    } catch (err) {
      console.error("Failed to save file", path, err);
    }
  }
}

export const editor = new EditorStore();
