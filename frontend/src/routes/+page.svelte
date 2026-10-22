<script lang="ts">
  import { MarkdownEditor } from "@the-sumeet/sveltedown";

  let value = $state("# Hello\n\nStart **writing**…");
</script>

<MarkdownEditor bind:value class="editor min-h-0 flex-1" placeholder="Start writing…" />

<style>
  /* Fill the editor's height; long documents scroll inside it. */
  :global(.editor .cm-editor) {
    height: 100%;
  }
</style>
