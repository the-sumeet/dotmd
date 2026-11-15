<!-- "⋯" menu on a Files tree row. "Copy relative path" copies the href
     that links the open note to this file, ready to paste into the link input.
     "Rename…" and "Delete…" show when the row passes handlers for them. -->
<script lang="ts">
  import { onDestroy } from "svelte";
  import EllipsisIcon from "@lucide/svelte/icons/ellipsis";
  import CheckIcon from "@lucide/svelte/icons/check";
  import LinkIcon from "@lucide/svelte/icons/link";
  import PencilIcon from "@lucide/svelte/icons/pencil";
  import Trash2Icon from "@lucide/svelte/icons/trash-2";
  import { Clipboard } from "@wailsio/runtime";
  import * as DropdownMenu from "$lib/components/ui/dropdown-menu/index.js";
  import { linkPath, noteTitle } from "$lib/notes/notes";

  let {
    path,
    activePath,
    copy: copyable = true,
    onrename,
    ondelete,
  }: {
    /** The file this row stands for. */
    path: string;
    /** The open note, which links are relative to. */
    activePath: string | null;
    /** Offer "Copy relative path" (folders can't be linked to). */
    copy?: boolean;
    /** Ask to rename the file (e.g. open a dialog); omitted, no item shows. */
    onrename?: () => void;
    /** Ask to delete the file (e.g. open a confirm dialog); omitted, no item shows. */
    ondelete?: () => void;
  } = $props();

  /** How long "Copied" shows before the menu closes. */
  const COPIED_MS = 900;

  let open = $state(false);
  let copied = $state(false);
  let failed = $state(false);
  let closeTimer: ReturnType<typeof setTimeout> | undefined;

  async function writeClipboard(text: string) {
    try {
      await Clipboard.SetText(text);
    } catch {
      await navigator.clipboard.writeText(text);
    }
  }

  async function copy(e: Event) {
    e.preventDefault(); // keep the menu open to show "Copied"
    if (!activePath) return;
    try {
      await writeClipboard(await linkPath(activePath, path));
      copied = true;
    } catch (err) {
      console.error("Failed to copy link path", err);
      failed = true;
    }
    if (open) closeTimer = setTimeout(() => (open = false), COPIED_MS);
  }

  // A menu closed early (Escape) must not be closed again when reopened.
  $effect(() => {
    if (open) return;
    clearTimeout(closeTimer);
    copied = failed = false;
  });

  onDestroy(() => clearTimeout(closeTimer));
</script>

<DropdownMenu.Root bind:open>
  <DropdownMenu.Trigger>
    {#snippet child({ props })}
      <button
        {...props}
        type="button"
        title="More for {noteTitle(path)}"
        aria-label="More for {noteTitle(path)}"
        class="grid size-5 place-items-center rounded text-sidebar-foreground/60 outline-hidden hover:bg-sidebar-foreground/10 hover:text-sidebar-foreground focus-visible:ring-2 focus-visible:ring-sidebar-ring [&_svg]:size-4"
      >
        <EllipsisIcon />
      </button>
    {/snippet}
  </DropdownMenu.Trigger>
  <DropdownMenu.Content align="start" class="w-56">
    {#if copyable}
      <DropdownMenu.Item disabled={!activePath} onSelect={copy}>
        {#if copied}
          <CheckIcon /> Copied
        {:else if failed}
          <LinkIcon /> Couldn't copy
        {:else}
          <LinkIcon /> Copy relative path
        {/if}
      </DropdownMenu.Item>
    {/if}
    {#if onrename}
      <DropdownMenu.Item onSelect={onrename}>
        <PencilIcon /> Rename…
      </DropdownMenu.Item>
    {/if}
    {#if ondelete}
      {#if copyable || onrename}<DropdownMenu.Separator />{/if}
      <DropdownMenu.Item variant="destructive" onSelect={ondelete}>
        <Trash2Icon /> Delete…
      </DropdownMenu.Item>
    {/if}
  </DropdownMenu.Content>
</DropdownMenu.Root>
