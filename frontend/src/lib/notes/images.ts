// Image attachments: thin wrapper over the Go bindings, plus the mapping from
// an image's href in a note to the URL the webview loads it from.
import { NotesService } from "../../../bindings/dotmd/internal/notes";
import { isExternalLink } from "./links";

/** URL prefix the Go side serves notes-folder images under (notes.FileRoute). */
const FILE_ROUTE = "/notes-file/";

const IMAGE_EXT = /\.(png|jpe?g|gif|webp|avif|bmp|svg)$/i;

/** Save `file` to the assets folder beside `note`; resolves to the href from the note. */
export async function saveImageFile(note: string, file: File): Promise<string> {
  return NotesService.SaveImage(note, file.name, await toBase64(file));
}

/** Download the image at `url` beside `note`; resolves to the href from the note. */
export function downloadImage(note: string, url: string): Promise<string> {
  return NotesService.DownloadImage(note, url);
}

/**
 * The http(s) src of the image when pasted HTML is nothing but one image
 * (a browser's "Copy Image"), so it can be downloaded; otherwise null.
 */
export function loneImageSrc(html: string): string | null {
  const doc = new DOMParser().parseFromString(html, "text/html");
  const imgs = doc.body.querySelectorAll("img");
  if (imgs.length !== 1 || doc.body.textContent?.trim()) return null;
  const src = imgs[0].getAttribute("src") ?? "";
  return /^https?:\/\//i.test(src) ? src : null;
}

/** True when pasted HTML holds nothing but images (no text, no table). */
export function htmlIsOnlyImages(html: string): boolean {
  const doc = new DOMParser().parseFromString(html, "text/html");
  return !doc.body.textContent?.trim() && doc.body.querySelector("img") !== null;
}

export function isImageFile(file: File): boolean {
  return file.type.startsWith("image/") || IMAGE_EXT.test(file.name);
}

/** True for a lone http(s) URL whose path names an image file. */
export function isImageUrl(text: string): boolean {
  const t = text.trim();
  if (!/^https?:\/\/\S+$/i.test(t)) return false;
  try {
    return IMAGE_EXT.test(new URL(t).pathname);
  } catch {
    return false;
  }
}

/** File name without its extension, e.g. for an image's alt text. */
export function baseName(name: string): string {
  return (name.split("/").pop() ?? name).replace(/\.[^.]+$/, "");
}

/**
 * The URL the webview loads image `src` (as written in note `note`) from.
 * Like links, relative srcs are relative to the note's folder and "/" is the
 * notes root. Web, data and blob URLs are used as they are.
 */
export function imageSrc(note: string | null, src: string): string {
  const path = rootPath(note, src);
  return path === null ? src : fileSrc(path);
}

/**
 * The URL the webview loads the root-relative file `path` from. Unlike
 * imageSrc, `path` is a file path, not an href: "#", "?" and "%" are part of
 * the name.
 */
export function fileSrc(path: string): string {
  return FILE_ROUTE + path.split("/").map(encodeURIComponent).join("/");
}

/**
 * Root-relative path of the file that `src` (as written in note `note`)
 * points to, or null for web/data URLs and paths outside the notes folder.
 */
export function rootPath(note: string | null, src: string): string | null {
  if (!src || isExternalLink(src) || src.startsWith("//")) return null;
  let target = src.replace(/[?#].*$/, "");
  try {
    target = decodeURIComponent(target);
  } catch {
    // Keep a malformed escape as written, like Go's ResolveLink.
  }
  const parts = target.startsWith("/") ? [] : (note ?? "").split("/").slice(0, -1);
  for (const seg of target.split("/")) {
    if (seg === "" || seg === ".") continue;
    if (seg === "..") {
      if (!parts.length) return null;
      parts.pop();
    } else {
      parts.push(seg);
    }
  }
  return parts.length ? parts.join("/") : null;
}

/**
 * Href that links note `from` to the root-relative file `to`, like Go's
 * LinkPath: relative to `from`'s folder, with "%", "#" and "?" escaped, and
 * "./" in front when the name would otherwise read as a scheme ("Re: x.md").
 */
export function relativeHref(from: string, to: string): string {
  const fromDir = from.split("/").slice(0, -1);
  const target = to.split("/");
  let common = 0;
  while (common < fromDir.length && common < target.length - 1 && fromDir[common] === target[common]) {
    common++;
  }
  let rel = [...fromDir.slice(common).map(() => ".."), ...target.slice(common)].join("/");
  if (isExternalLink(rel)) rel = "./" + rel;
  return rel.replace(/[%#?]/g, (ch) => encodeURIComponent(ch));
}

/**
 * `src` as written in note `from`, rewritten to work from note `to` (an image
 * copied into a note in another folder). Web URLs and root paths ("/a.png")
 * work anywhere and are kept, as is everything within the same note.
 */
export function rebaseSrc(from: string, to: string | null, src: string): string {
  if (!to || from === to || src.startsWith("/")) return src;
  const path = rootPath(from, src);
  return path === null ? src : relativeHref(to, path);
}

function toBase64(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve(String(reader.result).replace(/^[^,]*,/, ""));
    reader.onerror = () => reject(reader.error ?? new Error("couldn't read the file"));
    reader.readAsDataURL(file);
  });
}
