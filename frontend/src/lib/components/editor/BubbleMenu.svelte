<script lang="ts">
  import type { Editor } from "@tiptap/core";
  import { Bold, Check, Code, Heading1, Heading2, Italic, Link, Strikethrough, Unlink, X } from "@lucide/svelte";
  import { Button } from "$lib/components/ui/button";

  let {
    editor,
    revision,
    element = $bindable(),
  }: { editor: Editor | null; revision: number; element?: HTMLDivElement } = $props();

  let linkMode = $state(false);
  let linkUrl = $state("");
  let linkInput = $state<HTMLInputElement>();

  // `revision` changes on every transaction, so active states stay in sync.
  const active = $derived.by(() => {
    revision;
    const is = (name: string, attrs?: Record<string, unknown>) => editor?.isActive(name, attrs) ?? false;
    return {
      bold: is("bold"),
      italic: is("italic"),
      strike: is("strike"),
      code: is("code"),
      link: is("link"),
      h1: is("heading", { level: 1 }),
      h2: is("heading", { level: 2 }),
    };
  });

  // Leave link editing when the selection collapses (menu hides).
  $effect(() => {
    revision;
    if (editor && editor.state.selection.empty) linkMode = false;
  });

  function openLink() {
    linkUrl = editor?.getAttributes("link").href ?? "";
    linkMode = true;
    queueMicrotask(() => linkInput?.focus());
  }

  function applyLink() {
    if (!editor) return;
    const href = linkUrl.trim();
    const chain = editor.chain().focus().extendMarkRange("link");
    if (href) chain.setLink({ href }).run();
    else chain.unsetLink().run();
    linkMode = false;
  }

  function onLinkKeydown(e: KeyboardEvent) {
    if (e.key === "Enter") {
      e.preventDefault();
      applyLink();
    } else if (e.key === "Escape") {
      e.preventDefault();
      linkMode = false;
      editor?.commands.focus();
    }
  }

  const btn = (on: boolean) => (on ? "bg-muted text-foreground" : "text-muted-foreground");
</script>

<div
  bind:this={element}
  class="bg-popover text-popover-foreground flex items-center gap-0.5 rounded-lg border p-1 shadow-md"
>
  {#if editor}
    {#if linkMode}
      <input
        bind:this={linkInput}
        bind:value={linkUrl}
        onkeydown={onLinkKeydown}
        placeholder="Paste a link…"
        class="placeholder:text-muted-foreground h-7 w-56 bg-transparent px-2 text-sm outline-none"
      />
      <Button variant="ghost" size="icon-sm" aria-label="Apply link" onclick={applyLink}><Check /></Button>
      <Button variant="ghost" size="icon-sm" aria-label="Cancel" onclick={() => (linkMode = false)}><X /></Button>
    {:else}
      <Button variant="ghost" size="icon-sm" class={btn(active.h1)} aria-label="Heading 1"
        onclick={() => editor.chain().focus().toggleHeading({ level: 1 }).run()}><Heading1 /></Button>
      <Button variant="ghost" size="icon-sm" class={btn(active.h2)} aria-label="Heading 2"
        onclick={() => editor.chain().focus().toggleHeading({ level: 2 }).run()}><Heading2 /></Button>
      <div class="bg-border mx-0.5 h-5 w-px"></div>
      <Button variant="ghost" size="icon-sm" class={btn(active.bold)} aria-label="Bold"
        onclick={() => editor.chain().focus().toggleBold().run()}><Bold /></Button>
      <Button variant="ghost" size="icon-sm" class={btn(active.italic)} aria-label="Italic"
        onclick={() => editor.chain().focus().toggleItalic().run()}><Italic /></Button>
      <Button variant="ghost" size="icon-sm" class={btn(active.strike)} aria-label="Strikethrough"
        onclick={() => editor.chain().focus().toggleStrike().run()}><Strikethrough /></Button>
      <Button variant="ghost" size="icon-sm" class={btn(active.code)} aria-label="Inline code"
        onclick={() => editor.chain().focus().toggleCode().run()}><Code /></Button>
      <div class="bg-border mx-0.5 h-5 w-px"></div>
      <Button variant="ghost" size="icon-sm" class={btn(active.link)} aria-label="Link" onclick={openLink}><Link /></Button>
      {#if active.link}
        <Button variant="ghost" size="icon-sm" class="text-muted-foreground" aria-label="Remove link"
          onclick={() => editor.chain().focus().extendMarkRange("link").unsetLink().run()}><Unlink /></Button>
      {/if}
    {/if}
  {/if}
</div>
