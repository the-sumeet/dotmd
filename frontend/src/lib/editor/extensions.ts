import { findParentNode, type Editor, type Extensions } from "@tiptap/core";
import StarterKit from "@tiptap/starter-kit";
import Image from "@tiptap/extension-image";
import { TaskItem, TaskList } from "@tiptap/extension-list";
import { TableKit } from "@tiptap/extension-table";
import { Placeholder } from "@tiptap/extensions";
import { BubbleMenu, BubbleMenuPlugin } from "@tiptap/extension-bubble-menu";
import { Markdown } from "@tiptap/markdown";
import { CellSelection } from "@tiptap/pm/tables";
import type { EditorView } from "@tiptap/pm/view";
import { InlineImageParagraph } from "./paragraph";
import { SlashCommand } from "./slash-command.svelte";
import { TableShortcut } from "./table-shortcut";

/**
 * The plugin's default shouldShow hides menus while the editor is unfocused;
 * custom ones must check it too, or a menu shows over a note just opened.
 */
const hasFocus = (view: EditorView, menu: HTMLElement) =>
  view.hasFocus() || menu.contains(document.activeElement);

/** The DOM of the table around the selection, or null. */
function tableAround(editor: Editor): HTMLElement | null {
  if (!editor.isInitialized) return null; // view is a stub until mounted
  const table = findParentNode((node) => node.type.name === "table")(editor.state.selection);
  const dom = table ? editor.view.nodeDOM(table.pos) : null;
  return dom instanceof HTMLElement ? dom : null;
}

/** BubbleMenu anchored to the whole table, so it doesn't jump between cells. */
const TableMenu = BubbleMenu.extend({
  name: "tableMenu",

  addProseMirrorPlugins() {
    const { editor, options } = this;
    if (!options.element) return [];
    return [
      BubbleMenuPlugin({
        editor,
        element: options.element,
        pluginKey: options.pluginKey,
        updateDelay: options.updateDelay,
        options: options.options,
        appendTo: options.appendTo,
        shouldShow: options.shouldShow,
        getReferencedVirtualElement: () => {
          const table = tableAround(editor);
          return table && { getBoundingClientRect: () => table.getBoundingClientRect() };
        },
      }),
    ];
  },
});

/**
 * Only nodes and marks that have a plain markdown representation are enabled,
 * so serialization never has to fall back to HTML.
 */
export function createExtensions(bubbleMenuElement: HTMLElement, tableMenuElement: HTMLElement): Extensions {
  return [
    StarterKit.configure({
      underline: false, // no markdown syntax for underline
      paragraph: false, // replaced by InlineImageParagraph
      link: { openOnClick: false, autolink: true, defaultProtocol: "https" },
    }),
    InlineImageParagraph,
    Image.configure({ inline: true }), // markdown images are inline (`![alt](src)`)
    TaskList,
    TaskItem.configure({ nested: true }),
    TableShortcut,
    TableKit.configure({ table: { resizable: false } }),
    Placeholder.configure({ placeholder: "Write something, or type '/' for commands…" }),
    BubbleMenu.configure({
      element: bubbleMenuElement,
      shouldShow: ({ editor, view, element, state, from, to }) =>
        from !== to &&
        !(state.selection instanceof CellSelection) &&
        !editor.isActive("codeBlock") &&
        editor.isEditable &&
        hasFocus(view, element),
    }),
    TableMenu.configure({
      pluginKey: "tableMenu",
      element: tableMenuElement,
      // Shown for a caret or a cell selection; a text range gets the text menu.
      shouldShow: ({ editor, view, element, state, from, to }) =>
        editor.isEditable &&
        editor.isActive("table") &&
        (from === to || state.selection instanceof CellSelection) &&
        hasFocus(view, element),
      options: { placement: "top-start", offset: 6 },
    }),
    SlashCommand,
    Markdown,
  ];
}
