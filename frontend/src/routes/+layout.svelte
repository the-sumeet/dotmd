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
        <Breadcrumb.Root>
          <Breadcrumb.List>
            {#each segments as segment, i (i)}
              {#if i < segments.length - 1}
                <Breadcrumb.Item class="hidden md:block">{segment}</Breadcrumb.Item>
                <Breadcrumb.Separator class="hidden md:block" />
              {:else}
                <Breadcrumb.Item>
                  <Breadcrumb.Page>{segment}{editor.dirty ? " •" : ""}</Breadcrumb.Page>
                </Breadcrumb.Item>
              {/if}
            {/each}
          </Breadcrumb.List>
        </Breadcrumb.Root>
      {/if}
    </header>
    <div class="flex min-h-0 flex-1 flex-col gap-4 p-4">
      {@render children()}
    </div>
  </Sidebar.Inset>
</Sidebar.Provider>
