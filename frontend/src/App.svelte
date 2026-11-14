<script lang="ts">
  import { onMount } from "svelte";
  import { Browser, Events } from "@wailsio/runtime";
  import { FileText, PenLine } from "@lucide/svelte";
  import { Button } from "$lib/components/ui/button";
  import Editor from "$lib/components/editor/Editor.svelte";
  import { splitFrontmatter } from "$lib/editor/frontmatter";
  import sample from "$lib/editor/sample.md?raw";
  import * as Breadcrumb from "$lib/components/ui/breadcrumb/index.js";
  import * as Sidebar from "$lib/components/ui/sidebar/index.js";
  import { Separator } from "$lib/components/ui/separator/index.js";
  import AppSidebar from "$lib/components/app-sidebar.svelte";
  import { Service as Lifecycle } from "../bindings/dotmd/internal/lifecycle";
  import {
    createFolder,
    createNote,
    firstNote,
    loadTags,
    loadTree,
    noteTitle,
    readNote,
    resolveLink,
    tagsOf,
    withNoteTags,
    writeNote,
    type EntryKind,
    type Node,
    type Tag,
  } from "$lib/notes/notes";
  import { canOpenExternally, isExternalLink, normalizeHref } from "$lib/notes/links";
  import {
    baseName,
    deleteImage,
    downloadImage,
    loadImages,
    relativeHref,
    saveImageFile,
    type Image,
  } from "$lib/notes/images";

  type Mode = "edit" | "markdown";

  /** Saving several images rescans the folder once, this long after the last. */
  const IMAGES_REFRESH_DELAY_MS = 300;
  /** Wait this long after the last change before writing the note to disk. */
  const SAVE_DELAY_MS = 500;
  /** Sent by the Go side before quitting; see internal/lifecycle. */
  const FLUSH_EVENT = "dotmd:flush";

  /** The open note as plain markdown — the single source of truth. */
  let markdown = $state("");
  let mode = $state<Mode>("edit");
  let sourceEdited = false;
  let editor: Editor;
  let source = $state<HTMLTextAreaElement>();

  let tree = $state<Node[]>([]);
  let tags = $state<Tag[]>([]);
  let images = $state<Image[]>([]);
  /** Root-relative path of the open note. */
  let currentPath = $state<string | null>(null);
  /** True while switching notes; editing is paused so no keystroke is lost. */
  let opening = $state(false);
  /** Last save/open problem, shown in the note header. */
  let notice = $state("");
  /** Content last written to (or read from) disk. */
  let lastSaved: string | null = null;
  let saveTimer: ReturnType<typeof setTimeout> | undefined;
  let imagesTimer: ReturnType<typeof setTimeout> | undefined;
  /** Saves run one after another, so an older write never lands last. */
  let saveQueue: Promise<boolean> = Promise.resolve(true);
  /** Bumped per openNote call so only the latest click wins. */
  let openToken = 0;

  const canEdit = $derived(currentPath !== null && !opening);
  const fileName = $derived(currentPath?.split("/").pop() ?? "No note open");
  const folders = $derived(currentPath?.split("/").slice(0, -1) ?? []);

  function errorText(err: unknown) {
    return err instanceof Error ? err.message : String(err);
  }

  function setMode(next: Mode) {
    if (next === mode) return;
    if (next === "markdown") {
      editor.flush();
    } else {
      if (sourceEdited) editor.setMarkdown(markdown);
      sourceEdited = false;
      queueMicrotask(() => editor.focus());
    }
    mode = next;
  }

  /**
   * Write the open note to disk if it changed since it was last saved.
   * Resolves to false if the write failed (the note stays unsaved).
   */
  function save(): Promise<boolean> {
    clearTimeout(saveTimer);
    saveTimer = undefined;
    saveQueue = saveQueue.then(writeIfChanged).catch((err) => {
      notice = `Not saved: ${errorText(err)}`;
      return false;
    });
    return saveQueue;
  }

  async function writeIfChanged(): Promise<boolean> {
    // In markdown mode the textarea is the source of truth; flushing the
    // (hidden, stale) editor would overwrite its edits.
    if (mode === "edit") editor.flush();
    const path = currentPath;
    const content = markdown;
    if (!path || content === lastSaved) return true;
    const previous = lastSaved;
    try {
      await writeNote(path, content);
    } catch (err) {
      notice = `Not saved: ${errorText(err)}`;
      return false;
    }
    lastSaved = content;
    notice = "";
    if (previous === null || frontmatterOf(previous) !== frontmatterOf(content)) {
      void updateTags(path, content);
    }
    return true;
  }

  function frontmatterOf(text: string) {
    return splitFrontmatter(text).frontmatter;
  }

  /** Update the tag list for one saved note instead of rescanning them all. */
  async function updateTags(path: string, content: string) {
    try {
      tags = withNoteTags(tags, path, await tagsOf(content));
    } catch (err) {
      console.error("Failed to read tags", err);
    }
  }

  async function refreshTags() {
    try {
      tags = await loadTags();
    } catch (err) {
      console.error("Failed to load tags", err);
    }
  }

  async function refreshImages() {
    try {
      images = await loadImages();
    } catch (err) {
      console.error("Failed to load images", err);
    }
  }

  /** Refresh the sidebar's images once a burst of saves (a drop of many files) ends. */
  function scheduleImagesRefresh() {
    clearTimeout(imagesTimer);
    imagesTimer = setTimeout(refreshImages, IMAGES_REFRESH_DELAY_MS);
  }

  /** The open note, for saving an image beside it. */
  function noteForImages(): string {
    if (!currentPath || opening) throw new Error("open a note first");
    return currentPath;
  }

  /** Save a pasted, dropped or picked image beside the open note. */
  async function uploadImage(file: File): Promise<string> {
    const href = await saveImageFile(noteForImages(), file);
    scheduleImagesRefresh();
    return href;
  }

  /** Keep a local copy of a pasted image URL beside the open note. */
  async function fetchImage(url: string): Promise<string> {
    const href = await downloadImage(noteForImages(), url);
    scheduleImagesRefresh();
    return href;
  }

  /** Show an image from the sidebar in the open note. */
  function insertImage(path: string) {
    if (!canEdit || mode !== "edit") {
      notice = canEdit ? "Switch to Edit to add images" : "Open a note to add images";
      return;
    }
    editor.insertImage(relativeHref(currentPath!, path), baseName(path));
  }

  function pickImage() {
    if (!canEdit) return;
    if (mode !== "edit") setMode("edit");
    editor.pickImage();
  }

  async function removeImage(path: string) {
    await deleteImage(path);
    await refreshImages();
  }

  async function openNote(path: string) {
    // Bump first, so clicking back to the open note cancels a pending switch.
    const token = ++openToken;
    if (path === currentPath) {
      opening = false;
      return;
    }
    opening = true;
    try {
      // Keep the current note open if its edits could not be saved.
      if (!(await save()) || token !== openToken) return;
      let content: string;
      try {
        content = await readNote(path);
      } catch (err) {
        notice = `Couldn't open ${noteTitle(path)}: ${errorText(err)}`;
        return;
      }
      if (token !== openToken) return;
      lastSaved = content;
      markdown = content;
      currentPath = path;
      editor.setMarkdown(content);
      sourceEdited = false;
      notice = "";
    } finally {
      if (token === openToken) opening = false;
    }
  }

  /**
   * Follow a link from the open note: URLs open in the system browser, and
   * relative paths open the note they point to.
   */
  async function followLink(raw: string) {
    const href = normalizeHref(raw); // "example.com" is a web address, not a note
    try {
      if (isExternalLink(href)) {
        if (canOpenExternally(href)) await Browser.OpenURL(href);
        else notice = `Couldn't open link: only web, email and phone links open outside dotmd`;
        return;
      }
      if (href.startsWith("#") || !currentPath) return; // headings: not supported yet
      await openNote(await resolveLink(currentPath, href));
    } catch (err) {
      notice = `Couldn't open link: ${errorText(err)}`;
    }
  }

  /** Create a note or folder from the sidebar; new notes open right away. */
  async function createEntry(kind: EntryKind, name: string) {
    const path = kind === "note" ? await createNote(name) : await createFolder(name);
    tree = await loadTree();
    if (kind === "note") await openNote(path);
    return path;
  }

  onMount(() => {
    (async () => {
      try {
        tree = await loadTree();
        if (!tree.length) {
          await writeNote("Welcome.md", sample);
          tree = await loadTree();
        }
        void refreshTags();
        void refreshImages();
        const first = firstNote(tree);
        if (first) await openNote(first);
      } catch (err) {
        notice = `Couldn't load notes: ${errorText(err)}`;
      }
    })();

    const onVisibility = () => {
      if (document.visibilityState === "hidden") void save();
    };
    document.addEventListener("visibilitychange", onVisibility);

    // The Go side holds back quitting until pending edits are written.
    // If the save fails, the quit is called off once so the error stays
    // visible; quitting again retries and quits even if it fails again.
    const offFlush = Events.On(FLUSH_EVENT, async () => {
      void Lifecycle.Flushing(); // a slow save gets more time before Go gives up
      if (await save()) {
        await Lifecycle.Saved();
      } else {
        notice += " Quit again to quit without saving.";
        await Lifecycle.Unsaved();
      }
    });

    return () => {
      clearTimeout(imagesTimer);
      document.removeEventListener("visibilitychange", onVisibility);
      offFlush();
    };
  });

  // Autosave once edits pause.
  $effect(() => {
    const content = markdown;
    if (!currentPath || content === lastSaved) return;
    clearTimeout(saveTimer);
    saveTimer = setTimeout(save, SAVE_DELAY_MS);
  });

  // Count words in the body only, ignoring tokens with no letters or digits
  // (markdown syntax such as `#`, `-`, `>`, `|`, `---`, `[x]`).
  const words = $derived(
    splitFrontmatter(markdown)
      .body.split(/\s+/)
      .filter((t) => /[\p{L}\p{N}]/u.test(t)).length,
  );

  // Grow the source textarea with its content so the page, not the textarea, scrolls.
  $effect(() => {
    markdown;
    if (!source) return;
    source.style.height = "auto";
    source.style.height = `${source.scrollHeight}px`;
  });
</script>

<Sidebar.Provider style="--sidebar-width: 350px;">
  <AppSidebar {tree} {tags} {images} activePath={currentPath} onopen={openNote}
    oncreate={createEntry}
    onaddimage={pickImage}
    oninsertimage={insertImage}
    ondeleteimage={removeImage}
  />
  <Sidebar.Inset>
    <header
      class="sticky top-0 flex shrink-0 items-center gap-2 border-b bg-background p-4"
    >
      <Sidebar.Trigger class="-ms-1" />
      <Separator
        orientation="vertical"
        class="me-2 data-vertical:h-4 data-vertical:self-auto"
      />
      <Breadcrumb.Root>
        <Breadcrumb.List>
          {#each folders as folder, i (i)}
            <Breadcrumb.Item class="hidden md:block">
              <Breadcrumb.Page class="text-muted-foreground">{folder}</Breadcrumb.Page>
            </Breadcrumb.Item>
            <Breadcrumb.Separator class="hidden md:block" />
          {/each}
          <Breadcrumb.Item>
            <Breadcrumb.Page
              >{currentPath ? noteTitle(currentPath) : "No note open"}</Breadcrumb.Page
            >
          </Breadcrumb.Item>
        </Breadcrumb.List>
      </Breadcrumb.Root>
    </header>
    <div class="flex flex-1 flex-col gap-4 p-4">
      <div class="flex h-screen flex-col">
        <header
          class="bg-background/80 sticky top-0 z-10 flex h-12 shrink-0 items-center justify-between gap-4 border-b pr-3 pl-20 backdrop-blur"
          style="--wails-draggable: drag"
        >
          <div class="flex min-w-0 items-center gap-2 text-sm">
            <FileText class="text-muted-foreground size-4 shrink-0" />
            <span class="truncate font-medium">{fileName}</span>
            <span class="text-muted-foreground text-xs">{words} words</span>
            {#if notice}
              <span class="truncate text-xs text-destructive" title={notice}>{notice}</span>
            {/if}
          </div>

          <div
            class="bg-muted flex items-center gap-0.5 rounded-lg p-0.5"
            style="--wails-draggable: no-drag"
          >
            <Button
              size="xs"
              variant={mode === "edit" ? "outline" : "ghost"}
              aria-pressed={mode === "edit"}
              onclick={() => setMode("edit")}
            >
              <PenLine data-icon="inline-start" /> Edit
            </Button>
            <Button
              size="xs"
              variant={mode === "markdown" ? "outline" : "ghost"}
              aria-pressed={mode === "markdown"}
              onclick={() => setMode("markdown")}
            >
              <span class="font-mono text-[0.7rem]" data-icon="inline-start"
                >MD</span
              > Markdown
            </Button>
          </div>
        </header>

        <main class="flex-1 overflow-y-auto">
          <div class="mx-auto w-full max-w-3xl px-6 py-10 sm:px-10">
            <!-- Kept mounted while hidden so editor state (history, selection) survives toggling. -->
            <div class:hidden={mode !== "edit"}>
              <Editor
                bind:this={editor}
                bind:value={markdown}
                editable={canEdit && mode === "edit"}
                notePath={currentPath}
                onopenlink={followLink}
                onuploadimage={uploadImage}
                ondownloadimage={fetchImage}
                onerror={(message) => (notice = message)}
              />
            </div>

            {#if mode === "markdown"}
              <textarea
                bind:this={source}
                bind:value={markdown}
                readonly={!canEdit}
                oninput={() => (sourceEdited = true)}
                spellcheck="false"
                aria-label="Markdown source"
                class="text-foreground block min-h-[60vh] w-full resize-none overflow-hidden bg-transparent font-mono text-sm leading-relaxed outline-none"
              ></textarea>
            {/if}
          </div>
        </main>
      </div>
    </div>
  </Sidebar.Inset>
</Sidebar.Provider>
