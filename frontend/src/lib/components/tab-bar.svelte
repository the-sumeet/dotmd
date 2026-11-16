<!--
  The open notes as tabs, in the window's top row. Click a tab to show it,
  middle-click or × to close it, drag to reorder, right-click for more. The
  active tab is kept scrolled into view when the tabs overflow. Empty space
  in the row drags the window.
-->
<script lang="ts">
  import { tick } from "svelte";
  import FileTextIcon from "@lucide/svelte/icons/file-text";
  import PlusIcon from "@lucide/svelte/icons/plus";
  import XIcon from "@lucide/svelte/icons/x";
  import { Button } from "$lib/components/ui/button/index.js";
  import * as ContextMenu from "$lib/components/ui/context-menu/index.js";
  import * as Tabs from "$lib/components/ui/tabs/index.js";
  import { modKey } from "$lib/platform";
  import { tabLabels } from "$lib/notes/tabs";
  import { cn } from "$lib/utils.js";

  let {
    tabs,
    active,
    class: className,
    onactivate,
    onclose,
    oncloseothers,
    oncloseright,
    onreorder,
    onnew,
  }: {
    /** Root-relative paths of the open notes, in tab order. */
    tabs: string[];
    /** The note shown, or null. */
    active: string | null;
    class?: string;
    onactivate: (path: string) => void;
    onclose: (path: string) => void;
    oncloseothers: (path: string) => void;
    oncloseright: (path: string) => void;
    /** Move the tab at index `from` to index `to`. */
    onreorder: (from: number, to: number) => void;
    /** Open a note in a new tab (the "+" button). */
    onnew: () => void;
  } = $props();

  const labels = $derived(tabLabels(tabs));

  let strip = $state<HTMLElement | null>(null);
  /** Index of the tab being dragged, and of the tab it would land on. */
  let dragFrom = $state<number | null>(null);
  let dropAt = $state<number | null>(null);

  // Keep the active tab in view: a new tab, a switch from the keyboard, a
  // tab restored at launch.
  $effect(() => {
    const path = active;
    tabs.length; // also after tabs are added or closed around it
    if (!strip || !path) return;
    void tick().then(() => {
      const el = strip?.querySelector<HTMLElement>(`[data-path="${CSS.escape(path)}"]`);
      el?.scrollIntoView({ block: "nearest", inline: "nearest" });
    });
  });

  // A mouse wheel scrolls overflowing tabs sideways. Not passive, so the
  // page doesn't also get the wheel.
  $effect(() => {
    const el = strip;
    if (!el) return;
    const onWheel = (e: WheelEvent) => {
      if (Math.abs(e.deltaY) <= Math.abs(e.deltaX) || el.scrollWidth <= el.clientWidth) return;
      el.scrollLeft += e.deltaY;
      e.preventDefault();
    };
    el.addEventListener("wheel", onWheel, { passive: false });
    return () => el.removeEventListener("wheel", onWheel);
  });

  function endDrag() {
    dragFrom = dropAt = null;
  }

  /** Line showing where a dragged tab would land, on the side it moves to. */
  function dropLine(i: number) {
    if (dragFrom === null || dropAt !== i || dragFrom === i) return undefined;
    return cn(
      "before:absolute before:inset-y-1 before:w-0.5 before:rounded before:bg-primary",
      dragFrom < i ? "before:-inset-e-0.5" : "before:-inset-s-0.5",
    );
  }
</script>

<div
  class={cn("flex h-10 shrink-0 items-center gap-1 bg-background px-2", className)}
  style="--wails-draggable: drag"
>
  <!--
    The active tab follows `active`; a click only asks for a switch, which
    App can refuse (say, the open note couldn't be saved). "manual", so
    arrow keys move focus without opening every note on the way.
  -->
  <Tabs.Root
    bind:value={() => active ?? "", (path) => path && onactivate(path)}
    activationMode="manual"
    class="min-w-0 gap-0"
  >
    <Tabs.List
      bind:ref={strip}
      aria-label="Open notes"
      class="h-auto w-auto max-w-full justify-start gap-0.5 overflow-x-auto rounded-none bg-transparent p-0 scrollbar-none [&::-webkit-scrollbar]:hidden"
      style="--wails-draggable: no-drag"
    >
      {#each tabs as path, i (path)}
        {@const selected = path === active}
        <ContextMenu.Root>
          <ContextMenu.Trigger>
            {#snippet child({ props: menuProps })}
              <Tabs.Trigger
                {...menuProps}
                value={path}
                data-path={path}
                title={path}
                draggable="true"
                class={cn(
                  "group/tab h-7 max-w-52 min-w-24 flex-none cursor-default justify-start ps-2 pe-1 select-none",
                  "hover:bg-muted/70 aria-selected:bg-muted aria-selected:text-foreground",
                  dragFrom === i && "opacity-50",
                  dropLine(i),
                )}
                onmousedown={(e: MouseEvent) => {
                  if (e.button === 1) e.preventDefault(); // no autoscroll
                }}
                onauxclick={(e: MouseEvent) => {
                  if (e.button === 1) onclose(path);
                }}
                ondragstart={(e: DragEvent) => {
                  dragFrom = i;
                  e.dataTransfer?.setData("application/x-dotmd-tab", path);
                  if (e.dataTransfer) e.dataTransfer.effectAllowed = "move";
                }}
                ondragover={(e: DragEvent) => {
                  if (dragFrom === null) return;
                  e.preventDefault();
                  dropAt = i;
                }}
                ondrop={(e: DragEvent) => {
                  e.preventDefault();
                  if (dragFrom !== null && dragFrom !== i) onreorder(dragFrom, i);
                  endDrag();
                }}
                ondragend={endDrag}
              >
                <!-- A div, so the close button can sit inside the tab. -->
                {#snippet child({ props })}
                  <div {...props}>
                    <FileTextIcon class="size-3.5 opacity-60" />
                    <span class="min-w-0 flex-1 truncate text-start">
                      {labels[i].title}
                      {#if labels[i].folder}
                        <span class="text-xs font-normal text-muted-foreground">
                          — {labels[i].folder}
                        </span>
                      {/if}
                    </span>
                    <Button
                      variant="ghost"
                      size="icon-xs"
                      tabindex={-1}
                      aria-label="Close {labels[i].title}"
                      title="Close ({modKey}W)"
                      class={cn(
                        "size-5 text-muted-foreground hover:bg-foreground/10",
                        !selected && "opacity-0 group-hover/tab:opacity-100",
                      )}
                      onclick={(e: MouseEvent) => {
                        e.stopPropagation(); // not also a click on the tab
                        onclose(path);
                      }}
                    >
                      <XIcon class="size-3.5" />
                    </Button>
                  </div>
                {/snippet}
              </Tabs.Trigger>
            {/snippet}
          </ContextMenu.Trigger>
          <ContextMenu.Content class="w-52">
            <ContextMenu.Item onSelect={() => onclose(path)}>
              Close
              {#if selected}<ContextMenu.Shortcut>{modKey}W</ContextMenu.Shortcut>{/if}
            </ContextMenu.Item>
            <ContextMenu.Item disabled={tabs.length < 2} onSelect={() => oncloseothers(path)}>
              Close others
            </ContextMenu.Item>
            <ContextMenu.Item disabled={i === tabs.length - 1} onSelect={() => oncloseright(path)}>
              Close tabs to the right
            </ContextMenu.Item>
          </ContextMenu.Content>
        </ContextMenu.Root>
      {/each}
    </Tabs.List>
  </Tabs.Root>

  <Button
    variant="ghost"
    size="icon-sm"
    aria-label="Open a note in a new tab"
    title="New tab ({modKey}T)"
    class="text-muted-foreground"
    style="--wails-draggable: no-drag"
    onclick={onnew}
  >
    <PlusIcon />
  </Button>
</div>
