<script lang="ts">
  import "../app.css";
  import * as Breadcrumb from "$lib/components/ui/breadcrumb/index.js";
  import * as Sidebar from "$lib/components/ui/sidebar/index.js";
  import { Separator } from "$lib/components/ui/separator/index.js";
  import { Toaster } from "$lib/components/ui/sonner/index.js";
  import AppSidebar from "$lib/components/app-sidebar.svelte";
  import { windowState } from "$lib/stores/window.svelte.js";
  import { editor } from "$lib/stores/editor.svelte.js";
  let { children } = $props();

  $effect(() => {
    windowState.refresh();
  });
</script>

<svelte:window onresize={() => windowState.refresh()} />

<Toaster position="bottom-right" />

<Sidebar.Provider>
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
      <Sidebar.Trigger class="-ms-1" />
      <Separator
        orientation="vertical"
        class="me-2 data-vertical:h-4 data-vertical:self-auto"
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
