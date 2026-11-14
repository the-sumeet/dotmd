import { mergeAttributes } from "@tiptap/core";
import Image, { type ImageOptions } from "@tiptap/extension-image";
import { imageSrc, rebaseSrc } from "$lib/notes/images";
import { markdownHref } from "./markdown-href";

export interface NoteImageOptions extends ImageOptions {
  /** Root-relative path of the open note, which image srcs are relative to. */
  notePath: () => string | null;
}

/**
 * Image whose src stays exactly as written in the note (e.g. "assets/a.png"),
 * while the webview loads it from the notes folder (see imageSrc). The
 * original src and the note it belongs to are kept in `data-src` and
 * `data-note`, so an image copied into a note in another folder is re-linked
 * from there.
 */
export const NoteImage = Image.extend<NoteImageOptions>({
  addOptions() {
    return { ...this.parent!(), notePath: () => null };
  },

  addAttributes() {
    return {
      ...this.parent!(),
      src: {
        default: null,
        parseHTML: (el) => {
          const src = el.getAttribute("data-src");
          const from = el.getAttribute("data-note");
          if (src === null) return el.getAttribute("src");
          return from ? rebaseSrc(from, this.options.notePath(), src) : src;
        },
        renderHTML: (attrs) => {
          const src = attrs.src as string | null;
          if (!src) return {};
          const note = this.options.notePath();
          return { src: imageSrc(note, src), "data-src": src, "data-note": note ?? undefined };
        },
      },
    };
  },

  renderHTML({ HTMLAttributes }) {
    return ["img", mergeAttributes(this.options.HTMLAttributes, HTMLAttributes)];
  },

  // Like the stock renderer, but brackets in the alt text and spaces in the
  // src are escaped, so the image still parses as one.
  renderMarkdown: (node) => {
    const src = markdownHref(node.attrs?.src ?? "");
    const alt = (node.attrs?.alt ?? "").replace(/\s+/g, " ").replace(/[\\[\]]/g, "\\$&");
    const title = (node.attrs?.title ?? "").replace(/[\\"]/g, "\\$&");
    return title ? `![${alt}](${src} "${title}")` : `![${alt}](${src})`;
  },
});

/**
 * Point every image in `dom` at the open note's folder. ProseMirror reuses the
 * DOM of unchanged nodes when the document is replaced, so after switching
 * notes an image with the same relative src would still show the old file.
 */
export function refreshImageSrcs(dom: HTMLElement, note: string | null) {
  for (const img of dom.querySelectorAll<HTMLImageElement>("img[data-src]")) {
    const src = imageSrc(note, img.dataset.src!);
    if (img.getAttribute("src") !== src) img.setAttribute("src", src);
    img.dataset.note = note ?? "";
  }
}
