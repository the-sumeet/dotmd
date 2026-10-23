<script lang="ts">
  import ChevronRightIcon from "@lucide/svelte/icons/chevron-right";
  import FileIcon from "@lucide/svelte/icons/file";
  import FolderIcon from "@lucide/svelte/icons/folder";
  import PencilIcon from "@lucide/svelte/icons/pencil";
  import CopyIcon from "@lucide/svelte/icons/copy";
  import ClipboardIcon from "@lucide/svelte/icons/clipboard";
  import * as AlertDialog from "$lib/components/ui/alert-dialog/index.js";
  import { Button } from "$lib/components/ui/button/index.js";
  import * as Collapsible from "$lib/components/ui/collapsible/index.js";
  import * as ContextMenu from "$lib/components/ui/context-menu/index.js";
  import * as Dialog from "$lib/components/ui/dialog/index.js";
  import { Input } from "$lib/components/ui/input/index.js";
  import * as Sidebar from "$lib/components/ui/sidebar/index.js";
  import SunIcon from "@lucide/svelte/icons/sun";
  import MoonIcon from "@lucide/svelte/icons/moon";
  import { toggleMode } from "mode-watcher";
  import { goto } from "$app/navigation";
  import { page } from "$app/state";
  import { mergeProps } from "bits-ui";
  import type { ComponentProps } from "svelte";
  import { onMount, tick, untrack } from "svelte";
  import { toast } from "svelte-sonner";
  import { Clipboard } from "@wailsio/runtime";
  import { windowState } from "$lib/stores/window.svelte.js";
  import { editor } from "$lib/stores/editor.svelte.js";
  import { files } from "$lib/stores/files.svelte.js";
  import { relativeNotePath } from "$lib/note-links.js";
  import { FileService } from "../../../bindings/changeme";
  import type { FileNode } from "../../../bindings/changeme/models.js";
  import CommandIcon from "@lucide/svelte/icons/command";
  import SettingsIcon from "@lucide/svelte/icons/settings";
  import NotebookTextIcon from "@lucide/svelte/icons/notebook-text";
  import Trash2Icon from "@lucide/svelte/icons/trash-2";
  import { useSidebar } from "$lib/components/ui/sidebar/context.svelte.js";
  import NavUser from "./nav-user.svelte";
  let {
    ref = $bindable(null),
    ...restProps
  }: ComponentProps<typeof Sidebar.Root> = $props();

  const data = {
    user: {
      name: "shadcn",
      email: "m@example.com",
      avatar: "/avatars/shadcn.jpg",
    },
    navMain: [
      {
        title: "Notes",
        url: "/",
        icon: NotebookTextIcon,
        openSidebar: true,
      },
      {
        title: "Settings",
        url: "/settings",
        icon: SettingsIcon,
      },
    ],
  };
  onMount(() => {
    void files.refresh();
  });

  // Expand the folders above the open file and scroll it into view.
  $effect(() => {
    const path = editor.path;
    if (path === null) return;
    untrack(() => files.reveal(path));
    void tick().then(() =>
      document
        .querySelector(`[data-path="${CSS.escape(path)}"]`)
        ?.scrollIntoView({ block: "nearest" }),
    );
  });

  // Rename / delete dialogs, opened from an item's context menu.
  let target = $state<FileNode | null>(null);
  let renameOpen = $state(false);
  let deleteOpen = $state(false);
  let newName = $state("");
  let renameInput = $state<HTMLInputElement | null>(null);

  function startRename(node: FileNode) {
    target = node;
    newName = node.name;
    renameOpen = true;
  }

  function startDelete(node: FileNode) {
    target = node;
    deleteOpen = true;
  }

  async function submitRename() {
    if (!target) return;
    const name = newName.trim();
    const problem =
      name === ""
        ? "Name can't be empty."
        : /[/\\]/.test(name)
          ? "Name can't contain / or \\."
          : name.startsWith(".")
            ? "Names starting with . are hidden from the file list."
            : new TextEncoder().encode(name).length > 255
              ? "Name is too long (max 255 bytes)."
              : null;
    if (problem) {
      toast.error("Invalid name", { description: problem });
      return;
    }
    try {
      await files.rename(target.path, name);
      renameOpen = false;
    } catch (err) {
      toast.error("Couldn't rename", { description: String(err) });
    }
  }

  async function confirmDelete() {
    if (!target) return;
    const node = target;
    deleteOpen = false;
    try {
      await files.delete(node.path);
      toast.success(`Deleted ${node.name}`);
    } catch (err) {
      toast.error("Couldn't delete", { description: String(err) });
    }
  }

  async function duplicate(node: FileNode) {
    try {
      const copy = await files.duplicate(node.path);
      toast.success(`Created ${copy.split("/").pop()}`);
    } catch (err) {
      toast.error("Couldn't duplicate", { description: String(err) });
    }
  }

  async function copyPath(node: FileNode, relativeToNote: boolean) {
    try {
      await Clipboard.SetText(
        relativeToNote && editor.path !== null
          ? relativeNotePath(editor.path, node.path)
          : await FileService.AbsPath(node.path),
      );
      toast.success("Copied path to clipboard");
    } catch (err) {
      toast.error("Couldn't copy path", { description: String(err) });
    }
  }

  function isActive(url: string) {
    const current = page.url.pathname;
    return url === "/"
      ? current === "/"
      : current === url || current.startsWith(url + "/");
  }

  const sidebar = useSidebar();
</script>

<Sidebar.Root
  bind:ref
  collapsible="icon"
  class="overflow-hidden [&>[data-sidebar=sidebar]]:flex-row"
  {...restProps}
>
  <Sidebar.Root
    collapsible="none"
    class="!w-[calc(var(--sidebar-width-icon)_+_1px)] border-e"
  >
    <!-- Push the logo below the macOS traffic lights (hidden-inset title bar). -->
    <Sidebar.Header class={windowState.fullscreen ? "" : "pt-12"}>
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
                    void goto(item.url);
                    if (item.openSidebar) {
                      sidebar.setOpen(true);
                    }
                  }}
                  isActive={isActive(item.url)}
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
      <Sidebar.Menu>
        <Sidebar.MenuItem>
          <Sidebar.MenuButton
            tooltipContentProps={{ hidden: false }}
            onclick={toggleMode}
            class="px-2.5 md:px-2"
          >
            {#snippet tooltipContent()}
              Toggle theme
            {/snippet}
            <SunIcon class="dark:hidden" />
            <MoonIcon class="hidden dark:block" />
            <span>Toggle theme</span>
          </Sidebar.MenuButton>
        </Sidebar.MenuItem>
      </Sidebar.Menu>
      <NavUser user={data.user} />
    </Sidebar.Footer>
  </Sidebar.Root>

  <!-- File explorer -->
  <!-- min-w-0 so the panel can't grow past the sidebar to fit a long name, and the name truncates instead. -->
  <Sidebar.Root
    collapsible="none"
    class="hidden min-w-0 flex-1 md:flex"
    inert={!sidebar.open}
    bind:ref
    {...restProps}
  >
    <!-- Reserve space for the macOS traffic lights (hidden-inset title bar) and keep it draggable. -->
    <Sidebar.Header
      class={["shrink-0", windowState.fullscreen ? "h-4" : "h-12"]}
      style="--wails-draggable: drag"
    />
    <Sidebar.Content>
      <Sidebar.Group>
        <Sidebar.GroupLabel>Files</Sidebar.GroupLabel>
        <Sidebar.GroupContent>
          <Sidebar.Menu>
            {#each files.tree as node (node.path)}
              {@render Tree({ node })}
            {/each}
          </Sidebar.Menu>
        </Sidebar.GroupContent>
      </Sidebar.Group>
    </Sidebar.Content>
    <Sidebar.Rail />
  </Sidebar.Root>
</Sidebar.Root>

<Dialog.Root bind:open={renameOpen}>
  <Dialog.Content
    class="sm:max-w-sm"
    onOpenAutoFocus={(e) => {
      // Focus the input with the name selected, minus the extension for files.
      e.preventDefault();
      renameInput?.focus();
      const dot = target && !target.isDir ? newName.lastIndexOf(".") : -1;
      renameInput?.setSelectionRange(0, dot > 0 ? dot : newName.length);
    }}
  >
    <form
      class="grid gap-4"
      onsubmit={(e) => {
        e.preventDefault();
        void submitRename();
      }}
    >
      <Dialog.Header>
        <Dialog.Title>Rename {target?.isDir ? "folder" : "file"}</Dialog.Title>
      </Dialog.Header>
      <Input
        bind:ref={renameInput}
        bind:value={newName}
        aria-label="New name"
      />
      <Dialog.Footer>
        <Button
          type="button"
          variant="outline"
          onclick={() => (renameOpen = false)}>Cancel</Button
        >
        <Button type="submit">Rename</Button>
      </Dialog.Footer>
    </form>
  </Dialog.Content>
</Dialog.Root>

<AlertDialog.Root bind:open={deleteOpen}>
  <AlertDialog.Content>
    <AlertDialog.Header>
      <AlertDialog.Title class="wrap-anywhere"
        >Delete {target?.name}?</AlertDialog.Title
      >
      <AlertDialog.Description>
        {target?.isDir
          ? "This permanently deletes the folder and everything in it."
          : "This permanently deletes the file."}
      </AlertDialog.Description>
    </AlertDialog.Header>
    <AlertDialog.Footer>
      <AlertDialog.Cancel>Cancel</AlertDialog.Cancel>
      <AlertDialog.Action
        variant="destructive"
        onclick={() => void confirmDelete()}
      >
        Delete
      </AlertDialog.Action>
    </AlertDialog.Footer>
  </AlertDialog.Content>
</AlertDialog.Root>

{#snippet ItemMenu(node: FileNode)}
  <ContextMenu.Content class="w-56">
    <ContextMenu.Item onSelect={() => startRename(node)}>
      <PencilIcon />
      Rename
    </ContextMenu.Item>
    <ContextMenu.Item onSelect={() => void duplicate(node)}>
      <CopyIcon />
      Duplicate
    </ContextMenu.Item>
    <ContextMenu.Separator />
    <ContextMenu.Item onSelect={() => void copyPath(node, false)}>
      <ClipboardIcon />
      Copy Path
    </ContextMenu.Item>
    <ContextMenu.Item
      disabled={editor.path === null}
      onSelect={() => void copyPath(node, true)}
    >
      <ClipboardIcon />
      Copy Path from Open Note
    </ContextMenu.Item>
    <ContextMenu.Separator />
    <ContextMenu.Item variant="destructive" onSelect={() => startDelete(node)}>
      <Trash2Icon />
      Delete
    </ContextMenu.Item>
  </ContextMenu.Content>
{/snippet}

{#snippet Tree({ node }: { node: FileNode })}
  {#if !node.isDir}
    <Sidebar.MenuItem>
      <ContextMenu.Root>
        <ContextMenu.Trigger>
          {#snippet child({ props })}
            <Sidebar.MenuButton
              {...props}
              class="px-1.5"
              title={node.path}
              data-path={node.path}
              isActive={editor.path === node.path}
              onclick={() => editor.open(node.path)}
            >
              <FileIcon />
              <span>{node.name}</span>
            </Sidebar.MenuButton>
          {/snippet}
        </ContextMenu.Trigger>
        {@render ItemMenu(node)}
      </ContextMenu.Root>
    </Sidebar.MenuItem>
  {:else}
    <Sidebar.MenuItem>
      <Collapsible.Root
        bind:open={
          () => files.expanded.has(node.path),
          (open) => files.setExpanded(node.path, open)
        }
        class="group/collapsible [&[data-state=open]>button>svg:first-child]:rotate-90"
      >
        <ContextMenu.Root>
          <Collapsible.Trigger>
            {#snippet child({ props: collapsibleProps })}
              <ContextMenu.Trigger>
                {#snippet child({ props })}
                  <Sidebar.MenuButton
                    {...mergeProps(collapsibleProps, props)}
                    class="px-1.5"
                    title={node.path}
                    data-path={node.path}
                  >
                    <ChevronRightIcon class="transition-transform" />
                    <FolderIcon />
                    <span>{node.name}</span>
                  </Sidebar.MenuButton>
                {/snippet}
              </ContextMenu.Trigger>
            {/snippet}
          </Collapsible.Trigger>
          {@render ItemMenu(node)}
        </ContextMenu.Root>
        <Collapsible.Content>
          <!-- Compact indent so deep folders leave room for names; the guide line sits under the parent's chevron. -->
          <Sidebar.MenuSub class="mx-0 ms-3 ps-0.5 pe-0">
            {#each node.children ?? [] as child (child.path)}
              {@render Tree({ node: child })}
            {/each}
          </Sidebar.MenuSub>
        </Collapsible.Content>
      </Collapsible.Root>
    </Sidebar.MenuItem>
  {/if}
{/snippet}
