<script lang="ts" module>
  import FileTextIcon from "@lucide/svelte/icons/file-text";
  import FilePlusIcon from "@lucide/svelte/icons/file-plus";
  import HashIcon from "@lucide/svelte/icons/hash";
  import SearchIcon from "@lucide/svelte/icons/search";
  import SettingsIcon from "@lucide/svelte/icons/settings";
</script>

<script lang="ts">
  import * as Sidebar from "$lib/components/ui/sidebar/index.js";
  import { useSidebar } from "$lib/components/ui/sidebar/context.svelte.js";
  import SidebarSection from "./sidebar-section.svelte";
  import FileTreeItem from "./file-tree-item.svelte";
  import NoteActionsMenu from "./note-actions-menu.svelte";
  import NewEntryMenu from "./new-entry-menu.svelte";
  import TreeRow from "./tree-row.svelte";
  import NewEntryDialog from "./new-entry-dialog.svelte";
  import DeleteEntryDialog from "./delete-entry-dialog.svelte";
  import RenameEntryDialog from "./rename-entry-dialog.svelte";
  import {
    isWithin,
    kindOf,
    movedPath,
    notePaths,
    noteTitle,
    type EntryKind,
    type FileKind,
    type Node,
    type Tag,
  } from "$lib/notes/notes";
  import type { ComponentProps, Snippet } from "svelte";
  import { SvelteSet } from "svelte/reactivity";
  import { isMod, modKey } from "$lib/platform";

  let {
    ref = $bindable(null),
    tree = [],
    tags = [],
    activePath = null,
    onopen,
    oncreate,
    oninsertimage,
    onrename,
    ondelete,
    onsearch,
    onsettings,
    ...restProps
  }: ComponentProps<typeof Sidebar.Root> & {
    /** Folders, notes and images under the notes root. */
    tree?: Node[];
    /** Frontmatter tags and the notes carrying each. */
    tags?: Tag[];
    /** Root-relative path of the open note. */
    activePath?: string | null;
    /** Open a note, in a new tab or in place of the active one. */
    onopen?: (path: string, newTab: boolean) => void | Promise<void>;
    /**
     * Create a note or folder from a root-relative name (may contain "/").
     * Resolves to the created path; rejects with a user-facing message.
     */
    oncreate?: (kind: EntryKind, path: string) => Promise<string>;
    /** Show an existing image (root-relative path) in the open note. */
    oninsertimage?: (path: string) => void;
    /**
     * Rename a note, folder or image in place. Resolves to its new path;
     * rejects with a user-facing message.
     */
    onrename?: (path: string, name: string) => Promise<string>;
    /** Move a note, folder or image to the Trash; rejects with a user-facing message. */
    ondelete?: (path: string) => Promise<void>;
    /** Open the quick switcher. */
    onsearch?: () => void;
    /** Open the settings dialog. */
    onsettings?: () => void;
  } = $props();

  type Target = { path: string; kind: FileKind };

  /** Entry the rename dialog asks about. */
  let renameTarget = $state<Target>({ path: "", kind: "note" });
  let renameOpen = $state(false);

  /** Entry the delete dialog asks about; for a folder, how many notes it holds. */
  let deleteTarget = $state<Target & { noteCount: number }>({ path: "", kind: "note", noteCount: 0 });
  let deleteOpen = $state(false);

  function requestRename(path: string, kind: FileKind) {
    renameTarget = { path, kind };
    renameOpen = true;
  }

  function requestDelete(path: string, kind: FileKind, noteCount = 0) {
    deleteTarget = { path, kind, noteCount };
    deleteOpen = true;
  }

  const renameNode = (node: Node) => requestRename(node.path, kindOf(node));
  const deleteNode = (node: Node) =>
    requestDelete(node.path, kindOf(node), node.isDir ? notePaths(node.children ?? []).length : 0);

  async function renameEntry(path: string, name: string) {
    if (!onrename) return;
    const to = await onrename(path, name);
    // Keep a renamed folder, and folders inside it, expanded.
    for (const p of [...expanded]) {
      const moved = movedPath(p, path, to);
      if (moved !== null && moved !== p) {
        expanded.delete(p);
        expanded.add(moved);
      }
    }
    reveal(to);
  }

  async function deleteEntry(path: string) {
    await ondelete?.(path);
    for (const p of [...expanded]) if (isWithin(p, path)) expanded.delete(p);
  }

  /**
   * Add an image to the open note. With no note open, App says why instead.
   * On phones, close the sheet to show the note (or that message).
   */
  function insertImage(path: string) {
    oninsertimage?.(path);
    if (sidebar.isMobile) sidebar.setOpenMobile(false);
  }

  let filesOpen = $state(true);

  /** State of the "new note / new folder" dialog. */
  let dialogOpen = $state(false);
  let dialogKind = $state<EntryKind>("note");
  let dialogParent = $state("");

  function requestNew(kind: EntryKind, parent: string) {
    dialogKind = kind;
    dialogParent = parent;
    dialogOpen = true;
  }

  async function createEntry(kind: EntryKind, path: string) {
    if (!oncreate) return;
    const created = await oncreate(kind, path);
    // A new note opens right away; show it instead of the sheet.
    if (kind === "note" && sidebar.isMobile && activePath === created) sidebar.setOpenMobile(false);
    reveal(created);
    if (kind === "folder") expanded.add(created);
  }

  const sidebar = useSidebar();

  /** Ask for a new note's name; it goes in the open note's folder. */
  export function newNote() {
    requestNew("note", activePath?.split("/").slice(0, -1).join("/") ?? "");
  }

  function search() {
    if (sidebar.isMobile) sidebar.setOpenMobile(false);
    onsearch?.();
  }

  function openSettings() {
    if (sidebar.isMobile) sidebar.setOpenMobile(false);
    onsettings?.();
  }

  /**
   * Open a note; on phones also close the sidebar sheet so the note is
   * visible. The sheet stays open if the switch failed (e.g. the current note
   * could not be saved), so the user can retry.
   */
  async function openNote(path: string, newTab = false) {
    await onopen?.(path, newTab);
    if (sidebar.isMobile && activePath === path) sidebar.setOpenMobile(false);
  }

  /** Paths of expanded folders. */
  const expanded = new SvelteSet<string>();
  /** Names of expanded tags. */
  const expandedTags = new SvelteSet<string>();

  /** Expand every folder above `path`. */
  function reveal(path: string) {
    const parts = path.split("/").slice(0, -1);
    for (let i = 1; i <= parts.length; i++) {
      expanded.add(parts.slice(0, i).join("/"));
    }
  }

  // Reveal the open note.
  $effect(() => {
    if (activePath) reveal(activePath);
  });
</script>

<Sidebar.Root bind:ref collapsible="offcanvas" {...restProps}>
  <!-- Clear the macOS window buttons and let this strip drag the window. -->
  <div class="h-10 shrink-0" style="--wails-draggable: drag"></div>
  <div class="flex flex-col gap-px px-2">
    {@render action("Search", `${modKey}P`, search, searchIcon)}
    {@render action("New note", `${modKey}N`, newNote, newIcon)}
  </div>
  <Sidebar.Content class="pb-4">
    <SidebarSection title="Files" bind:open={filesOpen}>
      {#snippet actions()}
        <NewEntryMenu
          location="Notes"
          onnote={() => requestNew("note", "")}
          onfolder={() => requestNew("folder", "")}
        />
      {/snippet}

      {#each tree as node (node.path)}
        <FileTreeItem
          {node}
          {activePath}
          {expanded}
          onopen={openNote}
          onnew={requestNew}
          oninsertimage={insertImage}
          onrename={renameNode}
          ondelete={deleteNode}
        />
      {:else}
        <p class="px-2 py-1 text-sm text-sidebar-foreground/40">No notes yet</p>
      {/each}
    </SidebarSection>

    <SidebarSection title="Tags">
      {#each tags as tag (tag.name)}
        {@const open = expandedTags.has(tag.name)}
        {@const toggle = () =>
          open ? expandedTags.delete(tag.name) : expandedTags.add(tag.name)}
        <TreeRow
          label={tag.name}
          expandable
          expanded={open}
          trailing={String(tag.notes.length)}
          ontoggle={toggle}
          onselect={(e) => e.button === 0 && toggle()}
        >
          {#snippet icon()}<HashIcon />{/snippet}
        </TreeRow>
        {#if open}
          {#each tag.notes as path (path)}
            <TreeRow
              depth={1}
              label={noteTitle(path)}
              title={path}
              active={path === activePath}
              onselect={(e) => openNote(path, isMod(e) || e.button === 1)}
            >
              {#snippet icon()}<FileTextIcon />{/snippet}
              {#snippet actions()}
                <NoteActionsMenu
                  {path}
                  {activePath}
                  onrename={() => requestRename(path, "note")}
                  ondelete={() => requestDelete(path, "note")}
                />
              {/snippet}
            </TreeRow>
          {/each}
        {/if}
      {:else}
        <p class="px-2 py-1 text-sm text-sidebar-foreground/40">
          Add <code>tags: [idea]</code> to a note's frontmatter
        </p>
      {/each}
    </SidebarSection>
  </Sidebar.Content>
  <Sidebar.Footer class="pt-0">
    {@render action("Settings", `${modKey},`, openSettings, settingsIcon)}
  </Sidebar.Footer>
  <Sidebar.Rail />
</Sidebar.Root>

{#snippet searchIcon()}<SearchIcon />{/snippet}
{#snippet newIcon()}<FilePlusIcon />{/snippet}
{#snippet settingsIcon()}<SettingsIcon />{/snippet}

<!-- A sidebar command row with its keyboard shortcut on hover. -->
{#snippet action(label: string, shortcut: string, onclick: () => void, icon: Snippet)}
  <button
    type="button"
    {onclick}
    class="group/action flex h-7 items-center gap-1.5 rounded-md ps-1 pe-2 text-sm font-medium text-sidebar-foreground/70 outline-hidden hover:bg-sidebar-accent hover:text-sidebar-accent-foreground focus-visible:ring-2 focus-visible:ring-sidebar-ring"
  >
    <span class="grid size-5 place-items-center text-sidebar-foreground/50 [&_svg]:size-4">
      {@render icon()}
    </span>
    {label}
    <kbd
      class="ms-auto font-sans text-xs text-sidebar-foreground/40 opacity-0 group-hover/action:opacity-100 max-md:hidden"
      >{shortcut}</kbd
    >
  </button>
{/snippet}

<NewEntryDialog
  bind:open={dialogOpen}
  kind={dialogKind}
  parent={dialogParent}
  oncreate={createEntry}
/>

<RenameEntryDialog
  bind:open={renameOpen}
  kind={renameTarget.kind}
  path={renameTarget.path}
  onrename={renameEntry}
/>

<DeleteEntryDialog
  bind:open={deleteOpen}
  kind={deleteTarget.kind}
  path={deleteTarget.path}
  noteCount={deleteTarget.noteCount}
  holdsOpenNote={activePath !== null && isWithin(activePath, deleteTarget.path)}
  ondelete={deleteEntry}
/>
