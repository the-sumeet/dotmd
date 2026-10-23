<script lang="ts">
  import type { ImageSnippetProps } from "@the-sumeet/sveltedown";
  import { resolveAssetUrl } from "$lib/note-links.js";
  import { editor } from "$lib/stores/editor.svelte.js";

  let { src, alt, title, view }: ImageSnippetProps = $props();

  // Relative images are files next to the note; remote ones load as they are.
  const url = $derived((editor.path === null ? null : resolveAssetUrl(editor.path, src)) ?? src);
  // Remembering which url failed resets the error state when it changes.
  let failedUrl: string | null = $state(null);
</script>

{#if failedUrl === url}
  <span class="cm-md-image-error" title={src}>Image failed to load: {alt || src}</span>
{:else}
  <img
    src={url}
    {alt}
    {title}
    onload={() => view.requestMeasure()}
    onerror={() => {
      failedUrl = url;
      view.requestMeasure();
    }}
  />
{/if}
