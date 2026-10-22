import { FileService } from "../../../bindings/changeme";

const DEBOUNCE_MS = 1000;
const MAX_INTERVAL_MS = 5000;

class EditorStore {
  path = $state<string | null>(null);
  content = $state("");
  dirty = $state(false);
  loading = $state(false);

  #debounce: ReturnType<typeof setTimeout> | undefined;
  #interval: ReturnType<typeof setInterval> | undefined;
  #openSeq = 0;
  #saving: Promise<void> = Promise.resolve();

  /** Flush the current file, then load `path` into the editor. */
  async open(path: string) {
    if (path === this.path) return;
    const seq = ++this.#openSeq;
    await this.flush();
    if (seq !== this.#openSeq) return;
    this.loading = true;
    try {
      const text = await FileService.ReadFile(path);
      if (seq !== this.#openSeq) return;
      this.path = path;
      this.content = text;
      this.dirty = false;
    } catch (err) {
      console.error("Failed to read file", path, err);
    } finally {
      if (seq === this.#openSeq) this.loading = false;
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
