<script lang="ts">
  import { cn } from "$lib/utils";
  import type { SlashItem } from "$lib/editor/slash-items";

  let { items, command }: { items: SlashItem[]; command: (item: SlashItem) => void } = $props();

  let selected = $state(0);
  let listEl: HTMLDivElement;

  $effect(() => {
    items;
    selected = 0;
  });

  $effect(() => {
    listEl?.querySelector(`[data-index="${selected}"]`)?.scrollIntoView({ block: "nearest" });
  });

  /** Called by the suggestion plugin; returns true when the key was handled. */
  export function onKeyDown(event: KeyboardEvent): boolean {
    if (!items.length) return false;
    if (event.key === "ArrowDown") {
      selected = (selected + 1) % items.length;
      return true;
    }
    if (event.key === "ArrowUp") {
      selected = (selected - 1 + items.length) % items.length;
      return true;
    }
    if (event.key === "Enter" || event.key === "Tab") {
      command(items[selected]);
      return true;
    }
    return false;
  }
</script>

<div
  bind:this={listEl}
  class="bg-popover text-popover-foreground max-h-80 w-64 overflow-y-auto rounded-lg border p-1 shadow-md"
  role="listbox"
>
  {#each items as item, i (item.title)}
    <button
      type="button"
      role="option"
      aria-selected={i === selected}
      data-index={i}
      class={cn(
        "flex w-full items-center gap-3 rounded-md px-2 py-1.5 text-left text-sm",
        i === selected ? "bg-accent text-accent-foreground" : "hover:bg-accent/60",
      )}
      onmouseenter={() => (selected = i)}
      onmousedown={(e) => e.preventDefault()}
      onclick={() => command(item)}
    >
      <span class="bg-background flex size-8 shrink-0 items-center justify-center rounded-md border">
        <item.icon class="size-4" />
      </span>
      <span class="min-w-0">
        <span class="block font-medium">{item.title}</span>
        <span class="text-muted-foreground block truncate text-xs">{item.description}</span>
      </span>
    </button>
  {:else}
    <div class="text-muted-foreground px-2 py-1.5 text-sm">No matching blocks</div>
  {/each}
</div>
