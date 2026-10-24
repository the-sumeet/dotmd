<script lang="ts">
  import "../app.css";
  import * as Sidebar from "$lib/components/ui/sidebar/index.js";
  import { Separator } from "$lib/components/ui/separator/index.js";
  import { Toaster } from "$lib/components/ui/sonner/index.js";
  import { ModeWatcher } from "mode-watcher";
  import { Events, Window } from "@wailsio/runtime";
  import AppSidebar from "$lib/components/app-sidebar.svelte";
  import { windowState } from "$lib/stores/window.svelte.js";
  import { editor } from "$lib/stores/editor.svelte.js";
  import { tabs } from "$lib/stores/tabs.svelte.js";
  import { tabShortcut } from "$lib/tab-shortcuts.js";
  import TabBar from "$lib/components/tab-bar.svelte";
  import { page } from "$app/state";
  let { children } = $props();

  // Routes that show the file explorer; every other route gets just the icon rail.
  const EXPLORER_ROUTES = ["/"];
  const showExplorer = $derived(EXPLORER_ROUTES.includes(page.route.id ?? ""));

  // The user's own expanded/collapsed choice, kept while the explorer is hidden.
  let sidebarOpen = $state(true);

  $effect(() => {
    windowState.refresh();
  });

  $effect(() => {
    void tabs.restore();
  });

  $effect(() => {
    tabs.save();
  });

  // The native Close menu item (Cmd/Ctrl+W) closes the tab, or the window if there is none to close.
  $effect(() => {
    try {
      return Events.On("tab:close", () => {
        if (editor.path !== null && tabs.paths.includes(editor.path))
          void tabs.close(editor.path);
        else void Window.Close();
      });
    } catch (err) {
      console.error("Failed to listen for tab:close", err);
    }
  });

  // Capture phase, so the editor's own key bindings don't swallow tab shortcuts.
  function onKeydown(e: KeyboardEvent) {
    const action = tabShortcut(e);
    if (action === null) return;
    const acted =
      action.type === "step"
        ? tabs.step(action.step)
        : tabs.select(action.index);
    // With nothing to switch to, leave the key for inputs and dialogs.
    if (!acted) return;
    e.preventDefault();
    e.stopPropagation();
  }
</script>

<svelte:window
  onresize={() => windowState.refresh()}
  onkeydowncapture={onKeydown}
/>

<ModeWatcher />
<Toaster position="bottom-right" />

<!-- Wider than the 16rem default so the file explorer (next to the icon rail) has room for names. -->
<!-- Off explorer routes the sidebar is forced to its collapsed icon rail without touching the stored choice. -->
<Sidebar.Provider
  bind:open={() => showExplorer && sidebarOpen, (v) => (sidebarOpen = v)}
  style="--sidebar-width: 20rem;"
>
  <AppSidebar />
  <Sidebar.Inset class="h-svh min-w-0">
    <!-- When the sidebar is collapsed only the narrow icon rail is left of the header, so clear the rest of the macOS traffic lights. -->
    <header
      class={[
        "flex h-16 shrink-0 items-center gap-2 border-b px-4 transition-[padding]",
        !windowState.fullscreen &&
          "group-has-data-[state=collapsed]/sidebar-wrapper:ps-12",
      ]}
    >
      <!-- Nothing to toggle without the explorer, except on mobile where the trigger opens the rail sheet. -->
      <Sidebar.Trigger class={["-ms-1", !showExplorer && "md:hidden"]} />
      <Separator
        orientation="vertical"
        class={[
          "me-2 data-vertical:h-4 data-vertical:self-auto",
          !showExplorer && "md:hidden",
        ]}
      />
      {#if tabs.paths.length > 0}
        <TabBar />
      {/if}
    </header>
    <div class="flex min-h-0 flex-1 flex-col gap-4 p-4">
      {@render children()}
    </div>
  </Sidebar.Inset>
</Sidebar.Provider>
