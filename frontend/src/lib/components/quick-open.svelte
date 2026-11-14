<!--
  Quick switcher (⌘P): type part of a note's name or folder, pick with the
  arrow keys and Enter. With no query it lists recently opened notes first.
-->
<script lang="ts">
  import FileTextIcon from "@lucide/svelte/icons/file-text";
  import SearchIcon from "@lucide/svelte/icons/search";
  import CornerDownLeftIcon from "@lucide/svelte/icons/corner-down-left";
  import { Dialog as DialogPrimitive } from "bits-ui";
  import * as Dialog from "$lib/components/ui/dialog/index.js";
  import { noteTitle } from "$lib/notes/notes";
  import { cn } from "$lib/utils.js";

  let {
    open = $bindable(false),
    notes,
    recent = [],
    onopen,
  }: {
    open?: boolean;
    /** Root-relative paths of every note. */
    notes: string[];
    /** Recently opened notes, most recent first. */
    recent?: string[];
    onopen: (path: string) => void;
  } = $props();

  /** Results shown at once; typing narrows the rest. */
  const LIMIT = 50;

  let query = $state("");
  let selected = $state(0);
  let input = $state<HTMLInputElement | null>(null);
  let list = $state<HTMLElement | null>(null);

  /**
   * Lower is better; null if `q` doesn't match. Title prefix beats title
   * substring beats path substring beats letters in order (fuzzy).
   */
  function score(path: string, q: string): number | null {
    const title = noteTitle(path).toLowerCase();
    const lower = path.toLowerCase();
    if (title.startsWith(q)) return 0;
    if (title.includes(q)) return 1;
    if (lower.includes(q)) return 2;
    let i = 0;
    for (const ch of lower) if (ch === q[i]) i++;
    return i === q.length ? 3 : null;
  }

  /** Recent notes that still exist; listed first when there's no query. */
  const recents = $derived.by(() => {
    const known = new Set(notes);
    return recent.filter((p) => known.has(p));
  });

  /** How many of `results` are recents (headed "Recent"); 0 while searching. */
  const recentCount = $derived(query.trim() ? 0 : recents.length);

  const results = $derived.by(() => {
    const q = query.trim().toLowerCase();
    if (!q) {
      const seen = new Set(recents);
      return [...recents, ...notes.filter((p) => !seen.has(p))].slice(0, LIMIT);
    }
    return notes
      .map((path) => ({ path, s: score(path, q) }))
      .filter((r): r is { path: string; s: number } => r.s !== null)
      .sort((a, b) => a.s - b.s || a.path.length - b.path.length)
      .slice(0, LIMIT)
      .map((r) => r.path);
  });

  // Start fresh every time the switcher opens.
  $effect(() => {
    if (open) {
      query = "";
      selected = 0;
    }
  });

  function choose(path: string | undefined) {
    if (!path) return;
    open = false;
    onopen(path);
  }

  function move(by: number) {
    if (!results.length) return;
    selected = (selected + by + results.length) % results.length;
    list?.querySelector(`[data-index="${selected}"]`)?.scrollIntoView({ block: "nearest" });
  }

  function onkeydown(e: KeyboardEvent) {
    if (e.key === "ArrowDown" || (e.ctrlKey && e.key === "n")) move(1);
    else if (e.key === "ArrowUp" || (e.ctrlKey && e.key === "p")) move(-1);
    else if (e.key === "Enter" && !e.isComposing) choose(results[selected]);
    else return;
    e.preventDefault();
    e.stopPropagation(); // Ctrl+N/P here mustn't also reach the app shortcuts
  }

  function folderOf(path: string) {
    return path.split("/").slice(0, -1).join(" / ");
  }
</script>

<Dialog.Root bind:open>
  <Dialog.Content
    showCloseButton={false}
    class="top-[15%] translate-y-0 gap-0 overflow-hidden p-0 sm:max-w-lg"
    onOpenAutoFocus={(e) => {
      e.preventDefault();
      input?.focus();
    }}
  >
    <DialogPrimitive.Title class="sr-only">Go to note</DialogPrimitive.Title>
    <div class="flex items-center gap-2 border-b px-3">
      <SearchIcon class="size-4 shrink-0 text-muted-foreground" />
      <input
        bind:this={input}
        bind:value={query}
        oninput={() => (selected = 0)}
        {onkeydown}
        placeholder="Go to note…"
        aria-label="Note name"
        aria-controls="quick-open-results"
        aria-activedescendant={results.length ? `quick-open-${selected}` : undefined}
        role="combobox"
        aria-expanded="true"
        autocomplete="off"
        spellcheck="false"
        class="h-11 w-full bg-transparent text-sm outline-none placeholder:text-muted-foreground"
      />
    </div>

    <div
      bind:this={list}
      id="quick-open-results"
      role="listbox"
      class="max-h-80 overflow-y-auto p-1.5"
    >
      {#each results as path, i (path)}
        {#if recentCount && (i === 0 || i === recentCount)}
          <p
            class={cn(
              "px-2 pb-1.5 text-xs font-medium text-muted-foreground",
              i === 0 ? "pt-1" : "pt-3",
            )}
          >
            {i === 0 ? "Recent" : "All notes"}
          </p>
        {/if}
        <div
          id="quick-open-{i}"
          data-index={i}
          role="option"
          tabindex="-1"
          aria-selected={i === selected}
          class={cn(
            "flex cursor-pointer items-center gap-2.5 rounded-md px-2 py-1.5",
            i === selected && "bg-accent text-accent-foreground",
          )}
          onpointermove={() => (selected = i)}
          onclick={() => choose(path)}
          onkeydown={() => {}}
        >
          <FileTextIcon class="size-4 shrink-0 text-muted-foreground" />
          <span class="truncate">{noteTitle(path)}</span>
          {#if folderOf(path)}
            <span class="ms-auto shrink-0 truncate ps-2 text-xs text-muted-foreground">
              {folderOf(path)}
            </span>
          {/if}
          {#if i === selected}
            <CornerDownLeftIcon
              class={cn("size-3.5 shrink-0 text-muted-foreground", !folderOf(path) && "ms-auto")}
            />
          {/if}
        </div>
      {:else}
        <p class="px-2 py-6 text-center text-muted-foreground">
          {notes.length ? "No matching notes" : "No notes yet"}
        </p>
      {/each}
    </div>
  </Dialog.Content>
</Dialog.Root>
