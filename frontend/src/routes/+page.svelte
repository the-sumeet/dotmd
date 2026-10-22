<script lang="ts">
  import { MarkdownEditor } from "@the-sumeet/sveltedown";
  import { editor } from "$lib/stores/editor.svelte.js";
</script>

<svelte:window
  onbeforeunload={() => void editor.flush()}
  onblur={() => void editor.flush()}
/>

{#if editor.path === null}
  <p class="text-muted-foreground m-auto text-sm">Select a file to start editing.</p>
{:else}
  {#key editor.path}
    <MarkdownEditor
      value={editor.content}
      onchange={editor.edit}
      readonly={editor.loading}
      class="editor min-h-0 flex-1"
      placeholder="Start writing…"
    />
  {/key}
{/if}

<style>
  /* Fill the editor's height; long documents scroll inside it. */
  :global(.editor .cm-editor) {
    height: 100%;
  }
</style>
