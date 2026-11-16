import { Extension, type Editor } from "@tiptap/core";
import { Plugin, PluginKey, type Transaction } from "@tiptap/pm/state";
import { Decoration, DecorationSet, type EditorView } from "@tiptap/pm/view";
import {
  baseName,
  htmlIsOnlyImages,
  isImageFile,
  isImageUrl,
  loneImageSrc,
} from "$lib/notes/images";
import { errorText } from "$lib/utils";

export interface ImageUploadOptions {
  /**
   * Save a pasted, dropped or picked file; resolves to its href from the note.
   * The image is only inserted if the editor is still editable when the save
   * finishes: a hidden (Markdown mode) or switching editor would lose it.
   */
  upload: (file: File) => Promise<string>;
  /** Download a pasted image URL; resolves to the local copy's href. */
  download: (url: string) => Promise<string>;
  /** Open the system file picker (its files come back via `uploadImages`). */
  pick: () => void;
  onError: (message: string) => void;
}

declare module "@tiptap/core" {
  interface Commands<ReturnType> {
    imageUpload: {
      /** Let the user choose images from their computer. */
      pickImage: () => ReturnType;
    };
  }
  interface Storage {
    imageUpload: ImageUploadStorage;
  }
}

export interface ImageUploadStorage {
  /**
   * Set when the editor is created. Tiptap copies the storage object, so the
   * methods below read it through `this` (always call them as methods).
   */
  editor: Editor | null;
  /**
   * Save image files and insert them at `pos` (default: the selection).
   * Not a command: it dispatches its own transactions as saves finish.
   */
  uploadImages(this: ImageUploadStorage, files: File[], pos?: number): boolean;
  /**
   * Show a placeholder at `pos` until `href` settles, then put the image
   * there. `fallback` runs at the placeholder instead if saving fails.
   */
  track(
    this: ImageUploadStorage,
    pos: number,
    href: Promise<string>,
    alt: string,
    fallback?: (pos: number) => void,
  ): void;
}

/** Shown where an image goes while it is being saved. */
type Action =
  | { add: { id: object; pos: number } }
  | { remove: { id: object } }
  | { clear: true };

const key = new PluginKey<DecorationSet>("imageUpload");

/**
 * Drop every "Adding image…" placeholder, e.g. from a note's kept editor
 * state: its uploads finished (or failed) while another note was shown.
 */
export function clearImagePlaceholders(tr: Transaction): Transaction {
  return tr.setMeta(key, { clear: true } satisfies Action);
}

function placeholderAt(view: EditorView, id: object): number | null {
  const found = key.getState(view.state)?.find(undefined, undefined, (spec) => spec.id === id);
  return found?.length ? found[0].from : null;
}

/**
 * What a paste should do with images: upload the clipboard's image files,
 * download an image URL, or nothing (let the editor paste as usual).
 */
function pastedImages(data: DataTransfer): { files: File[] } | { url: string } | null {
  const files = Array.from(data.files).filter(isImageFile);
  const html = data.getData("text/html");
  const text = data.getData("text/plain").trim();
  // Office apps and some editors add a picture of the selection next to the
  // text or table; the text wins. A browser's "Copy Image" adds the image's
  // URL (as text or as a lone <img>), and copying files in Finder may add their
  // names; neither is a reason to skip the files.
  const namesOnly = text.split(/\r?\n/).every((line) => files.some((f) => f.name === line.trim()));
  const onlyImage = html ? htmlIsOnlyImages(html) : !text || isImageUrl(text) || namesOnly;
  if (files.length && onlyImage) return { files };
  if (files.length) return null;
  if (isImageUrl(text) && (!html || onlyImage)) return { url: text };
  const src = html ? loneImageSrc(html) : null;
  return src ? { url: src } : null;
}

/**
 * Pasted image data, a pasted image URL, dropped image files and picked files
 * are saved beside the note and inserted as `![name](assets/…)`. A placeholder
 * marks the spot while saving; it moves with edits made in the meantime. If
 * its text is deleted, the note is replaced, or the editor is hidden by then,
 * the image is still saved but not inserted, and onError says where it went.
 */
export const ImageUpload = Extension.create<ImageUploadOptions, ImageUploadStorage>({
  name: "imageUpload",

  addOptions() {
    return {
      upload: () => Promise.reject(new Error("open a note first")),
      download: () => Promise.reject(new Error("open a note first")),
      pick: () => {},
      onError: () => {},
    };
  },

  addCommands() {
    return {
      pickImage: () => () => {
        this.options.pick();
        return true;
      },
    };
  },

  addStorage() {
    const ext = this;
    return {
      editor: null,
      uploadImages(files, pos) {
        const editor = this.editor;
        const images = files.filter(isImageFile);
        if (!editor || !images.length) return false;
        const at = pos ?? editor.state.selection.from;
        // Saves run at once, but each image is inserted only after the one
        // before it settles, so they land in the order the user chose rather
        // than the order the saves finish.
        let previous: Promise<unknown> = Promise.resolve();
        for (const file of images) {
          const saved = ext.options.upload(file);
          const inOrder = previous.then(() => saved);
          // track's handlers are attached first, so they run (and insert)
          // before the next image's turn comes.
          this.track(at, inOrder, baseName(file.name) || "image");
          previous = inOrder.catch(() => {});
        }
        return true;
      },
      track(pos, href, alt, fallback) {
        const editor = this.editor;
        if (!editor) return;
        const id = {};
        editor.view.dispatch(editor.state.tr.setMeta(key, { add: { id, pos } } satisfies Action));
        /** Remove the placeholder; true if `insert` ran at its position. */
        const done = (insert: (pos: number) => void): boolean => {
          if (editor.isDestroyed) return false;
          const at = placeholderAt(editor.view, id);
          if (at === null) return false; // the note was replaced, or the spot deleted
          editor.view.dispatch(editor.state.tr.setMeta(key, { remove: { id } } satisfies Action));
          // Not editable: another note is opening (this one is already saved),
          // or Markdown mode hides the editor, which would overwrite the source
          // being edited when it syncs.
          if (!editor.isEditable) return false;
          insert(at);
          return true;
        };
        href.then(
          (src) => {
            const added = done((at) => {
              editor.chain().insertContentAt(at, { type: "image", attrs: { src, alt } }).run();
            });
            if (!added) ext.options.onError(`Saved the image as ${src}, but didn't add it to the note`);
          },
          (err) => {
            ext.options.onError(`Couldn't add image: ${errorText(err)}`);
            done(fallback ?? (() => {}));
          },
        );
      },
    };
  },

  onBeforeCreate() {
    this.storage.editor = this.editor;
  },

  addProseMirrorPlugins() {
    const ext = this;
    return [
      new Plugin<DecorationSet>({
        key,
        state: {
          init: () => DecorationSet.empty,
          apply(tr, set) {
            set = set.map(tr.mapping, tr.doc);
            const action = tr.getMeta(key) as Action | undefined;
            if (action && "add" in action) {
              const el = document.createElement("span");
              el.className = "dotmd-image-placeholder";
              el.setAttribute("aria-label", "Adding image…");
              set = set.add(tr.doc, [
                Decoration.widget(action.add.pos, el, { id: action.add.id }),
              ]);
            } else if (action && "clear" in action) {
              set = DecorationSet.empty;
            } else if (action && "remove" in action) {
              set = set.remove(
                set.find(undefined, undefined, (spec) => spec.id === action.remove.id),
              );
            }
            return set;
          },
        },
        props: {
          decorations: (state) => key.getState(state),

          handlePaste(view, event) {
            const data = event.clipboardData;
            // In code, a pasted URL is text; an image could not go there anyway.
            if (!data || !view.editable || ext.editor.isActive("codeBlock") || ext.editor.isActive("code")) {
              return false;
            }
            const images = pastedImages(data);
            if (!images) return false;
            event.preventDefault();
            if (!view.state.selection.empty) view.dispatch(view.state.tr.deleteSelection());
            const pos = view.state.selection.from;
            if ("files" in images) return ext.storage.uploadImages(images.files, pos);
            // An image URL: keep a local copy, or paste it as a link if that fails.
            const { url } = images;
            ext.storage.track(pos, ext.options.download(url), baseName(url), (at) => {
              ext.editor
                .chain()
                .insertContentAt(at, { type: "text", text: url, marks: [{ type: "link", attrs: { href: url } }] })
                .run();
            });
            return true;
          },

          handleDrop(view, event, _slice, moved) {
            if (moved || !view.editable) return false;
            const files = Array.from(event.dataTransfer?.files ?? []).filter(isImageFile);
            if (!files.length) return false;
            event.preventDefault();
            const pos =
              view.posAtCoords({ left: event.clientX, top: event.clientY })?.pos ??
              view.state.selection.from;
            return ext.storage.uploadImages(files, pos);
          },
        },
      }),
    ];
  },
});
