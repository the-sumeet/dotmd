<script lang="ts">
  import { onDestroy, onMount } from "svelte";
  import { Editor } from "@tiptap/core";
  import { EditorState } from "@tiptap/pm/state";
  import { createExtensions } from "$lib/editor/extensions";
  import { joinFrontmatter, splitFrontmatter } from "$lib/editor/frontmatter";
  import BubbleMenu from "./BubbleMenu.svelte";
  import "./editor.css";

  /** Full markdown of the note, including any YAML frontmatter. */
  let { value = $bindable("") }: { value?: string } = $props();

  /** Serializing the whole doc is O(n), so do it after typing pauses, not per keystroke. */
  const SYNC_DELAY_MS = 300;

  let element: HTMLDivElement;
  let bubbleElement = $state<HTMLDivElement>();
  let editor = $state.raw<Editor | null>(null);
  let revision = $state(0);
  let frontmatter = "";
  let syncTimer: ReturnType<typeof setTimeout> | undefined;

  onMount(() => {
    const { frontmatter: fm, body } = splitFrontmatter(value);
    frontmatter = fm;
    editor = new Editor({
      element,
      extensions: createExtensions(bubbleElement!),
      content: body,
      contentType: "markdown",
      editorProps: {
        attributes: {
          class: "prose prose-neutral dark:prose-invert max-w-none min-h-[60vh] focus:outline-none",
          spellcheck: "true",
        },
      },
      onUpdate: () => {
        clearTimeout(syncTimer);
        syncTimer = setTimeout(flush, SYNC_DELAY_MS);
      },
      onTransaction: () => {
        revision++;
      },
    });
  });

  onDestroy(() => {
    flush();
    editor?.destroy();
  });

  /** Write any pending edits to `value` immediately (before showing source, saving, …). */
  export function flush() {
    clearTimeout(syncTimer);
    syncTimer = undefined;
    if (editor && !editor.isDestroyed) value = joinFrontmatter(frontmatter, editor.getMarkdown());
  }

  /**
   * Replace the document with new markdown (e.g. after editing the raw source).
   * This acts like reloading the file: undo history is reset, because the
   * frontmatter lives outside the Tiptap document and undo could not restore it.
   */
  export function setMarkdown(markdown: string) {
    if (!editor) return;
    clearTimeout(syncTimer);
    const { frontmatter: fm, body } = splitFrontmatter(markdown);
    frontmatter = fm;
    editor.commands.setContent(body, { contentType: "markdown", emitUpdate: false });
    const { doc, plugins, selection } = editor.state;
    editor.view.updateState(EditorState.create({ doc, plugins, selection }));
  }

  export function focus() {
    editor?.commands.focus();
  }
</script>

<div bind:this={element} class="dotmd-editor"></div>
<BubbleMenu {editor} {revision} bind:element={bubbleElement} />
