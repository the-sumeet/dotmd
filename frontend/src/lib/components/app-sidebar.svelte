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
  import DeleteImageDialog from "./delete-image-dialog.svelte";
  import { noteTitle, type EntryKind, type Node, type Tag } from "$lib/notes/notes";
  import type { ComponentProps, Snippet } from "svelte";
  import { SvelteSet } from "svelte/reactivity";
  import { modKey } from "$lib/platform";

  let {
    ref = $bindable(null),
    tree = [],
    tags = [],
    activePath = null,
    onopen,
    oncreate,
    oninsertimage,
    ondeleteimage,
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
    onopen?: (path: string) => void | Promise<void>;
    /**
     * Create a note or folder from a root-relative name (may contain "/").
     * Resolves to the created path; rejects with a user-facing message.
     */
    oncreate?: (kind: EntryKind, path: string) => Promise<string>;
    /** Show an existing image (root-relative path) in the open note. */
    oninsertimage?: (path: string) => void;
    /** Delete an image file; rejects with a user-facing message. */
    ondeleteimage?: (path: string) => Promise<void>;
    /** Open the quick switcher. */
    onsearch?: () => void;
    /** Open the settings dialog. */
    onsettings?: () => void;
  } = $props();

  /** Image the delete dialog asks about. */
  let deletePath = $state("");
  let deleteOpen = $state(false);

  function requestDelete(path: string) {
    deletePath = path;
    deleteOpen = true;
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
  async function openNote(path: string) {
    await onopen?.(path);
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
          ondeleteimage={requestDelete}
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
          onselect={toggle}
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
              onselect={() => openNote(path)}
            >
              {#snippet icon()}<FileTextIcon />{/snippet}
              {#snippet actions()}<NoteActionsMenu {path} {activePath} />{/snippet}
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

<DeleteImageDialog
  bind:open={deleteOpen}
  path={deletePath}
  ondelete={ondeleteimage}
/>
