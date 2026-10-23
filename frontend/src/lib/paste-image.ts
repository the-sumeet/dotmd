import { EditorView } from "@codemirror/view";
import { toast } from "svelte-sonner";
import { FileService } from "../../bindings/changeme";
import { relativeNotePath } from "./note-links.js";
import { editor } from "./stores/editor.svelte.js";
import { files } from "./stores/files.svelte.js";

const EXTENSIONS: Record<string, string> = {
  "image/png": ".png",
  "image/jpeg": ".jpg",
  "image/gif": ".gif",
  "image/webp": ".webp",
  "image/svg+xml": ".svg",
};

/**
 * Pasting an image saves it next to the open note and inserts a relative link to it.
 * Reads the note at paste time, so one instance serves every document.
 */
export const pasteImage = EditorView.domEventHandlers({
  paste(event, view) {
    const note = editor.path;
    const image = [...(event.clipboardData?.files ?? [])].find((f) => f.type in EXTENSIONS);
    // Copying from a spreadsheet puts text and a picture of it on the clipboard; the text wins.
    if (note === null || !image || event.clipboardData?.getData("text/plain")) return false;
    event.preventDefault();
    void save(view, note, editor.docId, image);
    return true;
  },
});

async function save(view: EditorView, note: string, docId: number, image: File) {
  try {
    const dir = note.slice(0, Math.max(note.lastIndexOf("/"), 0));
    const saved = await FileService.SaveImage(dir, EXTENSIONS[image.type], await toBase64(image));
    // The user may have opened another note while the image was saved. A rename doesn't matter:
    // the link is relative to the folder, which a rename of the file leaves alone.
    if (editor.docId !== docId || !view.dom.isConnected) {
      toast.info("Image saved, but the note was closed", { description: saved });
      void files.refresh();
      return;
    }
    const link = `![](${relativeNotePath(note, saved)})`;
    const { from, to } = view.state.selection.main;
    view.dispatch({
      changes: { from, to, insert: link },
      selection: { anchor: from + link.length },
      userEvent: "input.paste",
    });
    void files.refresh();
  } catch (err) {
    console.error("Failed to save pasted image", err);
    toast.error("Couldn't save the pasted image");
  }
}

function toBase64(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    // A data URL is "data:<type>;base64,<payload>".
    reader.onload = () => resolve((reader.result as string).split(",", 2)[1]);
    reader.onerror = () => reject(reader.error);
    reader.readAsDataURL(file);
  });
}
