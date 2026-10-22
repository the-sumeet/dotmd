<script lang="ts">
	import ChevronRightIcon from "@lucide/svelte/icons/chevron-right";
	import FileIcon from "@lucide/svelte/icons/file";
	import FolderIcon from "@lucide/svelte/icons/folder";
	import * as Collapsible from "$lib/components/ui/collapsible/index.js";
	import * as Sidebar from "$lib/components/ui/sidebar/index.js";
	import type { ComponentProps } from "svelte";
	import { onMount } from "svelte";
	import { FileService } from "../../../bindings/changeme";
	import type { FileNode } from "../../../bindings/changeme/models.js";

	let { ref = $bindable(null), ...restProps }: ComponentProps<typeof Sidebar.Root> = $props();

	let tree = $state<FileNode[]>([]);

	onMount(async () => {
		try {
			tree = (await FileService.ListTree()) ?? [];
		} catch (err) {
			console.error("Failed to load data directory", err);
		}
	});
</script>

<Sidebar.Root bind:ref {...restProps}>
	<!-- Reserve space for the macOS traffic lights (hidden-inset title bar) and keep it draggable. -->
	<Sidebar.Header class="h-12 shrink-0" style="--wails-draggable: drag" />
	<Sidebar.Content>
		<Sidebar.Group>
			<Sidebar.GroupLabel>Files</Sidebar.GroupLabel>
			<Sidebar.GroupContent>
				<Sidebar.Menu>
					{#each tree as node (node.path)}
						{@render Tree({ node })}
					{/each}
				</Sidebar.Menu>
			</Sidebar.GroupContent>
		</Sidebar.Group>
	</Sidebar.Content>
	<Sidebar.Rail />
</Sidebar.Root>

{#snippet Tree({ node }: { node: FileNode })}
	{#if !node.isDir}
		<Sidebar.MenuItem>
			<Sidebar.MenuButton>
				<FileIcon />
				{node.name}
			</Sidebar.MenuButton>
		</Sidebar.MenuItem>
	{:else}
		<Sidebar.MenuItem>
			<Collapsible.Root
				class="group/collapsible [&[data-state=open]>button>svg:first-child]:rotate-90"
			>
				<Collapsible.Trigger>
					{#snippet child({ props })}
						<Sidebar.MenuButton {...props}>
							<ChevronRightIcon class="transition-transform" />
							<FolderIcon />
							{node.name}
						</Sidebar.MenuButton>
					{/snippet}
				</Collapsible.Trigger>
				<Collapsible.Content>
					<Sidebar.MenuSub>
						{#each node.children ?? [] as child (child.path)}
							{@render Tree({ node: child })}
						{/each}
					</Sidebar.MenuSub>
				</Collapsible.Content>
			</Collapsible.Root>
		</Sidebar.MenuItem>
	{/if}
{/snippet}
