<script lang="ts" module>
  import ArchiveXIcon from "@lucide/svelte/icons/archive-x";
  import FileIcon from "@lucide/svelte/icons/file";
  import InboxIcon from "@lucide/svelte/icons/inbox";
  import SendIcon from "@lucide/svelte/icons/send";
  import Trash2Icon from "@lucide/svelte/icons/trash-2";
  import FileTextIcon from "@lucide/svelte/icons/file-text";
  import HashIcon from "@lucide/svelte/icons/hash";
  import ImagePlusIcon from "@lucide/svelte/icons/image-plus";

  // This is sample data
  const data = {
    user: {
      name: "shadcn",
      email: "m@example.com",
      avatar: "/avatars/shadcn.jpg",
    },
    navMain: [
      {
        title: "Inbox",
        url: "#",
        icon: InboxIcon,
        isActive: true,
      },
      {
        title: "Drafts",
        url: "#",
        icon: FileIcon,
        isActive: false,
      },
      {
        title: "Sent",
        url: "#",
        icon: SendIcon,
        isActive: false,
      },
      {
        title: "Junk",
        url: "#",
        icon: ArchiveXIcon,
        isActive: false,
      },
      {
        title: "Trash",
        url: "#",
        icon: Trash2Icon,
        isActive: false,
      },
    ],
  };
</script>

<script lang="ts">
  import CommandIcon from "@lucide/svelte/icons/command";
  import * as Sidebar from "$lib/components/ui/sidebar/index.js";
  import { useSidebar } from "$lib/components/ui/sidebar/context.svelte.js";
  import NavUser from "./nav-user.svelte";
  import SidebarSection from "./sidebar-section.svelte";
  import FileTreeItem from "./file-tree-item.svelte";
  import NoteActionsMenu from "./note-actions-menu.svelte";
  import NewEntryMenu from "./new-entry-menu.svelte";
  import TreeRow from "./tree-row.svelte";
  import NewEntryDialog from "./new-entry-dialog.svelte";
  import DeleteImageDialog from "./delete-image-dialog.svelte";
  import { noteTitle, type EntryKind, type Node, type Tag } from "$lib/notes/notes";
  import { imageSrc, type Image } from "$lib/notes/images";
  import type { ComponentProps } from "svelte";
  import { SvelteSet } from "svelte/reactivity";

  let {
    ref = $bindable(null),
    tree = [],
    tags = [],
    activePath = null,
    images = [],
    onopen,
    oncreate,
    onaddimage,
    oninsertimage,
    ondeleteimage,
    ...restProps
  }: ComponentProps<typeof Sidebar.Root> & {
    /** Folders and notes under the notes root. */
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
    /** Every image in the notes folder. */
    images?: Image[];
    /** Choose images from the computer and add them to the open note. */
    onaddimage?: () => void;
    /** Show an existing image (root-relative path) in the open note. */
    oninsertimage?: (path: string) => void;
    /** Delete an image file; rejects with a user-facing message. */
    ondeleteimage?: (path: string) => Promise<void>;
  } = $props();

  /** Image the delete dialog asks about. */
  let deletePath = $state("");
  let deleteOpen = $state(false);

  function requestDelete(path: string) {
    deletePath = path;
    deleteOpen = true;
  }

  /** On phones, show the note the image went into instead of the sheet. */
  function insertImage(path: string) {
    if (!activePath) return;
    oninsertimage?.(path);
    if (sidebar.isMobile) sidebar.setOpenMobile(false);
  }

  function addImage() {
    if (sidebar.isMobile) sidebar.setOpenMobile(false);
    onaddimage?.();
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

  let activeItem = $state(data.navMain[0]);
  const sidebar = useSidebar();

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

<Sidebar.Root
  bind:ref
  collapsible="icon"
  class="overflow-hidden [&>[data-sidebar=sidebar]]:flex-row"
  {...restProps}
>
  <!-- This is the first sidebar -->
  <!-- We disable collapsible and adjust width to icon. -->
  <!-- This will make the sidebar appear as icons. -->
  <!-- Hidden on phones, where the narrow sheet is better spent on the files. -->
  <Sidebar.Root
    collapsible="none"
    class="hidden !w-[calc(var(--sidebar-width-icon)_+_1px)] border-e md:flex"
  >
    <Sidebar.Header>
      <Sidebar.Menu>
        <Sidebar.MenuItem>
          <Sidebar.MenuButton size="lg" class="md:h-8 md:p-0">
            {#snippet child({ props })}
              <a href="##" {...props}>
                <div
                  class="flex aspect-square size-8 items-center justify-center rounded-lg bg-sidebar-primary text-sidebar-primary-foreground"
                >
                  <CommandIcon class="size-4" />
                </div>
                <div class="grid flex-1 text-start text-sm leading-tight">
                  <span class="truncate font-medium">Acme Inc</span>
                  <span class="truncate text-xs">Enterprise</span>
                </div>
              </a>
            {/snippet}
          </Sidebar.MenuButton>
        </Sidebar.MenuItem>
      </Sidebar.Menu>
    </Sidebar.Header>
    <Sidebar.Content>
      <Sidebar.Group>
        <Sidebar.GroupContent class="px-1.5 md:px-0">
          <Sidebar.Menu>
            {#each data.navMain as item (item.title)}
              <Sidebar.MenuItem>
                <Sidebar.MenuButton
                  tooltipContentProps={{
                    hidden: false,
                  }}
                  onclick={() => {
                    activeItem = item;
                    sidebar.setOpen(true);
                  }}
                  isActive={activeItem.title === item.title}
                  class="px-2.5 md:px-2"
                >
                  {#snippet tooltipContent()}
                    {item.title}
                  {/snippet}
                  <item.icon />
                  <span>{item.title}</span>
                </Sidebar.MenuButton>
              </Sidebar.MenuItem>
            {/each}
          </Sidebar.Menu>
        </Sidebar.GroupContent>
      </Sidebar.Group>
    </Sidebar.Content>
    <Sidebar.Footer>
      <NavUser user={data.user} />
    </Sidebar.Footer>
  </Sidebar.Root>


  <Sidebar.Root collapsible="none" class="flex flex-1">
    <!-- Clear the macOS window buttons and let this strip drag the window. -->
    <div class="h-9 shrink-0" style="--wails-draggable: drag"></div>
    <Sidebar.Content>
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

      <SidebarSection title="Images">
        {#snippet actions()}
          <button
            type="button"
            title={activePath ? "Add image to note" : "Open a note to add images"}
            aria-label="Add image to note"
            disabled={!activePath}
            onclick={addImage}
            class="grid size-5 place-items-center rounded text-sidebar-foreground/60 outline-hidden hover:bg-sidebar-foreground/10 hover:text-sidebar-foreground focus-visible:ring-2 focus-visible:ring-sidebar-ring disabled:opacity-40 disabled:hover:bg-transparent [&_svg]:size-4"
          >
            <ImagePlusIcon />
          </button>
        {/snippet}

        {#each images as image (image.path)}
          <TreeRow
            label={image.name}
            title={activePath ? `${image.path} — click to add to the note` : image.path}
            onselect={() => insertImage(image.path)}
          >
            {#snippet icon()}
              <img
                src={imageSrc(null, `/${image.path}`)}
                alt=""
                loading="lazy"
                class="size-5 rounded-sm bg-sidebar-foreground/10 object-cover"
              />
            {/snippet}
            {#snippet actions()}
              <NoteActionsMenu
                path={image.path}
                {activePath}
                ondelete={() => requestDelete(image.path)}
              />
            {/snippet}
          </TreeRow>
        {:else}
          <p class="px-2 py-1 text-sm text-sidebar-foreground/40">
            Paste, drop or add an image to a note
          </p>
        {/each}
      </SidebarSection>
    </Sidebar.Content>
    <Sidebar.Rail />
  </Sidebar.Root>
</Sidebar.Root>

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
