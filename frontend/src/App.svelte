<script lang="ts">
  import { FileText, PenLine } from "@lucide/svelte";
  import { Button } from "$lib/components/ui/button";
  import Editor from "$lib/components/editor/Editor.svelte";
  import { splitFrontmatter } from "$lib/editor/frontmatter";
  import sample from "$lib/editor/sample.md?raw";

  type Mode = "edit" | "markdown";

  /** The note as plain markdown — the single source of truth. */
  let markdown = $state(sample);
  let mode = $state<Mode>("edit");
  let sourceEdited = false;
  let editor: Editor;
  let source = $state<HTMLTextAreaElement>();

  function setMode(next: Mode) {
    if (next === mode) return;
    if (next === "markdown") {
      editor.flush();
    } else {
      if (sourceEdited) editor.setMarkdown(markdown);
      sourceEdited = false;
      queueMicrotask(() => editor.focus());
    }
    mode = next;
  }

  // Count words in the body only, ignoring tokens with no letters or digits
  // (markdown syntax such as `#`, `-`, `>`, `|`, `---`, `[x]`).
  const words = $derived(
    splitFrontmatter(markdown).body.split(/\s+/).filter((t) => /[\p{L}\p{N}]/u.test(t)).length,
  );

  // Grow the source textarea with its content so the page, not the textarea, scrolls.
  $effect(() => {
    markdown;
    if (!source) return;
    source.style.height = "auto";
    source.style.height = `${source.scrollHeight}px`;
  });
</script>

<div class="flex h-screen flex-col">
  <header
    class="bg-background/80 sticky top-0 z-10 flex h-12 shrink-0 items-center justify-between gap-4 border-b pr-3 pl-20 backdrop-blur"
    style="--wails-draggable: drag"
  >
    <div class="flex min-w-0 items-center gap-2 text-sm">
      <FileText class="text-muted-foreground size-4 shrink-0" />
      <span class="truncate font-medium">Untitled.md</span>
      <span class="text-muted-foreground text-xs">{words} words</span>
    </div>

    <div class="bg-muted flex items-center gap-0.5 rounded-lg p-0.5" style="--wails-draggable: no-drag">
      <Button
        size="xs"
        variant={mode === "edit" ? "outline" : "ghost"}
        aria-pressed={mode === "edit"}
        onclick={() => setMode("edit")}
      >
        <PenLine data-icon="inline-start" /> Edit
      </Button>
      <Button
        size="xs"
        variant={mode === "markdown" ? "outline" : "ghost"}
        aria-pressed={mode === "markdown"}
        onclick={() => setMode("markdown")}
      >
        <span class="font-mono text-[0.7rem]" data-icon="inline-start">MD</span> Markdown
      </Button>
    </div>
  </header>

  <main class="flex-1 overflow-y-auto">
    <div class="mx-auto w-full max-w-3xl px-6 py-10 sm:px-10">
      <!-- Kept mounted while hidden so editor state (history, selection) survives toggling. -->
      <div class:hidden={mode !== "edit"}>
        <Editor bind:this={editor} bind:value={markdown} />
      </div>

      {#if mode === "markdown"}
        <textarea
          bind:this={source}
          bind:value={markdown}
          oninput={() => (sourceEdited = true)}
          spellcheck="false"
          aria-label="Markdown source"
          class="text-foreground block min-h-[60vh] w-full resize-none overflow-hidden bg-transparent font-mono text-sm leading-relaxed outline-none"
        ></textarea>
      {/if}
    </div>
  </main>
</div>
