<!-- App settings: notes folder, theme and layout. -->
<script lang="ts">
  import MonitorIcon from "@lucide/svelte/icons/monitor";
  import SunIcon from "@lucide/svelte/icons/sun";
  import MoonIcon from "@lucide/svelte/icons/moon";
  import type { Component } from "svelte";
  import * as Dialog from "$lib/components/ui/dialog/index.js";
  import { Button } from "$lib/components/ui/button/index.js";
  import { Label } from "$lib/components/ui/label/index.js";
  import { Separator } from "$lib/components/ui/separator/index.js";
  import { Switch } from "$lib/components/ui/switch/index.js";
  import { settings, type Theme } from "$lib/settings.svelte";

  let {
    open = $bindable(false),
    onchoosefolder,
  }: {
    open?: boolean;
    /** Pick and switch to another notes folder; rejects with a user-facing message. */
    onchoosefolder: () => Promise<void>;
  } = $props();

  const themes: { value: Theme; label: string; icon: Component }[] = [
    { value: "system", label: "System", icon: MonitorIcon },
    { value: "light", label: "Light", icon: SunIcon },
    { value: "dark", label: "Dark", icon: MoonIcon },
  ];

  let error = $state("");
  let choosing = $state(false);

  $effect(() => {
    if (open) error = "";
  });

  /** Run a settings change, showing why it failed instead of throwing. */
  async function attempt(change: () => Promise<unknown>) {
    error = "";
    try {
      await change();
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
    }
  }

  async function chooseFolder() {
    choosing = true;
    try {
      await attempt(onchoosefolder);
    } finally {
      choosing = false;
    }
  }
</script>

<Dialog.Root bind:open>
  <Dialog.Content class="sm:max-w-md">
    <Dialog.Header>
      <Dialog.Title>Settings</Dialog.Title>
    </Dialog.Header>

    <div class="grid gap-4">
      {#if settings.canChooseFolder}
        <div class="grid gap-1.5">
          <span class="text-sm font-medium">Notes folder</span>
          <div class="flex items-center gap-2">
            <p
              class="min-w-0 flex-1 truncate rounded-md text-left bg-muted px-2 py-1 font-mono text-xs text-muted-foreground"
              title={settings.notesFolder}
              dir="rtl"
            >
              <!-- rtl keeps the end of a long path (the folder's own name) in view. -->
              <bdi>{settings.notesFolder}</bdi>
            </p>
            <Button variant="outline" size="sm" disabled={choosing} onclick={chooseFolder}>
              Change…
            </Button>
          </div>
        </div>
        <Separator />
      {/if}

      <div class="flex items-center justify-between gap-4">
        <span id="theme-label" class="text-sm font-medium">Theme</span>
        <div
          role="group"
          aria-labelledby="theme-label"
          class="flex shrink-0 items-center gap-0.5 rounded-lg bg-muted p-0.5"
        >
          {#each themes as theme (theme.value)}
            {@const selected = settings.theme === theme.value}
            <Button
              size="xs"
              variant={selected ? "outline" : "ghost"}
              aria-pressed={selected}
              onclick={() => attempt(() => settings.setTheme(theme.value))}
            >
              <theme.icon data-icon="inline-start" />
              {theme.label}
            </Button>
          {/each}
        </div>
      </div>

      <div class="flex items-center justify-between gap-4">
        <Label for="wide-layout" class="grid gap-0.5">
          <span>Wide layout</span>
          <span class="text-xs font-normal text-muted-foreground">
            Let note text use more of the window's width.
          </span>
        </Label>
        <Switch
          id="wide-layout"
          bind:checked={
            () => settings.wideLayout, (on) => void attempt(() => settings.setWideLayout(on))
          }
        />
      </div>

      {#if error}
        <p role="alert" class="text-xs text-destructive">{error}</p>
      {/if}
    </div>
  </Dialog.Content>
</Dialog.Root>
