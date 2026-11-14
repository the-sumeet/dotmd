<!--
  Shown while the caret is inside a link: the link's target with Open / Edit /
  Remove. Opening is also Cmd/Ctrl+click, but this works on touch screens too.
-->
<script lang="ts">
  import type { Editor } from "@tiptap/core";
  import { ExternalLink, FileText, Pencil, Unlink } from "@lucide/svelte";
  import { Button } from "$lib/components/ui/button";
  import { isExternalLink, normalizeHref } from "$lib/notes/links";
  import LinkInput from "./LinkInput.svelte";

  let {
    editor,
    revision,
    element = $bindable(),
    onopen,
  }: {
    editor: Editor | null;
    revision: number;
    element?: HTMLDivElement;
    /** Follow a link, given its raw href. */
    onopen?: (href: string) => void;
  } = $props();

  let editing = $state(false);

  const href = $derived.by(() => {
    revision;
    return (editor?.getAttributes("link").href as string | undefined) ?? "";
  });

  // Leave editing once the caret moves out of the link (menu hides).
  $effect(() => {
    revision;
    if (editor && !editor.isActive("link")) editing = false;
  });
</script>

<!-- Starts hidden, like BubbleMenu; the plugin shows and positions it. -->
<div
  bind:this={element}
  style="visibility: hidden; position: absolute;"
  class="bg-popover text-popover-foreground flex max-w-[calc(100vw-2rem)] items-center gap-0.5 rounded-lg border p-1 shadow-md"
>
  {#if editor}
    {#if editing}
      <LinkInput {editor} collapse onclose={() => (editing = false)} />
    {:else}
      <span class="text-muted-foreground flex min-w-0 items-center gap-1.5 px-2 text-sm" title={href}>
        {#if isExternalLink(normalizeHref(href))}
          <ExternalLink class="size-3.5 shrink-0" />
        {:else}
          <FileText class="size-3.5 shrink-0" />
        {/if}
        <span class="max-w-56 truncate">{href}</span>
      </span>
      <Button variant="ghost" size="sm" class="h-7 px-2" disabled={!href} onclick={() => onopen?.(href)}>
        Open
      </Button>
      <div class="bg-border mx-0.5 h-5 w-px"></div>
      <Button variant="ghost" size="icon-sm" class="text-muted-foreground" aria-label="Edit link"
        onclick={() => (editing = true)}><Pencil /></Button>
      <Button variant="ghost" size="icon-sm" class="text-muted-foreground" aria-label="Remove link"
        onclick={() => editor.chain().focus().unsetLink().run()}><Unlink /></Button>
    {/if}
  {/if}
</div>
