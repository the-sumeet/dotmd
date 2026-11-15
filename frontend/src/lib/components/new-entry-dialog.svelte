<!-- Dialog asking for the name of a new note or folder. -->
<script lang="ts">
  import { tick } from "svelte";
  import * as Dialog from "$lib/components/ui/dialog/index.js";
  import { Button } from "$lib/components/ui/button/index.js";
  import { Input } from "$lib/components/ui/input/index.js";
  import type { EntryKind } from "$lib/notes/notes";

  let {
    open = $bindable(false),
    kind,
    parent,
    oncreate,
  }: {
    open?: boolean;
    kind: EntryKind;
    /** Root-relative folder to create in; "" is the notes folder itself. */
    parent: string;
    /** Create the entry at a root-relative path; rejects with a user-facing message. */
    oncreate: (kind: EntryKind, path: string) => Promise<unknown>;
  } = $props();

  let name = $state("");
  let error = $state("");
  let busy = $state(false);
  let input = $state<HTMLInputElement | null>(null);

  const noun = $derived(kind === "note" ? "note" : "folder");

  // Start fresh every time the dialog opens.
  $effect(() => {
    if (open) {
      name = "";
      error = "";
    }
  });

  async function submit(e: SubmitEvent) {
    e.preventDefault();
    const trimmed = name.trim();
    if (!trimmed || busy) return;
    busy = true;
    try {
      await oncreate(kind, parent ? `${parent}/${trimmed}` : trimmed);
      open = false;
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
    } finally {
      busy = false;
    }
    // The field is disabled while busy; focus it once it's enabled again.
    if (error) {
      await tick();
      input?.focus();
    }
  }
</script>

<Dialog.Root bind:open>
  <Dialog.Content
    onOpenAutoFocus={(e) => {
      e.preventDefault();
      input?.focus();
    }}
  >
    <form class="grid gap-4" onsubmit={submit}>
      <Dialog.Header>
        <Dialog.Title>New {noun}</Dialog.Title>
        <Dialog.Description>
          {#if parent}
            Creates the {noun} in <span class="font-medium text-foreground">{parent}</span>.
          {:else}
            Creates the {noun} at the top of your notes folder.
          {/if}
        </Dialog.Description>
      </Dialog.Header>

      <div class="grid gap-1.5">
        <Input
          bind:ref={input}
          bind:value={name}
          placeholder={kind === "note" ? "Meeting notes" : "Projects"}
          aria-label="{noun} name"
          aria-invalid={error ? true : undefined}
          disabled={busy}
          oninput={() => (error = "")}
        />
        {#if error}
          <p class="text-xs text-destructive">{error}</p>
        {/if}
      </div>

      <Dialog.Footer>
        <Dialog.Close>
          {#snippet child({ props })}
            <Button variant="outline" {...props}>Cancel</Button>
          {/snippet}
        </Dialog.Close>
        <Button type="submit" disabled={busy || !name.trim()}>Create</Button>
      </Dialog.Footer>
    </form>
  </Dialog.Content>
</Dialog.Root>
