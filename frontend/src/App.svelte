<script lang="ts">
  import { onMount } from "svelte";
  import { Browser, Events } from "@wailsio/runtime";
  import { Check, FilePlus, LoaderCircle, PenLine, Search, X } from "@lucide/svelte";
  import { Button } from "$lib/components/ui/button";
  import Editor from "$lib/components/editor/Editor.svelte";
  import { splitFrontmatter } from "$lib/editor/frontmatter";
  import sample from "$lib/editor/sample.md?raw";
  import * as Breadcrumb from "$lib/components/ui/breadcrumb/index.js";
  import * as Sidebar from "$lib/components/ui/sidebar/index.js";
  import { Separator } from "$lib/components/ui/separator/index.js";
  import AppSidebar from "$lib/components/app-sidebar.svelte";
  import QuickOpen from "$lib/components/quick-open.svelte";
  import SettingsDialog from "$lib/components/settings-dialog.svelte";
  import { settings } from "$lib/settings.svelte";
  import { dialogOpen, isMac, isMod, modKey, shortcutKey } from "$lib/platform";
  import { IsMobile } from "$lib/hooks/is-mobile.svelte";
  import { Service as Lifecycle } from "../bindings/dotmd/internal/lifecycle";
  import {
    createFolder,
    createNote,
    loadTags,
    loadTree,
    notePaths,
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
    relativeHref,
    saveImageFile,
  } from "$lib/notes/images";

  type Mode = "edit" | "markdown";

  /** Saving several images rescans the files once, this long after the last. */
  const TREE_REFRESH_DELAY_MS = 300;
  /** Wait this long after the last change before writing the note to disk. */
  const SAVE_DELAY_MS = 500;
  /** Sent by the Go side before quitting; see internal/lifecycle. */
  const FLUSH_EVENT = "dotmd:flush";
  /** Recently opened notes kept for the quick switcher. */
  const RECENT_LIMIT = 8;
  const RECENT_KEY = "dotmd:recent";

  /** The open note as plain markdown — the single source of truth. */
  let markdown = $state("");
  let mode = $state<Mode>("edit");
  let sourceEdited = false;
  let editor: Editor;
  let source = $state<HTMLTextAreaElement>();

  let tree = $state<Node[]>([]);
  let tags = $state<Tag[]>([]);
  /** Root-relative path of the open note. */
  let currentPath = $state<string | null>(null);
  /** True while switching notes; editing is paused so no keystroke is lost. */
  let opening = $state(false);
  /** Last save/open problem, shown in the note header. */
  let notice = $state("");
  /** Content last written to (or read from) disk. */
  let lastSaved = $state<string | null>(null);
  /** True while a write is in flight. */
  let saving = $state(false);
  /** The last write failed; edits stay unsaved until a save succeeds. */
  let saveFailed = $state(false);
  /** Edits typed in the rich editor that haven't reached `markdown` yet. */
  let editorPending = $state(false);
  const mobile = new IsMobile();
  /** True once the notes folder has been read (or failed to). */
  let loaded = $state(false);
  let quickOpen = $state(false);
  let settingsOpen = $state(false);
  /** Notes opened lately, most recent first; remembered across launches. */
  let recent = $state<string[]>(loadRecent());
  let sidebar: AppSidebar;
  /** Desktop sidebar shown; the shadcn sidebar keeps this in a cookie. */
  let sidebarOpen = $state(!document.cookie.includes("sidebar_state=false"));
  let scroller = $state<HTMLElement>();
  let saveTimer: ReturnType<typeof setTimeout> | undefined;
  let treeTimer: ReturnType<typeof setTimeout> | undefined;
  /** Saves run one after another, so an older write never lands last. */
  let saveQueue: Promise<boolean> = Promise.resolve(true);
  /** Bumped per openNote call so only the latest click wins. */
  let openToken = 0;

  const canEdit = $derived(currentPath !== null && !opening);
  const folders = $derived(currentPath?.split("/").slice(0, -1) ?? []);

  /** Every note path, in sidebar order, for the quick switcher. */
  const allNotes = $derived(notePaths(tree));

  function loadRecent(): string[] {
    try {
      const v = JSON.parse(localStorage.getItem(RECENT_KEY) ?? "[]");
      return Array.isArray(v) ? v.filter((p) => typeof p === "string") : [];
    } catch {
      return [];
    }
  }

  function remember(path: string) {
    saveRecent([path, ...recent.filter((p) => p !== path)].slice(0, RECENT_LIMIT));
  }

  function saveRecent(paths: string[]) {
    recent = paths;
    try {
      localStorage.setItem(RECENT_KEY, JSON.stringify(recent));
    } catch {
      // Recents are a convenience; private storage just means none are kept.
    }
  }

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
      saveFailed = true;
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
    saving = true;
    try {
      await writeNote(path, content);
    } catch (err) {
      notice = `Not saved: ${errorText(err)}`;
      saveFailed = true;
      return false;
    } finally {
      saving = false;
    }
    lastSaved = content;
    saveFailed = false;
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

  async function refreshTree() {
    try {
      tree = await loadTree();
    } catch (err) {
      console.error("Failed to load files", err);
    }
  }

  /** Show new images in the sidebar once a burst of saves (a drop of many files) ends. */
  function scheduleTreeRefresh() {
    clearTimeout(treeTimer);
    treeTimer = setTimeout(refreshTree, TREE_REFRESH_DELAY_MS);
  }

  /** The open note, for saving an image beside it. */
  function noteForImages(): string {
    if (!currentPath || opening) throw new Error("open a note first");
    return currentPath;
  }

  /** Save a pasted, dropped or picked image beside the open note. */
  async function uploadImage(file: File): Promise<string> {
    const href = await saveImageFile(noteForImages(), file);
    scheduleTreeRefresh();
    return href;
  }

  /** Keep a local copy of a pasted image URL beside the open note. */
  async function fetchImage(url: string): Promise<string> {
    const href = await downloadImage(noteForImages(), url);
    scheduleTreeRefresh();
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

  async function removeImage(path: string) {
    await deleteImage(path);
    await refreshTree();
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
      saveFailed = false;
      markdown = content;
      currentPath = path;
      editor.setMarkdown(content);
      sourceEdited = false;
      notice = "";
      remember(path);
      if (scroller) scroller.scrollTop = 0;
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

  /**
   * Read the notes folder and open its first note. An empty default folder
   * gets the Welcome note; a folder the user picked is left as it is, and so
   * is the default while it stands in for a missing one.
   */
  async function loadNotes() {
    try {
      tree = await loadTree();
      if (!tree.length && settings.defaultFolder && !settings.missingFolder) {
        await writeNote("Welcome.md", sample);
        tree = await loadTree();
      }
      void refreshTags();
      const first = notePaths(tree)[0];
      if (first) await openNote(first);
    } catch (err) {
      notice = `Couldn't load notes: ${errorText(err)}`;
    }
  }

  /**
   * Switch to a notes folder the user picks. The open note is saved first;
   * if that fails the folder isn't changed, so no edits are lost.
   */
  async function changeNotesFolder() {
    if (!(await save())) {
      throw new Error("The open note couldn't be saved, so the folder wasn't changed.");
    }
    if (!(await settings.chooseNotesFolder())) return; // cancelled
    // Go has switched folders: drop the old folder's note before anything
    // (an autosave, an await) can write it into the new one.
    openToken++; // a note still opening belongs to the old folder
    opening = false;
    currentPath = null;
    lastSaved = null;
    saveFailed = false;
    notice = "";
    markdown = "";
    editor.setMarkdown("");
    sourceEdited = false;
    tree = [];
    tags = [];
    saveRecent([]); // recent paths belonged to the old folder
    await loadNotes();
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
      await settings.load();
      await loadNotes();
      if (settings.missingFolder) {
        notice = `Couldn't find ${settings.missingFolder}; showing ${settings.notesFolder}. Reconnect it and restart, or pick another folder in Settings.`;
      }
      loaded = true;
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
      clearTimeout(treeTimer);
      document.removeEventListener("visibilitychange", onVisibility);
      offFlush();
    };
  });

  /**
   * App-wide shortcuts. Editor shortcuts (⌘B, ⌘I, …) are left alone, and
   * while a dialog is open only ⌘S works, so its contents aren't swapped
   * or covered.
   */
  function onKeydown(e: KeyboardEvent) {
    if (!isMod(e)) return;
    const key = shortcutKey(e);
    // Shift is allowed for "/" only, where layouts need it to type one.
    if (e.shiftKey && key !== "/") return;
    if (key === "s") {
      e.preventDefault();
      void save();
      return;
    }
    if (dialogOpen()) return;
    switch (key) {
      case "p":
      case "o":
        quickOpen = true;
        break;
      case "n":
        sidebar.newNote();
        break;
      case ",":
        settingsOpen = true;
        break;
      case "/":
        if (!canEdit) return;
        setMode(mode === "edit" ? "markdown" : "edit");
        break;
      default:
        return;
    }
    e.preventDefault();
  }

  type SaveStatus = "saving" | "saved" | "unsaved" | null;
  const saveStatus = $derived.by<SaveStatus>(() => {
    if (!currentPath || notice) return null;
    if (saving) return "saving";
    const dirty = editorPending || markdown !== lastSaved;
    if (!dirty) return "saved";
    // A failed save isn't retried until the next edit, so don't claim one is running.
    return saveFailed ? "unsaved" : "saving";
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

<svelte:window onkeydown={onKeydown} />

<Sidebar.Provider bind:open={sidebarOpen} style="--sidebar-width: 280px;">
  <AppSidebar
    bind:this={sidebar}
    {tree}
    {tags}
    activePath={currentPath}
    onopen={openNote}
    oncreate={createEntry}
    oninsertimage={insertImage}
    ondeleteimage={removeImage}
    onsearch={() => (quickOpen = true)}
    onsettings={() => (settingsOpen = true)}
  />
  <Sidebar.Inset class="h-svh overflow-hidden">
    <header
      class={[
        "flex h-12 shrink-0 items-center gap-2 border-b border-border/70 bg-background px-3",
        // Clear the macOS window buttons whenever the sidebar isn't beside
        // the header: collapsed, or a sheet in a narrow window.
        isMac && (!sidebarOpen || mobile.current) && "ps-20",
      ]}
      style="--wails-draggable: drag"
    >
      <Sidebar.Trigger
        class="text-muted-foreground"
        title="Toggle sidebar ({modKey}\)"
        style="--wails-draggable: no-drag"
      />
      <Separator orientation="vertical" class="data-vertical:h-4 data-vertical:self-auto" />

      <Breadcrumb.Root class="min-w-0 ps-1">
        <Breadcrumb.List class="flex-nowrap">
          {#each folders as folder, i (i)}
            <Breadcrumb.Item class="hidden md:inline-flex">
              <Breadcrumb.Page class="text-muted-foreground">{folder}</Breadcrumb.Page>
            </Breadcrumb.Item>
            <Breadcrumb.Separator class="hidden md:block" />
          {/each}
          <Breadcrumb.Item class="min-w-0">
            <Breadcrumb.Page class="truncate font-medium" title={currentPath ?? undefined}>
              {currentPath ? noteTitle(currentPath) : "dotmd"}
            </Breadcrumb.Page>
          </Breadcrumb.Item>
        </Breadcrumb.List>
      </Breadcrumb.Root>

      <div class="ms-auto flex min-w-0 items-center gap-3 text-xs text-muted-foreground">
        {#if notice}
          <div
            role="alert"
            class="flex min-w-0 items-center gap-1 rounded-md bg-destructive/10 py-0.5 ps-2 pe-0.5 text-destructive"
            style="--wails-draggable: no-drag"
          >
            <span class="truncate" title={notice}>{notice}</span>
            <button
              type="button"
              aria-label="Dismiss"
              class="grid size-5 shrink-0 place-items-center rounded hover:bg-destructive/15"
              onclick={() => (notice = "")}
            >
              <X class="size-3.5" />
            </button>
          </div>
        {:else if saveStatus}
          <span class="flex shrink-0 items-center gap-1" aria-live="polite">
            {#if saveStatus === "saving"}
              <LoaderCircle class="size-3 animate-spin" /> Saving…
            {:else if saveStatus === "unsaved"}
              <span class="text-destructive">Not saved</span>
              <button
                type="button"
                class="rounded px-1 font-medium text-foreground hover:bg-muted"
                style="--wails-draggable: no-drag"
                onclick={() => void save()}
              >
                Retry
              </button>
            {:else}
              <Check class="size-3" /> Saved
            {/if}
          </span>
        {/if}
        {#if currentPath}
          <span class="hidden shrink-0 tabular-nums sm:inline">
            {words.toLocaleString()} {words === 1 ? "word" : "words"}
          </span>
        {/if}
      </div>

      {#if currentPath}
        <div
          class="flex shrink-0 items-center gap-0.5 rounded-lg bg-muted p-0.5"
          style="--wails-draggable: no-drag"
        >
          <Button
            size="xs"
            variant={mode === "edit" ? "outline" : "ghost"}
            aria-pressed={mode === "edit"}
            title="Rich editor ({modKey}/)"
            onclick={() => setMode("edit")}
          >
            <PenLine data-icon="inline-start" /> <span class="max-sm:sr-only">Edit</span>
          </Button>
          <Button
            size="xs"
            variant={mode === "markdown" ? "outline" : "ghost"}
            aria-pressed={mode === "markdown"}
            title="Markdown source ({modKey}/)"
            onclick={() => setMode("markdown")}
          >
            <span class="font-mono text-[0.7rem]" data-icon="inline-start">MD</span>
            <span class="max-sm:sr-only">Markdown</span>
          </Button>
        </div>
      {/if}
    </header>

    <div bind:this={scroller} class="flex-1 overflow-y-auto">
      <div
        class={[
          "mx-auto w-full px-6 py-10 sm:px-12",
          settings.wideLayout ? "max-w-5xl" : "max-w-3xl",
        ]}
      >
        <!-- Kept mounted while hidden so editor state (history, selection) survives toggling. -->
        <div class:hidden={mode !== "edit" || !currentPath}>
          <Editor
            bind:this={editor}
            bind:value={markdown}
            bind:pending={editorPending}
            editable={canEdit && mode === "edit"}
            notePath={currentPath}
            onopenlink={followLink}
            onuploadimage={uploadImage}
            ondownloadimage={fetchImage}
            onerror={(message) => (notice = message)}
          />
        </div>

        {#if mode === "markdown" && currentPath}
          <textarea
            bind:this={source}
            bind:value={markdown}
            readonly={!canEdit}
            oninput={() => (sourceEdited = true)}
            spellcheck="false"
            aria-label="Markdown source"
            class="block min-h-[60vh] w-full resize-none overflow-hidden bg-transparent font-mono text-sm leading-relaxed text-foreground outline-none"
          ></textarea>
        {/if}

        {#if loaded && !currentPath}
          <div class="flex flex-col items-center gap-4 pt-[18vh] text-center">
            <p class="text-sm text-muted-foreground">
              {allNotes.length ? "Pick a note from the sidebar, or" : "You don't have any notes yet."}
            </p>
            <div class="flex flex-wrap justify-center gap-2">
              <Button variant="outline" size="sm" onclick={() => sidebar.newNote()}>
                <FilePlus data-icon="inline-start" /> New note
                <kbd class="ms-1 font-sans text-muted-foreground">{modKey}N</kbd>
              </Button>
              {#if allNotes.length}
                <Button variant="outline" size="sm" onclick={() => (quickOpen = true)}>
                  <Search data-icon="inline-start" /> Go to note
                  <kbd class="ms-1 font-sans text-muted-foreground">{modKey}P</kbd>
                </Button>
              {/if}
            </div>
          </div>
        {/if}
      </div>
    </div>
  </Sidebar.Inset>
</Sidebar.Provider>

<QuickOpen bind:open={quickOpen} notes={allNotes} {recent} onopen={openNote} />
<SettingsDialog bind:open={settingsOpen} onchoosefolder={changeNotesFolder} />
