<script lang="ts">
  import { MarkdownEditor, type ImageSnippetProps } from "@the-sumeet/sveltedown";
  import { editor } from "$lib/stores/editor.svelte.js";
  import { files } from "$lib/stores/files.svelte.js";
  import { resolveNoteLink } from "$lib/note-links.js";
  import { pasteImage } from "$lib/paste-image.js";
  import NoteImage from "$lib/components/note-image.svelte";
  import { toast } from "svelte-sonner";

  // A stable reference, as the editor reconfigures whenever it changes.
  const extensions = [pasteImage];

  /** Opens relative links as notes; anything else (http, mailto, …) opens as usual. */
  function onlinkclick(url: string) {
    const path = editor.path === null ? null : resolveNoteLink(editor.path, url);
    if (path === null) return false;
    void openNote(path);
    return true;
  }

  async function openNote(path: string) {
    // The tree may be stale if the note was created outside the app.
    if (!files.find(path)) await files.refresh();
    const node = files.find(path);
    if (!node || node.isDir) {
      toast.error("Note not found", { description: path });
      return;
    }
    await editor.open(path);
  }
</script>

<svelte:window
  onbeforeunload={() => void editor.flush()}
  onblur={() => void editor.flush()}
/>

{#snippet imageSnippet(props: ImageSnippetProps)}
  <NoteImage {...props} />
{/snippet}

{#if editor.path === null}
  <p class="text-muted-foreground m-auto text-sm">Select a file to start editing.</p>
{:else}
  {#key editor.docId}
    <MarkdownEditor
      value={editor.content}
      onchange={editor.edit}
      readonly={editor.loading}
      class="editor min-h-0 flex-1"
      placeholder="Start writing…"
      {onlinkclick}
      {extensions}
      {imageSnippet}
    />
  {/key}
{/if}

<style>
  /* Fill the editor's height; long documents scroll inside it. */
  :global(.editor .cm-editor) {
    height: 100%;
  }
</style>
