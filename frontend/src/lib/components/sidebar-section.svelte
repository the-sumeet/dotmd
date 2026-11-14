<!--
  A Notion-style collapsible sidebar section: a small muted heading that toggles
  its content when clicked, with optional actions shown on hover.
-->
<script lang="ts">
  import ChevronRightIcon from "@lucide/svelte/icons/chevron-right";
  import * as Collapsible from "$lib/components/ui/collapsible/index.js";
  import type { Snippet } from "svelte";

  let {
    title,
    open = $bindable(true),
    actions,
    children,
  }: {
    title: string;
    open?: boolean;
    /** Controls at the right of the heading, shown on hover. */
    actions?: Snippet;
    children: Snippet;
  } = $props();
</script>

<Collapsible.Root bind:open class="group/section px-2 pt-3">
  <div
    class="group/header flex h-7 items-center rounded-md pe-1 hover:bg-sidebar-accent has-[:focus-visible]:ring-2 has-[:focus-visible]:ring-sidebar-ring"
  >
    <Collapsible.Trigger
      class="flex h-full min-w-0 flex-1 items-center gap-1 ps-2 text-xs font-medium text-sidebar-foreground/50 outline-hidden"
    >
      <span class="truncate">{title}</span>
      <ChevronRightIcon
        class="size-3 shrink-0 opacity-0 transition group-hover/header:opacity-100 group-data-[state=open]/section:rotate-90"
      />
    </Collapsible.Trigger>
    {#if actions}
      <div
        class="flex items-center opacity-0 group-focus-within/header:opacity-100 group-hover/header:opacity-100 has-data-[state=open]:opacity-100"
      >
        {@render actions()}
      </div>
    {/if}
  </div>
  <Collapsible.Content class="flex flex-col gap-px pt-0.5">
    {@render children()}
  </Collapsible.Content>
</Collapsible.Root>
