<script lang="ts">
  import { onMount, tick } from "svelte";
  import { Browser, Events } from "@wailsio/runtime";
  import { Check, FilePlus, LoaderCircle, PenLine, Search, X } from "@lucide/svelte";
  import { Button } from "$lib/components/ui/button";
  import Editor, { type EditorSnapshot } from "$lib/components/editor/Editor.svelte";
  import TabBar from "$lib/components/tab-bar.svelte";
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
  import { errorText } from "$lib/utils";
  import { Service as Lifecycle } from "../bindings/dotmd/internal/lifecycle";
  import {
    createFolder,
    createNote,
    isWithin,
    loadTags,
    loadTree,
    movedPath,
    notePaths,
    noteTitle,
    readNote,
    renameEntry as renameOnDisk,
    resolveLink,
    tagsOf,
    trashEntry,
    withMovedNotes,
    withNoteTags,
    writeNote,
    type EntryKind,
    type Node,
    type Tag,
  } from "$lib/notes/notes";
  import { canOpenExternally, isExternalLink, normalizeHref } from "$lib/notes/links";
  import { loadTabs, movedTab, neighborAfterClose, saveTabs, withTab } from "$lib/notes/tabs";
  import {
    baseName,
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
  /** A reopened note stops holding its scroll position after this long. */
  const RESTORE_WINDOW_MS = 5000;
  /** Input on the note pane that means the user is scrolling it themselves. */
  const SCROLL_INPUT_EVENTS = ["wheel", "touchstart", "pointerdown"] as const;
  /** Closed tabs kept for reopening (⇧⌘T). */
  const CLOSED_TABS_LIMIT = 20;

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
  /** The quick switcher opens its pick in a new tab (⌘T, "+"). */
  let quickOpenNewTab = $state(false);
  let settingsOpen = $state(false);
  /**
   * Open notes, in tab order. The active tab is `currentPath`; a note is
   * open in at most one tab. Remembered across launches.
   */
  let tabs = $state<string[]>([]);
  /** Tabs closed this session, most recent last, for ⇧⌘T. */
  let closedTabs: string[] = [];
  /** Editor state (undo history, selection) of open tabs not shown, by path. */
  const editorStates = new Map<string, EditorSnapshot>();
  /** Notes opened lately, most recent first; remembered across launches. */
  let recent = $state<string[]>(loadRecent());
  let sidebar: AppSidebar;
  /** Desktop sidebar shown; the shadcn sidebar keeps this in a cookie. */
  let sidebarOpen = $state(!document.cookie.includes("sidebar_state=false"));
  let scroller = $state<HTMLElement>();
  /** Where each note opened this session was scrolled to, by path. */
  const scrollPositions = new Map<string, number>();
  /** The open note's scroll restore, while it still holds the note at `top`. */
  let restoring: { top: number; stop: () => void } | undefined;
  let saveTimer: ReturnType<typeof setTimeout> | undefined;
  let treeTimer: ReturnType<typeof setTimeout> | undefined;
  /** Saves run one after another, so an older write never lands last. */
  let saveQueue: Promise<boolean> = Promise.resolve(true);
  /** Bumped per openNote call so only the latest click wins. */
  let openToken = 0;
  /** The note being opened while `opening` is true. */
  let openingPath: string | null = null;
  /** A save on quit failed; cleared (and Go told) once a save succeeds. */
  let quitUnsaved = false;
  /** True while the notes-folder picker is open; editing is paused. */
  let switchingFolder = $state(false);

  const canEdit = $derived(currentPath !== null && !opening && !switchingFolder);
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

  function setMode(next: Mode) {
    if (next === mode) return;
    stopRestoring(); // the other view's height has nothing to do with the offset
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
    if (quitUnsaved) {
      // The failure the user was warned about on quit is over; a new one warns again.
      quitUnsaved = false;
      void Lifecycle.Recovered();
    }
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

  /** Note where the open note is scrolled to, before another replaces it. */
  function rememberScroll() {
    if (!currentPath || !scroller) return;
    // Mid-restore the page may still be too short to reach the saved spot
    // (images loading), so keep the spot rather than where it got to.
    scrollPositions.set(currentPath, restoring ? restoring.top : scroller.scrollTop);
  }

  function stopRestoring() {
    restoring?.stop();
  }

  /**
   * Scroll a newly opened note to `top`. Images take no space until they
   * load, so the offset is re-applied whenever the note's layout changes,
   * until the user scrolls or types, or RESTORE_WINDOW_MS passes.
   */
  async function restoreScroll(top: number) {
    stopRestoring();
    const el = scroller;
    const content = el?.firstElementChild;
    if (!el || !content) return;
    if (top <= 0) {
      el.scrollTop = 0;
      return;
    }
    const apply = () => {
      el.scrollTop = top;
    };
    const observer = new ResizeObserver(apply);
    const onKeydown = (e: KeyboardEvent) => {
      if (!isMod(e)) stop(); // shortcuts (⌘P, ⌘S, …) don't scroll
    };
    let timer: ReturnType<typeof setTimeout> | undefined;
    const stop = () => {
      clearTimeout(timer);
      observer.disconnect();
      for (const type of SCROLL_INPUT_EVENTS) el.removeEventListener(type, stop);
      window.removeEventListener("keydown", onKeydown, true);
      if (restoring?.stop === stop) restoring = undefined;
    };
    restoring = { top, stop };
    // Let the new note (and the source textarea's height) lay out first;
    // a frame callback still runs before the next paint, so nothing flashes.
    await tick();
    await new Promise(requestAnimationFrame);
    if (restoring?.stop !== stop) return; // stopped, or another note opened
    apply();
    observer.observe(content);
    for (const type of SCROLL_INPUT_EVENTS) el.addEventListener(type, stop, { passive: true });
    window.addEventListener("keydown", onKeydown, true);
    timer = setTimeout(stop, RESTORE_WINDOW_MS);
  }

  /**
   * Keep the open note's editor state (undo history, selection) for when its
   * tab is shown again. In markdown mode with source edits the editor is
   * stale, so nothing is kept.
   */
  function stashEditor() {
    if (!currentPath) return;
    const snap = mode === "edit" || !sourceEdited ? editor.snapshot() : null;
    if (snap) editorStates.set(currentPath, snap);
    else editorStates.delete(currentPath);
  }

  /**
   * Show a note. A note already open in a tab switches to that tab; any
   * other replaces the active tab, or gets a tab of its own with `newTab`.
   */
  async function openNote(path: string, { newTab = false } = {}) {
    // Bump first, so clicking back to the open note cancels a pending switch.
    const token = ++openToken;
    if (path === currentPath) {
      opening = false;
      return;
    }
    opening = true;
    openingPath = path;
    try {
      // Keep the current note open if its edits could not be saved.
      if (!(await save()) || token !== openToken) return;
      let content: string;
      try {
        content = await readNote(path);
      } catch (err) {
        // A cancelled open (its file was just renamed or trashed) says nothing.
        if (token === openToken) notice = `Couldn't open ${noteTitle(path)}: ${errorText(err)}`;
        return;
      }
      if (token !== openToken) return;
      rememberScroll();
      stashEditor();
      const from = currentPath;
      lastSaved = content;
      saveFailed = false;
      markdown = content;
      currentPath = path;
      // Its undo history still applies if the file hasn't changed since.
      const snap = editorStates.get(path);
      if (snap?.markdown === content) editor.restore(snap);
      else editor.setMarkdown(content);
      editorStates.delete(path); // the editor holds it now
      sourceEdited = false;
      notice = "";
      remember(path);
      const at = from === null ? -1 : tabs.indexOf(from);
      tabs = withTab(tabs, path, at, !newTab && at >= 0);
      if (from !== null && !tabs.includes(from)) editorStates.delete(from); // replaced
      void restoreScroll(scrollPositions.get(path) ?? 0);
    } finally {
      if (token === openToken) opening = false;
    }
  }

  /**
   * Follow a link from the open note: URLs open in the system browser, and
   * relative paths open the note they point to.
   */
  async function followLink(raw: string, newTab = false) {
    const href = normalizeHref(raw); // "example.com" is a web address, not a note
    try {
      if (isExternalLink(href)) {
        if (canOpenExternally(href)) await Browser.OpenURL(href);
        else notice = `Couldn't open link: only web, email and phone links open outside dotmd`;
        return;
      }
      if (href.startsWith("#") || !currentPath) return; // headings: not supported yet
      await openNote(await resolveLink(currentPath, href), { newTab });
    } catch (err) {
      notice = `Couldn't open link: ${errorText(err)}`;
    }
  }

  /** Drop a tab that is not shown, keeping it for ⇧⌘T. */
  function forgetTab(path: string) {
    // Still opening: cancel it, or it would land in the active tab's place.
    if (opening && openingPath === path) {
      openToken++;
      opening = false;
    }
    tabs = tabs.filter((p) => p !== path);
    editorStates.delete(path);
    closedTabs = [...closedTabs.filter((p) => p !== path), path].slice(-CLOSED_TABS_LIMIT);
  }

  /**
   * Close a tab. Closing the active one shows the tab to its right (else its
   * left) once the note is saved; if it can't be saved, the tab stays open.
   */
  async function closeTab(path: string) {
    if (!tabs.includes(path)) return;
    if (path === currentPath) {
      const next = neighborAfterClose(tabs, path);
      if (next) {
        await openNote(next);
        if (currentPath === path) return; // not saved, or the open failed
      } else {
        if (!(await save())) return;
        if (currentPath === path) closeNote();
      }
    }
    forgetTab(path);
  }

  /** Close every tab but `keep`, which is shown. */
  async function closeOtherTabs(keep: string) {
    if (currentPath !== keep) {
      await openNote(keep);
      if (currentPath !== keep) return;
    }
    for (const p of [...tabs]) if (p !== keep) forgetTab(p);
  }

  /** Close the tabs after `path`; if one of them is shown, `path` is shown instead. */
  async function closeTabsToRight(path: string) {
    const right = tabs.slice(tabs.indexOf(path) + 1);
    if (currentPath !== null && right.includes(currentPath)) {
      await openNote(path);
      if (currentPath !== path) return;
    }
    for (const p of right) forgetTab(p);
  }

  /** Open the most recently closed tab whose note still exists. */
  async function reopenClosedTab() {
    const known = new Set(allNotes);
    while (closedTabs.length) {
      const path = closedTabs.at(-1)!;
      if (!known.has(path) || tabs.includes(path)) {
        closedTabs.pop();
        continue;
      }
      await openNote(path, { newTab: true });
      // Kept if the open failed (say, the shown note couldn't be saved).
      if (currentPath === path) closedTabs = closedTabs.filter((p) => p !== path);
      return;
    }
  }

  /** Show the next (`by` = 1) or previous (-1) tab, wrapping around. */
  function cycleTab(by: 1 | -1) {
    if (!tabs.length) return;
    // Repeated quickly, step on from the tab still opening.
    const from = opening && openingPath !== null ? openingPath : currentPath;
    const i = from === null ? -1 : tabs.indexOf(from);
    const next = i < 0 ? (by > 0 ? 0 : tabs.length - 1) : (i + by + tabs.length) % tabs.length;
    void openNote(tabs[next]);
  }

  /** Show tab `n` (1-based); 9 is always the last tab, as in browsers. */
  function goToTab(n: number) {
    const path = n === 9 ? tabs.at(-1) : tabs[n - 1];
    if (path) void openNote(path);
  }

  function showQuickOpen(newTab: boolean) {
    quickOpenNewTab = newTab;
    quickOpen = true;
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
      // Bring back the tabs open last time, minus notes gone since.
      const paths = notePaths(tree);
      const known = new Set(paths);
      const saved = loadTabs();
      const restored = saved.paths.filter((p) => known.has(p));
      if (restored.length) {
        tabs = restored;
        const active = saved.active !== null && restored.includes(saved.active);
        await openNote(active ? saved.active! : restored[0]);
      } else if (paths[0]) {
        await openNote(paths[0]);
      }
    } catch (err) {
      notice = `Couldn't load notes: ${errorText(err)}`;
    }
  }

  /**
   * Switch to a notes folder the user picks. The open note is saved first;
   * if that fails the folder isn't changed, so no edits are lost.
   */
  async function changeNotesFolder() {
    // In the save queue, so no autosave can run while the picker is open and
    // reach Go after it has switched folders. Editing is paused meanwhile, so
    // an image that finishes saving isn't added to a note about to close.
    const changed = await exclusive(async () => {
      switchingFolder = true;
      try {
        if (!(await writeIfChanged())) {
          throw new Error("The open note couldn't be saved, so the folder wasn't changed.");
        }
        if (!(await settings.chooseNotesFolder())) return false; // cancelled
        // Go has switched folders: drop the old folder's note before
        // anything (an autosave, an await) can write it into the new one.
        closeNote();
        tree = [];
        tags = [];
        saveRecent([]); // recent paths belonged to the old folder
        scrollPositions.clear();
        tabs = [];
        closedTabs = [];
        editorStates.clear();
        // Now, not via the effect: loadNotes reads them back straight away.
        saveTabs({ paths: [], active: null });
        return true;
      } finally {
        switchingFolder = false;
      }
    });
    if (changed) await loadNotes();
  }

  /**
   * Close the open note without saving it, leaving the "pick a note" screen.
   * A note still opening is cancelled too.
   */
  function closeNote() {
    clearTimeout(saveTimer);
    saveTimer = undefined;
    stopRestoring();
    openToken++;
    opening = false;
    currentPath = null;
    lastSaved = null;
    saveFailed = false;
    notice = "";
    markdown = "";
    editor.setMarkdown("");
    sourceEdited = false;
  }

  /**
   * Stop opening a note that is inside `path` (or is it): its file is about
   * to move, and opening it at the old path would let autosave write it back.
   */
  function cancelOpeningWithin(path: string) {
    if (opening && openingPath !== null && isWithin(openingPath, path)) {
      openToken++;
      opening = false;
    }
  }

  /**
   * Run `fn` in the save queue, so no autosave of the open note can write
   * while it moves the note's file. Inside, use writeIfChanged, not save().
   */
  function exclusive<T>(fn: () => Promise<T>): Promise<T> {
    const run = saveQueue.then(fn);
    saveQueue = run.then(
      () => true,
      () => true,
    );
    return run;
  }

  /**
   * Rename a note, folder or image from the sidebar. The open note is saved
   * first and stays open at its new path.
   */
  function renameEntry(path: string, name: string): Promise<string> {
    return exclusive(async () => {
      cancelOpeningWithin(path);
      const affected = currentPath !== null && isWithin(currentPath, path);
      if (affected && !(await writeIfChanged())) {
        throw new Error("The open note couldn't be saved, so it wasn't renamed.");
      }
      const to = await renameOnDisk(path, name);
      if (currentPath) currentPath = movedPath(currentPath, path, to) ?? currentPath;
      const move = (p: string) => movedPath(p, path, to) ?? p;
      saveRecent(recent.map(move));
      tabs = tabs.map(move);
      closedTabs = closedTabs.map(move);
      rekey(scrollPositions, move);
      rekey(editorStates, move);
      tags = withMovedNotes(tags, path, to);
      await refreshTree();
      return to;
    });
  }

  /** Re-key `map` after a rename moved some of its paths. */
  function rekey<T>(map: Map<string, T>, move: (path: string) => string) {
    const entries = [...map].map(([p, v]) => [move(p), v] as const);
    map.clear();
    for (const [p, v] of entries) map.set(p, v);
  }

  /**
   * Move a note, folder or image to the Trash. If that takes the open note
   * with it, the note is saved first (so the Trash holds the latest edits)
   * and then closed, so autosave can't write it back; the nearest tab left
   * is shown instead. Tabs of trashed notes close.
   */
  async function deleteEntry(path: string): Promise<void> {
    const next = await exclusive(async () => {
      cancelOpeningWithin(path);
      const affected = currentPath !== null && isWithin(currentPath, path);
      if (affected && !(await writeIfChanged())) {
        throw new Error("The open note couldn't be saved, so it wasn't deleted.");
      }
      await trashEntry(path);
      const kept = (p: string) => !isWithin(p, path);
      let next: string | null = null;
      if (affected) {
        next = neighborAfterClose(tabs, currentPath!, (p) => !kept(p));
        closeNote();
      }
      tabs = tabs.filter(kept);
      closedTabs = closedTabs.filter(kept);
      saveRecent(recent.filter(kept));
      for (const map of [scrollPositions, editorStates]) {
        for (const p of [...map.keys()]) if (!kept(p)) map.delete(p);
      }
      tags = withMovedNotes(tags, path, null);
      await refreshTree();
      return next;
    });
    // Outside the save queue: opening a note waits on it.
    if (next && currentPath === null) await openNote(next);
  }

  /** Create a note or folder from the sidebar; new notes open right away. */
  async function createEntry(kind: EntryKind, name: string) {
    const path = kind === "note" ? await createNote(name) : await createFolder(name);
    tree = await loadTree();
    if (kind === "note") await openNote(path, { newTab: true });
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
    // visible; quitting again retries and quits even if it fails again,
    // unless a save has worked in between.
    const offFlush = Events.On(FLUSH_EVENT, async () => {
      void Lifecycle.Flushing(); // a slow save gets more time before Go gives up
      if (await save()) {
        await Lifecycle.Saved();
      } else {
        notice += " Quit again to quit without saving.";
        quitUnsaved = true;
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
    const mod = isMod(e);
    const key = shortcutKey(e);
    if (mod && !e.shiftKey && key === "s") {
      e.preventDefault();
      void save();
      return;
    }
    if (dialogOpen() || e.defaultPrevented) return; // e.g. handled by the editor
    // Ctrl+Tab and Ctrl+PageDown/PageUp switch tabs; on macOS Ctrl isn't ⌘.
    if (e.ctrlKey && !e.metaKey && !e.altKey) {
      const by =
        e.key === "Tab" ? (e.shiftKey ? -1 : 1)
        : e.shiftKey ? 0
        : e.key === "PageDown" ? 1
        : e.key === "PageUp" ? -1
        : 0;
      if (by) {
        e.preventDefault();
        cycleTab(by);
        return;
      }
    }
    if (!mod) return;
    // ⌘1–⌘9 by physical key, so AZERTY (where the key types "&" unshifted)
    // works too. Never with Shift: ⌘⇧7/8/9 make lists in the editor, and
    // some layouts type ⌘/ as Shift+7.
    const digit = /^Digit([1-9])$/.exec(e.code);
    if (digit && !e.shiftKey) {
      e.preventDefault();
      goToTab(Number(digit[1]));
      return;
    }
    // Shift is allowed for "/", where layouts need it to type one, and for
    // these tab shortcuts.
    if (e.shiftKey && key !== "/") {
      // Shift turns "[" into "{", so brackets match the physical key.
      if (e.code === "BracketRight") cycleTab(1);
      else if (e.code === "BracketLeft") cycleTab(-1);
      else if (key === "t") void reopenClosedTab();
      else return;
      e.preventDefault();
      return;
    }
    switch (key) {
      case "p":
      case "o":
        showQuickOpen(false);
        break;
      case "t":
        showQuickOpen(true);
        break;
      case "w":
        if (currentPath) void closeTab(currentPath);
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

  // Remember the open tabs for the next launch, once the saved ones are back.
  $effect(() => {
    const state = { paths: tabs, active: currentPath };
    if (loaded) saveTabs(state);
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
    onopen={(path, newTab) => openNote(path, { newTab })}
    oncreate={createEntry}
    oninsertimage={insertImage}
    onrename={renameEntry}
    ondelete={deleteEntry}
    onsearch={() => showQuickOpen(false)}
    onsettings={() => (settingsOpen = true)}
  />
  <Sidebar.Inset class="h-svh overflow-hidden">
    <TabBar
      {tabs}
      active={currentPath}
      class={
        // Clear the macOS window buttons whenever the sidebar isn't beside
        // the tabs: collapsed, or a sheet in a narrow window.
        isMac && (!sidebarOpen || mobile.current) ? "ps-20" : undefined
      }
      onactivate={(path) => void openNote(path)}
      onclose={(path) => void closeTab(path)}
      oncloseothers={(path) => void closeOtherTabs(path)}
      oncloseright={(path) => void closeTabsToRight(path)}
      onreorder={(from, to) => (tabs = movedTab(tabs, from, to))}
      onnew={() => showQuickOpen(true)}
    />
    <header
      class="flex h-12 shrink-0 items-center gap-2 border-b border-border/70 bg-background px-3"
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

        <!-- Not while a note opens, e.g. the tab next to a deleted note. -->
        {#if loaded && !currentPath && !opening}
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
                <Button variant="outline" size="sm" onclick={() => showQuickOpen(false)}>
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

<QuickOpen
  bind:open={quickOpen}
  newTab={quickOpenNewTab}
  notes={allNotes}
  {recent}
  onopen={(path, newTab) => openNote(path, { newTab })}
/>
<SettingsDialog bind:open={settingsOpen} onchoosefolder={changeNotesFolder} />
