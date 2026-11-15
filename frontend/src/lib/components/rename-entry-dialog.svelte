<!-- Dialog asking for a new name for a note, folder or image. -->
<script lang="ts">
  import { tick } from "svelte";
  import * as Dialog from "$lib/components/ui/dialog/index.js";
  import { Button } from "$lib/components/ui/button/index.js";
  import { Input } from "$lib/components/ui/input/index.js";
  import { baseName } from "$lib/notes/images";
  import { noteTitle, type FileKind } from "$lib/notes/notes";

  let {
    open = $bindable(false),
    kind,
    path,
    onrename,
  }: {
    open?: boolean;
    kind: FileKind;
    /** Root-relative path of the entry to rename. */
    path: string;
    /** Rename the entry; rejects with a user-facing message. */
    onrename: (path: string, name: string) => Promise<unknown>;
  } = $props();

  let name = $state("");
  let error = $state("");
  let busy = $state(false);
  let input = $state<HTMLInputElement | null>(null);

  /** The name as shown in the field: notes and images without their extension. */
  const current = $derived(
    kind === "note" ? noteTitle(path) : kind === "image" ? baseName(path) : (path.split("/").pop() ?? path),
  );

  // Start from the current name every time the dialog opens.
  $effect(() => {
    if (open) {
      name = current;
      error = "";
    }
  });

  async function submit(e: SubmitEvent) {
    e.preventDefault();
    const trimmed = name.trim();
    if (!trimmed || trimmed === current || busy) return;
    busy = true;
    try {
      await onrename(path, trimmed);
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
      input?.select();
    }}
  >
    <form class="grid gap-4" onsubmit={submit}>
      <Dialog.Header>
        <Dialog.Title>Rename {kind}</Dialog.Title>
        <Dialog.Description>
          {#if kind === "folder"}
            Links from notes outside this folder to notes inside it won't be updated.
          {:else}
            Links to this {kind} from other notes won't be updated.
          {/if}
        </Dialog.Description>
      </Dialog.Header>

      <div class="grid gap-1.5">
        <Input
          bind:ref={input}
          bind:value={name}
          aria-label="New {kind} name"
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
        <Button type="submit" disabled={busy || !name.trim() || name.trim() === current}>
          Rename
        </Button>
      </Dialog.Footer>
    </form>
  </Dialog.Content>
</Dialog.Root>
