<!-- Confirms deleting an image file from the notes folder. -->
<script lang="ts">
  import * as Dialog from "$lib/components/ui/dialog/index.js";
  import { Button } from "$lib/components/ui/button/index.js";

  let {
    open = $bindable(false),
    path,
    ondelete,
  }: {
    open?: boolean;
    /** Root-relative path of the image to delete. */
    path: string;
    /** Delete the image; rejects with a user-facing message. */
    ondelete?: (path: string) => Promise<void>;
  } = $props();

  let error = $state("");
  let busy = $state(false);

  const name = $derived(path.split("/").pop() ?? path);

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
      <Dialog.Title>Delete image?</Dialog.Title>
      <Dialog.Description>
        <span class="font-medium break-all text-foreground">{name}</span> will be deleted from
        your notes folder. Notes that show it will have a broken image.
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
      <Button variant="destructive" disabled={busy} onclick={confirm}>Delete</Button>
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>
