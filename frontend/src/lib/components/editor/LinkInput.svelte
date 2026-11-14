<!-- Edits the href of the link at the selection; an empty href removes the link. -->
<script lang="ts">
  import { onMount } from "svelte";
  import type { Editor } from "@tiptap/core";
  import { Check, X } from "@lucide/svelte";
  import { Button } from "$lib/components/ui/button";
  import { normalizeHref } from "$lib/notes/links";

  let {
    editor,
    collapse = false,
    onclose,
  }: {
    editor: Editor;
    /** Put the caret after the link when done, instead of leaving it selected. */
    collapse?: boolean;
    onclose: () => void;
  } = $props();

  let input = $state<HTMLInputElement>();
  let href = $state("");

  onMount(() => {
    href = editor.getAttributes("link").href ?? "";
    queueMicrotask(() => input?.focus());
  });

  function apply() {
    const value = normalizeHref(href.trim());
    const chain = editor.chain().focus().extendMarkRange("link");
    if (value) chain.setLink({ href: value });
    else chain.unsetLink();
    chain.run();
    if (collapse) editor.commands.setTextSelection(editor.state.selection.to);
    onclose();
  }

  function onkeydown(e: KeyboardEvent) {
    if (e.key === "Enter") {
      e.preventDefault();
      apply();
    } else if (e.key === "Escape") {
      e.preventDefault();
      onclose();
      editor.commands.focus();
    }
  }
</script>

<input
  bind:this={input}
  bind:value={href}
  {onkeydown}
  placeholder="Paste a link or note path…"
  class="placeholder:text-muted-foreground h-7 w-56 bg-transparent px-2 text-sm outline-none"
/>
<Button variant="ghost" size="icon-sm" aria-label="Apply link" onclick={apply}><Check /></Button>
<Button variant="ghost" size="icon-sm" aria-label="Cancel" onclick={onclose}><X /></Button>
