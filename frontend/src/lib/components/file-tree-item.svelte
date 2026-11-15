<!--
  One entry of the Files tree: a note, an image, or a folder that renders its
  children (recursively) when expanded. Every row has a "+" menu for creating a
  note or folder in the folder the row belongs to, and a "⋯" menu to rename or
  delete the entry. Clicking an image adds it to the open note.
-->
<script lang="ts">
  import FileTextIcon from "@lucide/svelte/icons/file-text";
  import FolderIcon from "@lucide/svelte/icons/folder";
  import FolderOpenIcon from "@lucide/svelte/icons/folder-open";
  import type { SvelteSet } from "svelte/reactivity";
  import { noteTitle, type EntryKind, type Node } from "$lib/notes/notes";
  import { imageSrc } from "$lib/notes/images";
  import NewEntryMenu from "./new-entry-menu.svelte";
  import NoteActionsMenu from "./note-actions-menu.svelte";
  import TreeRow from "./tree-row.svelte";
  import Self from "./file-tree-item.svelte";

  let {
    node,
    depth = 0,
    activePath,
    expanded,
    onopen,
    onnew,
    oninsertimage,
    onrename,
    ondelete,
  }: {
    node: Node;
    depth?: number;
    activePath: string | null;
    /** Paths of expanded folders, shared by the whole tree. */
    expanded: SvelteSet<string>;
    onopen: (path: string) => void;
    /** Ask to create a note or folder inside `parent` ("" is the root). */
    onnew: (kind: EntryKind, parent: string) => void;
    /** Add an image (root-relative path) to the open note. */
    oninsertimage: (path: string) => void;
    /** Ask to rename an entry (e.g. open a dialog). */
    onrename: (node: Node) => void;
    /** Ask to delete an entry (e.g. open a confirm dialog). */
    ondelete: (node: Node) => void;
  } = $props();

  const title = $derived(
    !node.isImage
      ? node.path
      : `${node.path} — ${activePath ? "click to add to the note" : "open a note to add it"}`,
  );

  function select() {
    if (node.isDir) toggle();
    else if (node.isImage) oninsertimage(node.path);
    else onopen(node.path);
  }

  const isOpen = $derived(node.isDir && expanded.has(node.path));

  /** Folder that new entries from this row go into. */
  const parent = $derived(
    node.isDir ? node.path : node.path.split("/").slice(0, -1).join("/"),
  );

  function toggle() {
    if (isOpen) expanded.delete(node.path);
    else expanded.add(node.path);
  }
</script>

<TreeRow
  {depth}
  label={node.isDir || node.isImage ? node.name : noteTitle(node.path)}
  {title}
  active={!node.isDir && node.path === activePath}
  expandable={node.isDir}
  expanded={isOpen}
  ontoggle={toggle}
  onselect={select}
>
  {#snippet icon()}
    {#if node.isImage}
      <img
        src={imageSrc(null, `/${node.path}`)}
        alt=""
        loading="lazy"
        class="size-4 rounded-sm bg-sidebar-foreground/10 object-cover"
      />
    {:else if !node.isDir}
      <FileTextIcon />
    {:else if isOpen}
      <FolderOpenIcon />
    {:else}
      <FolderIcon />
    {/if}
  {/snippet}
  {#snippet actions()}
    <NewEntryMenu
      location={parent ? parent.split("/").pop()! : "Notes"}
      onnote={() => onnew("note", parent)}
      onfolder={() => onnew("folder", parent)}
    />
    <NoteActionsMenu
      path={node.path}
      {activePath}
      copy={!node.isDir}
      onrename={() => onrename(node)}
      ondelete={() => ondelete(node)}
    />
  {/snippet}
</TreeRow>

{#if isOpen}
  {#each node.children ?? [] as child (child.path)}
    <Self
      node={child}
      depth={depth + 1}
      {activePath}
      {expanded}
      {onopen}
      {onnew}
      {oninsertimage}
      {onrename}
      {ondelete}
    />
  {:else}
    <p
      class="flex h-7 items-center text-sm text-sidebar-foreground/40"
      style:padding-inline-start="{4 + (depth + 1) * 16 + 26}px"
    >
      Empty folder
    </p>
  {/each}
{/if}
