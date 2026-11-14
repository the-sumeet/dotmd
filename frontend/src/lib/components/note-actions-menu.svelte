<!-- "⋯" menu on a note row. "Copy relative path" copies the href that links
     the open note to this one, ready to paste into the link input. -->
<script lang="ts">
  import { onDestroy } from "svelte";
  import EllipsisIcon from "@lucide/svelte/icons/ellipsis";
  import CheckIcon from "@lucide/svelte/icons/check";
  import LinkIcon from "@lucide/svelte/icons/link";
  import { Clipboard } from "@wailsio/runtime";
  import * as DropdownMenu from "$lib/components/ui/dropdown-menu/index.js";
  import { linkPath, noteTitle } from "$lib/notes/notes";

  let {
    path,
    activePath,
  }: {
    /** The note this row stands for. */
    path: string;
    /** The open note, which links are relative to. */
    activePath: string | null;
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
    <DropdownMenu.Item disabled={!activePath} onSelect={copy}>
      {#if copied}
        <CheckIcon /> Copied
      {:else if failed}
        <LinkIcon /> Couldn't copy
      {:else}
        <LinkIcon /> Copy relative path
      {/if}
    </DropdownMenu.Item>
  </DropdownMenu.Content>
</DropdownMenu.Root>
