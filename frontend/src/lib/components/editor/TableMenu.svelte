<script lang="ts">
  import type { Editor } from "@tiptap/core";
  import {
    BetweenHorizontalEnd,
    BetweenHorizontalStart,
    BetweenVerticalEnd,
    BetweenVerticalStart,
    Grid2x2X,
    Minus,
  } from "@lucide/svelte";
  import { selectedRect } from "@tiptap/pm/tables";
  import { Button } from "$lib/components/ui/button";

  let {
    editor,
    revision,
    element = $bindable(),
  }: { editor: Editor | null; revision: number; element?: HTMLDivElement } = $props();

  const none = {
    addRowBefore: false,
    addRowAfter: false,
    addColumnBefore: false,
    addColumnAfter: false,
    deleteRow: false,
    deleteColumn: false,
    deleteTable: false,
  };

  // `revision` changes on every transaction, so availability stays in sync.
  // Outside a table the menu is hidden, so skip the dry runs entirely.
  const can = $derived.by(() => {
    revision;
    if (!editor?.isActive("table")) return none;
    const c = editor.can();
    // GFM tables always start with a header row: a row inserted above it, or
    // deleting it, would turn the header into body text when saved.
    const inHeader = selectedRect(editor.state).top === 0;
    return {
      addRowBefore: !inHeader && c.addRowBefore(),
      addRowAfter: c.addRowAfter(),
      addColumnBefore: c.addColumnBefore(),
      addColumnAfter: c.addColumnAfter(),
      deleteRow: !inHeader && c.deleteRow(),
      deleteColumn: c.deleteColumn(),
      deleteTable: c.deleteTable(),
    };
  });

  type Command = keyof typeof can;
  const run = (cmd: Command) => editor?.chain().focus()[cmd]().run();
</script>

<!-- Starts hidden: the bubble menu plugin only hides elements it has shown, and
     it overrides these inline styles when it positions the menu. -->
<div
  bind:this={element}
  style="visibility: hidden; position: absolute;"
  class="bg-popover text-popover-foreground flex items-center gap-0.5 rounded-lg border p-1 shadow-md"
>
  {#if editor}
    <Button variant="ghost" size="icon-sm" title="Add row above" aria-label="Add row above"
      disabled={!can.addRowBefore} onclick={() => run("addRowBefore")}><BetweenHorizontalStart /></Button>
    <Button variant="ghost" size="icon-sm" title="Add row below" aria-label="Add row below"
      disabled={!can.addRowAfter} onclick={() => run("addRowAfter")}><BetweenHorizontalEnd /></Button>
    <Button variant="ghost" size="icon-sm" title="Add column left" aria-label="Add column left"
      disabled={!can.addColumnBefore} onclick={() => run("addColumnBefore")}><BetweenVerticalStart /></Button>
    <Button variant="ghost" size="icon-sm" title="Add column right" aria-label="Add column right"
      disabled={!can.addColumnAfter} onclick={() => run("addColumnAfter")}><BetweenVerticalEnd /></Button>
    <div class="bg-border mx-0.5 h-5 w-px"></div>
    <Button variant="ghost" size="sm" class="h-7 gap-1 px-2" title="Delete row"
      disabled={!can.deleteRow} onclick={() => run("deleteRow")}><Minus />Row</Button>
    <Button variant="ghost" size="sm" class="h-7 gap-1 px-2" title="Delete column"
      disabled={!can.deleteColumn} onclick={() => run("deleteColumn")}><Minus />Column</Button>
    <div class="bg-border mx-0.5 h-5 w-px"></div>
    <Button variant="ghost" size="icon-sm" class="hover:text-destructive" title="Delete table"
      aria-label="Delete table" disabled={!can.deleteTable} onclick={() => run("deleteTable")}><Grid2x2X /></Button>
  {/if}
</div>
