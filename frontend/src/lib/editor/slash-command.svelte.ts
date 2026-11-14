import { Extension } from "@tiptap/core";
import { PluginKey } from "@tiptap/pm/state";
import Suggestion, { type SuggestionOptions, type SuggestionProps } from "@tiptap/suggestion";
import { computePosition, flip, offset, shift } from "@floating-ui/dom";
import { mount, unmount } from "svelte";
import SlashMenu from "$lib/components/editor/SlashMenu.svelte";
import { filterSlashItems, type SlashItem } from "./slash-items";

type Props = SuggestionProps<SlashItem, SlashItem>;

function renderSlashMenu(): ReturnType<NonNullable<SuggestionOptions<SlashItem, SlashItem>["render"]>> {
  let container: HTMLDivElement | null = null;
  let instance: ReturnType<typeof mount> | null = null;
  const state = $state<{ items: SlashItem[]; command: (item: SlashItem) => void }>({
    items: [],
    command: () => {},
  });

  const position = (props: Props) => {
    const rect = props.clientRect?.();
    if (!container || !rect) return;
    const reference = { getBoundingClientRect: () => rect };
    computePosition(reference, container, {
      placement: "bottom-start",
      strategy: "fixed",
      middleware: [offset(6), flip(), shift({ padding: 8 })],
    }).then(({ x, y }) => {
      if (container) Object.assign(container.style, { left: `${x}px`, top: `${y}px` });
    });
  };

  return {
    onStart(props) {
      state.items = props.items;
      state.command = props.command;
      container = document.createElement("div");
      Object.assign(container.style, { position: "fixed", zIndex: "50", left: "0", top: "0" });
      document.body.appendChild(container);
      instance = mount(SlashMenu, { target: container, props: state });
      position(props);
    },
    onUpdate(props) {
      state.items = props.items;
      state.command = props.command;
      position(props);
    },
    onKeyDown({ event }) {
      if (event.key === "Escape") return false;
      return (instance as { onKeyDown?: (e: KeyboardEvent) => boolean } | null)?.onKeyDown?.(event) ?? false;
    },
    onExit() {
      if (instance) unmount(instance);
      container?.remove();
      instance = null;
      container = null;
    },
  };
}

export const SlashCommand = Extension.create({
  name: "slashCommand",

  addProseMirrorPlugins() {
    return [
      Suggestion<SlashItem, SlashItem>({
        editor: this.editor,
        pluginKey: new PluginKey("slashCommand"),
        char: "/",
        allowSpaces: false,
        items: ({ query }) => filterSlashItems(query),
        command: ({ editor, range, props }) => props.command({ editor, range }),
        render: renderSlashMenu,
      }),
    ];
  },
});
