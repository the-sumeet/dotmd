<script lang="ts" module>
  import ArchiveXIcon from "@lucide/svelte/icons/archive-x";
  import FileIcon from "@lucide/svelte/icons/file";
  import InboxIcon from "@lucide/svelte/icons/inbox";
  import SendIcon from "@lucide/svelte/icons/send";
  import Trash2Icon from "@lucide/svelte/icons/trash-2";
  import FileTextIcon from "@lucide/svelte/icons/file-text";
  import HashIcon from "@lucide/svelte/icons/hash";

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
  import NewEntryMenu from "./new-entry-menu.svelte";
  import TreeRow from "./tree-row.svelte";
  import NewEntryDialog from "./new-entry-dialog.svelte";
  import { noteTitle, type EntryKind, type Node, type Tag } from "$lib/notes/notes";
  import type { ComponentProps } from "svelte";
  import { SvelteSet } from "svelte/reactivity";

  let {
    ref = $bindable(null),
    tree = [],
    tags = [],
    activePath = null,
    onopen,
    oncreate,
    ...restProps
  }: ComponentProps<typeof Sidebar.Root> & {
    /** Folders and notes under the notes root. */
    tree?: Node[];
    /** Frontmatter tags and the notes carrying each. */
    tags?: Tag[];
    /** Root-relative path of the open note. */
    activePath?: string | null;
    onopen?: (path: string) => void;
    /**
     * Create a note or folder from a root-relative name (may contain "/").
     * Resolves to the created path; rejects with a user-facing message.
     */
    oncreate?: (kind: EntryKind, path: string) => Promise<string>;
  } = $props();

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
    reveal(created);
    if (kind === "folder") expanded.add(created);
  }

  let activeItem = $state(data.navMain[0]);
  const sidebar = useSidebar();

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
  <Sidebar.Root
    collapsible="none"
    class="!w-[calc(var(--sidebar-width-icon)_+_1px)] border-e"
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


  <Sidebar.Root collapsible="none" class="hidden flex-1 md:flex">
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
            onopen={(path) => onopen?.(path)}
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
                onselect={() => onopen?.(path)}
              >
                {#snippet icon()}<FileTextIcon />{/snippet}
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
    <Sidebar.Rail />
  </Sidebar.Root>
</Sidebar.Root>

<NewEntryDialog
  bind:open={dialogOpen}
  kind={dialogKind}
  parent={dialogParent}
  oncreate={createEntry}
/>
