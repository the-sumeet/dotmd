<!-- Confirms moving a note, folder or image to the Trash. -->
<script lang="ts">
  import * as Dialog from "$lib/components/ui/dialog/index.js";
  import { Button } from "$lib/components/ui/button/index.js";
  import { noteTitle, type FileKind } from "$lib/notes/notes";

  let {
    open = $bindable(false),
    kind,
    path,
    noteCount = 0,
    holdsOpenNote = false,
    ondelete,
  }: {
    open?: boolean;
    kind: FileKind;
    /** Root-relative path of the entry to delete. */
    path: string;
    /** For a folder: how many notes are inside it, at any depth. */
    noteCount?: number;
    /** The entry is the open note, or a folder holding it. */
    holdsOpenNote?: boolean;
    /** Delete the entry; rejects with a user-facing message. */
    ondelete?: (path: string) => Promise<void>;
  } = $props();

  let error = $state("");
  let busy = $state(false);

  const name = $derived(kind === "note" ? noteTitle(path) : (path.split("/").pop() ?? path));

  $effect(() => {
    if (open) error = "";
  });

  async function confirm() {
    if (busy) return;
    busy = true;
    try {
      await ondelete?.(path);
      open = false;
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
    } finally {
      busy = false;
    }
  }
</script>

<Dialog.Root bind:open>
  <Dialog.Content>
    <Dialog.Header>
      <Dialog.Title>Delete {kind}?</Dialog.Title>
      <Dialog.Description>
        <span class="font-medium break-all text-foreground">{name}</span>
        {#if kind === "image"}
          will be moved to the Trash. Notes that show it will have a broken image.
        {:else if kind === "folder"}
          and everything in it{noteCount
            ? ` (${noteCount} ${noteCount === 1 ? "note" : "notes"})`
            : ""} will be moved to the Trash.
        {:else}
          will be moved to the Trash.
        {/if}
        {#if holdsOpenNote}
          {kind === "note" ? "It's" : "It holds the note that's"} open now, which will be closed.
        {/if}
      </Dialog.Description>
    </Dialog.Header>
    {#if error}
      <p class="text-xs text-destructive">{error}</p>
    {/if}
    <Dialog.Footer>
      <Dialog.Close>
        {#snippet child({ props })}
          <Button variant="outline" {...props}>Cancel</Button>
        {/snippet}
      </Dialog.Close>
      <Button variant="destructive" disabled={busy} onclick={confirm}>
        Move to Trash
      </Button>
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>
