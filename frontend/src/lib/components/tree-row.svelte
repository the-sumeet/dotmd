<!--
  A Notion-style sidebar row. Nesting is shown by left padding only, so every
  row spans the full sidebar width and hover actions line up at the right edge.
  Expandable rows swap their icon for a chevron on hover.
-->
<script lang="ts">
  import ChevronRightIcon from "@lucide/svelte/icons/chevron-right";
  import { cn } from "$lib/utils.js";
  import type { Snippet } from "svelte";

  let {
    depth = 0,
    label,
    title,
    active = false,
    expandable = false,
    expanded = false,
    trailing,
    ontoggle,
    onselect,
    icon,
    actions,
  }: {
    depth?: number;
    label: string;
    /** Native tooltip, e.g. the full path. */
    title?: string;
    active?: boolean;
    expandable?: boolean;
    expanded?: boolean;
    /** Muted text at the end of the row (e.g. a count). */
    trailing?: string;
    ontoggle?: () => void;
    /** Clicking the row (or Enter on it); also a middle-click, with button 1. */
    onselect: (e: MouseEvent) => void;
    icon: Snippet;
    /** Buttons shown at the right edge while the row is hovered or focused. */
    actions?: Snippet;
  } = $props();

  /** Indent per nesting level, in px. */
  const STEP = 16;
</script>

<div
  role="presentation"
  class={cn(
    "group/row flex h-7 cursor-pointer items-center gap-1.5 rounded-md pe-1 text-sm font-medium select-none",
    "text-sidebar-foreground/70 hover:bg-sidebar-accent hover:text-sidebar-accent-foreground",
    "has-[:focus-visible]:ring-2 has-[:focus-visible]:ring-sidebar-ring",
    active && "bg-sidebar-accent text-sidebar-accent-foreground",
  )}
  style:padding-inline-start="{4 + depth * STEP}px"
  onclick={onselect}
  onauxclick={(e) => {
    if (e.button === 1) onselect(e);
  }}
  onmousedown={(e) => {
    if (e.button === 1) e.preventDefault(); // no autoscroll on middle-click
  }}
>
  <span
    class="relative grid size-5 shrink-0 place-items-center text-sidebar-foreground/50 [&_svg]:size-4"
  >
    {#if expandable}
      <span class="grid place-items-center group-hover/row:invisible">
        {@render icon()}
      </span>
      <button
        type="button"
        tabindex="-1"
        aria-label={expanded ? "Collapse" : "Expand"}
        class="absolute inset-0 hidden place-items-center rounded text-sidebar-foreground/70 group-hover/row:grid hover:bg-sidebar-foreground/10"
        onclick={(e) => {
          e.stopPropagation();
          ontoggle?.();
        }}
      >
        <ChevronRightIcon class="transition-transform {expanded ? 'rotate-90' : ''}" />
      </button>
    {:else}
      {@render icon()}
    {/if}
  </span>

  <button
    type="button"
    {title}
    aria-expanded={expandable ? expanded : undefined}
    aria-current={active ? "page" : undefined}
    class="min-w-0 flex-1 truncate text-start outline-none"
  >
    {label}
  </button>

  {#if trailing}
    <span class="shrink-0 px-1 text-xs text-sidebar-foreground/40 tabular-nums">{trailing}</span>
  {/if}

  {#if actions}
    <!-- Stop clicks here from also selecting the row. -->
    <div
      role="presentation"
      class="flex shrink-0 items-center opacity-0 group-focus-within/row:opacity-100 group-hover/row:opacity-100 has-data-[state=open]:opacity-100"
      onclick={(e) => e.stopPropagation()}
      onauxclick={(e) => e.stopPropagation()}
    >
      {@render actions()}
    </div>
  {/if}
</div>
