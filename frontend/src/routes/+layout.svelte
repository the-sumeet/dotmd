<script lang="ts">
  import "../app.css";
  import * as Breadcrumb from "$lib/components/ui/breadcrumb/index.js";
  import * as Sidebar from "$lib/components/ui/sidebar/index.js";
  import { Separator } from "$lib/components/ui/separator/index.js";
  import { Window } from "@wailsio/runtime";
  import AppSidebar from "$lib/components/app-sidebar.svelte";
  let { children } = $props();

  // In native macOS fullscreen the traffic lights are hidden and the window
  // fills the whole screen, so no space needs to be reserved for them.
  let fullscreen = $state(false);
  async function checkFullscreen() {
    try {
      fullscreen = await Window.IsFullscreen();
    } catch (err) {
      console.error("Failed to read fullscreen state", err);
    }
  }
  $effect(checkFullscreen);
</script>

<svelte:window onresize={checkFullscreen} />

<Sidebar.Provider>
  <AppSidebar />
  <Sidebar.Inset class="h-svh">
    <!-- When the sidebar is collapsed the header reaches the window's left edge, so clear the macOS traffic lights. -->
    <header
      class={[
        "flex h-16 shrink-0 items-center gap-2 border-b px-4 transition-[padding]",
        !fullscreen &&
          "group-has-data-[state=collapsed]/sidebar-wrapper:ps-24",
      ]}
    >
      <Sidebar.Trigger class="-ms-1" />
      <Separator
        orientation="vertical"
        class="me-2 data-vertical:h-4 data-vertical:self-auto"
      />
      <Breadcrumb.Root>
        <Breadcrumb.List>
          <Breadcrumb.Item class="hidden md:block">
            <Breadcrumb.Link href="##">lib</Breadcrumb.Link>
          </Breadcrumb.Item>
          <Breadcrumb.Separator class="hidden md:block" />
          <Breadcrumb.Item class="hidden md:block">
            <Breadcrumb.Link href="##">components</Breadcrumb.Link>
          </Breadcrumb.Item>
          <Breadcrumb.Separator class="hidden md:block" />
          <Breadcrumb.Item>
            <Breadcrumb.Page>button.svelte</Breadcrumb.Page>
          </Breadcrumb.Item>
        </Breadcrumb.List>
      </Breadcrumb.Root>
    </header>
    <div class="flex min-h-0 flex-1 flex-col gap-4 p-4">
      {@render children()}
    </div>
  </Sidebar.Inset>
</Sidebar.Provider>
