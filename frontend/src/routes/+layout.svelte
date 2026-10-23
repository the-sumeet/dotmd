<script lang="ts">
  import "../app.css";
  import * as Breadcrumb from "$lib/components/ui/breadcrumb/index.js";
  import * as Sidebar from "$lib/components/ui/sidebar/index.js";
  import { Separator } from "$lib/components/ui/separator/index.js";
  import { Toaster } from "$lib/components/ui/sonner/index.js";
  import { ModeWatcher } from "mode-watcher";
  import AppSidebar from "$lib/components/app-sidebar.svelte";
  import { windowState } from "$lib/stores/window.svelte.js";
  import { editor } from "$lib/stores/editor.svelte.js";
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
</script>

<svelte:window onresize={() => windowState.refresh()} />

<ModeWatcher />
<Toaster position="bottom-right" />

<!-- Wider than the 16rem default so the file explorer (next to the icon rail) has room for names. -->
<!-- Off explorer routes the sidebar is forced to its collapsed icon rail without touching the stored choice. -->
<Sidebar.Provider
  bind:open={() => showExplorer && sidebarOpen, (v) => (sidebarOpen = v)}
  style="--sidebar-width: 20rem;"
>
  <AppSidebar />
  <Sidebar.Inset class="h-svh">
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
      {#if editor.path}
        {@const segments = editor.path.split("/")}
        {@const name = segments[segments.length - 1]}
        <!-- Deep paths collapse to first › … › parent › file (null marks the ellipsis); long names truncate. -->
        {@const folders =
          segments.length > 3
            ? [segments[0], null, segments[segments.length - 2]]
            : segments.slice(0, -1)}
        <Breadcrumb.Root class="min-w-0" title={editor.path}>
          <Breadcrumb.List class="min-w-0 flex-nowrap">
            {#each folders as folder, i (i)}
              <Breadcrumb.Item class="hidden min-w-0 md:inline-flex">
                {#if folder === null}
                  <Breadcrumb.Ellipsis />
                {:else}
                  <span class="max-w-40 truncate">{folder}</span>
                {/if}
              </Breadcrumb.Item>
              <Breadcrumb.Separator class="hidden md:block" />
            {/each}
            <Breadcrumb.Item class="min-w-0">
              <Breadcrumb.Page class="truncate">{name}</Breadcrumb.Page>
              {#if editor.dirty}<span class="text-foreground">•</span>{/if}
            </Breadcrumb.Item>
          </Breadcrumb.List>
        </Breadcrumb.Root>
      {/if}
    </header>
    <div class="flex min-h-0 flex-1 flex-col gap-4 p-4">
      {@render children()}
    </div>
  </Sidebar.Inset>
</Sidebar.Provider>
