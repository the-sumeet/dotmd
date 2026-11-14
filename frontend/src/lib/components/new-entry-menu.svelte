<!-- Notion-style "+" button that offers "New note" / "New folder". -->
<script lang="ts">
  import PlusIcon from "@lucide/svelte/icons/plus";
  import FilePlusIcon from "@lucide/svelte/icons/file-plus";
  import FolderPlusIcon from "@lucide/svelte/icons/folder-plus";
  import * as DropdownMenu from "$lib/components/ui/dropdown-menu/index.js";

  let {
    location,
    onnote,
    onfolder,
  }: {
    /** Where new entries go, shown as the menu heading. */
    location: string;
    onnote: () => void;
    onfolder: () => void;
  } = $props();
</script>

<DropdownMenu.Root>
  <DropdownMenu.Trigger>
    {#snippet child({ props })}
      <button
        {...props}
        type="button"
        title="Add to {location}"
        aria-label="Add to {location}"
        class="grid size-5 place-items-center rounded text-sidebar-foreground/60 outline-hidden hover:bg-sidebar-foreground/10 hover:text-sidebar-foreground focus-visible:ring-2 focus-visible:ring-sidebar-ring [&_svg]:size-4"
      >
        <PlusIcon />
      </button>
    {/snippet}
  </DropdownMenu.Trigger>
  <!-- Keep focus from returning to the trigger, so the name dialog keeps it. -->
  <DropdownMenu.Content
    align="start"
    class="w-48"
    onCloseAutoFocus={(e) => e.preventDefault()}
  >
    <DropdownMenu.Label class="truncate text-xs text-muted-foreground">
      {location}
    </DropdownMenu.Label>
    <DropdownMenu.Item onSelect={onnote}>
      <FilePlusIcon /> New note
    </DropdownMenu.Item>
    <DropdownMenu.Item onSelect={onfolder}>
      <FolderPlusIcon /> New folder
    </DropdownMenu.Item>
  </DropdownMenu.Content>
</DropdownMenu.Root>
