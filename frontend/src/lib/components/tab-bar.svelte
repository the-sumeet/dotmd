<script lang="ts">
  import XIcon from "@lucide/svelte/icons/x";
  import * as Tabs from "$lib/components/ui/tabs/index.js";
  import { editor } from "$lib/stores/editor.svelte.js";
  import { tabs } from "$lib/stores/tabs.svelte.js";

  let list = $state<HTMLElement | null>(null);

  // Keep the active tab visible when the strip overflows.
  $effect(() => {
    const path = editor.path;
    // A new tab is added after the editor switches, so re-run once it's in the DOM.
    void tabs.paths.length;
    if (path === null || list === null) return;
    const tab = list.querySelector<HTMLElement>(
      `[data-tab="${CSS.escape(path)}"]`,
    );
    if (!tab) return;
    // Scroll the strip itself; scrollIntoView would also shift the page's clipped ancestors.
    const t = tab.getBoundingClientRect();
    const l = list.getBoundingClientRect();
    if (t.left < l.left) list.scrollLeft -= l.left - t.left;
    else if (t.right > l.right) list.scrollLeft += t.right - l.right;
  });

  const name = (path: string) => path.split("/").pop() ?? path;
</script>

<Tabs.Root
  bind:value={() => editor.path ?? "", (path) => void tabs.open(path)}
  class="min-w-0 flex-1"
>
  <Tabs.List
    bind:ref={list}
    variant="line"
    class="w-full justify-start gap-0 overflow-x-auto p-0 [scrollbar-width:none] [&::-webkit-scrollbar]:hidden"
  >
    {#each tabs.paths as path (path)}
      <!-- The close button is a sibling of the trigger: a button can't nest inside a button. -->
      <div
        class="group relative flex h-full shrink-0 items-center"
        data-tab={path}
        title={path}
        onauxclick={(e) => {
          if (e.button !== 1) return;
          e.preventDefault();
          void tabs.close(path);
        }}
        role="presentation"
      >
        <Tabs.Trigger value={path} class="max-w-48 flex-none rounded-none pe-7">
          <span class="truncate">{name(path)}</span>
          {#if editor.path === path && editor.dirty}<span>•</span>{/if}
        </Tabs.Trigger>
        <button
          type="button"
          aria-label="Close {name(path)}"
          class="text-muted-foreground hover:bg-muted hover:text-foreground absolute inset-e-1 top-1/2 -translate-y-1/2 rounded-sm p-0.5 opacity-0 group-focus-within:opacity-100 group-hover:opacity-100 {editor.path ===
          path
            ? 'opacity-100'
            : ''}"
          onclick={() => void tabs.close(path)}
        >
          <XIcon class="size-3.5" />
        </button>
      </div>
    {/each}
  </Tabs.List>
</Tabs.Root>
