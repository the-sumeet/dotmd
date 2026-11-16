<script lang="ts" module>
  import type { EditorState as State } from "@tiptap/pm/state";

  /** A note's editor state, kept while another note is shown (see snapshot). */
  export interface EditorSnapshot {
    state: State;
    frontmatter: string;
    /** The note's full markdown that `state` was made from. */
    markdown: string;
  }
</script>

<script lang="ts">
  import { onDestroy, onMount } from "svelte";
  import { Editor } from "@tiptap/core";
  import { EditorState } from "@tiptap/pm/state";
  import { createExtensions } from "$lib/editor/extensions";
  import { joinFrontmatter, splitFrontmatter } from "$lib/editor/frontmatter";
  import { refreshImageSrcs } from "$lib/editor/note-image";
  import { clearImagePlaceholders } from "$lib/editor/image-upload";
  import BubbleMenu from "./BubbleMenu.svelte";
  import LinkMenu from "./LinkMenu.svelte";
  import TableMenu from "./TableMenu.svelte";
  import "./editor.css";

  let {
    value = $bindable(""),
    pending = $bindable(false),
    editable = true,
    notePath = null,
    onopenlink,
    onuploadimage,
    ondownloadimage,
    onerror,
  }: {
    /** Full markdown of the note, including any YAML frontmatter. */
    value?: string;
    /** True while edits are in the editor but not yet in `value` (see flush). */
    pending?: boolean;
    /** When false the document is read-only (e.g. no note is open). */
    editable?: boolean;
    /** Follow a link (Cmd/Ctrl+click or the link menu), given its raw href. */
    onopenlink?: (href: string, newTab: boolean) => void;
    /** Root-relative path of the open note; image srcs are relative to it. */
    notePath?: string | null;
    /** Save a pasted, dropped or picked image beside the note; resolves to its href. */
    onuploadimage: (file: File) => Promise<string>;
    /** Download a pasted image URL beside the note; resolves to its href. */
    ondownloadimage: (url: string) => Promise<string>;
    /** Report a problem (e.g. an image that couldn't be saved). */
    onerror?: (message: string) => void;
  } = $props();

  /** Serializing the whole doc is O(n), so do it after typing pauses, not per keystroke. */
  const SYNC_DELAY_MS = 300;

  let element: HTMLDivElement;
  let fileInput: HTMLInputElement;
  let bubbleElement = $state<HTMLDivElement>();
  let tableMenuElement = $state<HTMLDivElement>();
  let linkMenuElement = $state<HTMLDivElement>();
  let editor = $state.raw<Editor | null>(null);
  let revision = $state(0);
  let frontmatter = "";
  let syncTimer: ReturnType<typeof setTimeout> | undefined;
  /** The note whose folder image srcs last pointed at. */
  let imagesNote: string | null = null;

  function refreshImages() {
    if (!editor || editor.isDestroyed) return;
    refreshImageSrcs(editor.view.dom, notePath);
    imagesNote = notePath;
  }

  onMount(() => {
    const { frontmatter: fm, body } = splitFrontmatter(value);
    frontmatter = fm;
    editor = new Editor({
      element,
      extensions: createExtensions({
        bubbleMenu: bubbleElement!,
        tableMenu: tableMenuElement!,
        linkMenu: linkMenuElement!,
        onOpenLink: (href, newTab) => onopenlink?.(href, newTab),
        // Read at render time, so images resolve against the note now open.
        notePath: () => notePath,
        uploadImage: (file) => onuploadimage(file),
        downloadImage: (url) => ondownloadimage(url),
        pickImage: () => fileInput.click(),
        onImageError: (message) => onerror?.(message),
      }),
      content: body,
      contentType: "markdown",
      editable,
      editorProps: {
        attributes: {
          class: "prose max-w-none min-h-[60vh] focus:outline-none",
          spellcheck: "true",
        },
      },
      onUpdate: () => {
        pending = true;
        clearTimeout(syncTimer);
        syncTimer = setTimeout(flush, SYNC_DELAY_MS);
      },
      onTransaction: () => {
        revision++;
      },
    });
  });

  $effect(() => {
    // emitUpdate=false: toggling editability is not an edit and must not
    // trigger a sync (which would re-serialize, and so rewrite, the note).
    if (editor && editor.isEditable !== editable) editor.setEditable(editable, false);
  });

  // The open note was renamed or moved with its folder: point its images at
  // the new folder (switching notes is handled by setMarkdown).
  $effect(() => {
    if (notePath !== imagesNote) refreshImages();
  });

  onDestroy(() => {
    flush();
    editor?.destroy();
  });

  /**
   * Write pending edits to `value` immediately (before showing source, saving, …).
   * Does nothing when there are none: serializing re-formats the markdown
   * (bullet style, blank lines), so an untouched note must keep its original text.
   */
  export function flush() {
    if (syncTimer === undefined) return;
    clearTimeout(syncTimer);
    syncTimer = undefined;
    pending = false;
    if (editor && !editor.isDestroyed) value = joinFrontmatter(frontmatter, editor.getMarkdown());
  }

  /**
   * Replace the document with new markdown (e.g. after editing the raw source).
   * This acts like reloading the file: undo history is reset, because the
   * frontmatter lives outside the Tiptap document and undo could not restore it.
   */
  export function setMarkdown(markdown: string) {
    if (!editor) return;
    clearTimeout(syncTimer);
    syncTimer = undefined;
    pending = false;
    const { frontmatter: fm, body } = splitFrontmatter(markdown);
    frontmatter = fm;
    editor.commands.setContent(body, { contentType: "markdown", emitUpdate: false });
    const { doc, plugins, selection } = editor.state;
    editor.view.updateState(EditorState.create({ doc, plugins, selection }));
    refreshImages();
  }

  /**
   * The document with its undo history and selection, to put back with
   * restore() when the note is shown again. Pending edits are flushed first,
   * so `markdown` is the text the state stands for.
   */
  export function snapshot(): EditorSnapshot | null {
    if (!editor || editor.isDestroyed) return null;
    flush();
    return { state: editor.state, frontmatter, markdown: value };
  }

  /** Show a note from its snapshot; like setMarkdown, but keeps undo history. */
  export function restore(snap: EditorSnapshot) {
    if (!editor) return;
    clearTimeout(syncTimer);
    syncTimer = undefined;
    pending = false;
    frontmatter = snap.frontmatter;
    editor.view.updateState(snap.state);
    // Placeholders whose uploads settled while the note was hidden would
    // spin forever. Dispatching also bumps `revision`, so the menus re-read
    // the restored selection instead of the previous note's.
    editor.view.dispatch(clearImagePlaceholders(editor.state.tr));
    refreshImages();
  }

  export function focus() {
    editor?.commands.focus();
  }

  /** Insert an existing image at the caret, given its href from the note. */
  export function insertImage(src: string, alt: string) {
    if (!editor?.isEditable) return;
    editor.chain().focus().insertContent({ type: "image", attrs: { src, alt } }).run();
  }

  function onPicked() {
    const files = Array.from(fileInput.files ?? []);
    fileInput.value = ""; // picking the same file again still fires "change"
    if (editor && files.length) editor.storage.imageUpload.uploadImages(files);
  }
</script>

<div bind:this={element} class="dotmd-editor"></div>
<input
  bind:this={fileInput}
  type="file"
  accept="image/*"
  multiple
  hidden
  onchange={onPicked}
/>
<BubbleMenu {editor} {revision} bind:element={bubbleElement} />
<TableMenu {editor} {revision} bind:element={tableMenuElement} />
<LinkMenu {editor} {revision} bind:element={linkMenuElement} onopen={onopenlink} />
