import { findParentNode, type Editor, type Extensions } from "@tiptap/core";
import StarterKit from "@tiptap/starter-kit";
import Image from "@tiptap/extension-image";
import { TaskItem, TaskList } from "@tiptap/extension-list";
import { TableKit } from "@tiptap/extension-table";
import { Placeholder } from "@tiptap/extensions";
import { BubbleMenu, BubbleMenuPlugin } from "@tiptap/extension-bubble-menu";
import Link from "@tiptap/extension-link";
import { Markdown } from "@tiptap/markdown";
import { CellSelection } from "@tiptap/pm/tables";
import type { EditorView } from "@tiptap/pm/view";
import { looksLikeNoteFile } from "$lib/notes/links";
import { LinkOpen } from "./link-open";
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

/** BubbleMenu for a caret inside a link (the text menu needs a selection). */
const LinkMenu = BubbleMenu.extend({ name: "linkMenu" });

/** True if every ")" in `text` closes an earlier "(". */
function balancedParens(text: string): boolean {
  let depth = 0;
  for (const ch of text) {
    if (ch === "(") depth++;
    else if (ch === ")" && --depth < 0) return false;
  }
  return depth === 0;
}

/**
 * Markdown link destination for `href`. One with spaces, angle brackets or
 * unbalanced parentheses (a note named "My Note.md") is wrapped in `<…>`,
 * or it would not parse as a link.
 */
export function markdownHref(href: string): string {
  if (!/[\s<>]/.test(href) && balancedParens(href)) return href;
  return `<${href.replace(/</g, "%3C").replace(/>/g, "%3E")}>`;
}

/**
 * Link mark whose markdown keeps hrefs with spaces valid (see markdownHref)
 * and that never autolinks a note file name.
 */
const NoteLink = Link.extend({
  addOptions() {
    const parent = this.parent!();
    return {
      ...parent,
      shouldAutoLink: (url: string) => !looksLikeNoteFile(url) && parent.shouldAutoLink(url),
    };
  },
  renderMarkdown: (node, h) => {
    const href = markdownHref(node.attrs?.href ?? "");
    const title = (node.attrs?.title ?? "").replace(/[\\"]/g, "\\$&");
    const text = h.renderChildren(node);
    return title ? `[${text}](${href} "${title}")` : `[${text}](${href})`;
  },
});

export interface ExtensionOptions {
  bubbleMenu: HTMLElement;
  tableMenu: HTMLElement;
  linkMenu: HTMLElement;
  /** Called with a link's raw href when it is Cmd/Ctrl+clicked. */
  onOpenLink: (href: string) => void;
}

/**
 * Only nodes and marks that have a plain markdown representation are enabled,
 * so serialization never has to fall back to HTML.
 */
export function createExtensions(options: ExtensionOptions): Extensions {
  return [
    StarterKit.configure({
      underline: false, // no markdown syntax for underline
      paragraph: false, // replaced by InlineImageParagraph
      link: false, // replaced by NoteLink
    }),
    NoteLink.configure({ openOnClick: false, autolink: true, defaultProtocol: "https" }),
    LinkOpen.configure({ onOpen: options.onOpenLink }),
    InlineImageParagraph,
    Image.configure({ inline: true }), // markdown images are inline (`![alt](src)`)
    TaskList,
    TaskItem.configure({ nested: true }),
    TableShortcut,
    TableKit.configure({ table: { resizable: false } }),
    Placeholder.configure({ placeholder: "Write something, or type '/' for commands…" }),
    BubbleMenu.configure({
      element: options.bubbleMenu,
      shouldShow: ({ editor, view, element, state, from, to }) =>
        from !== to &&
        !(state.selection instanceof CellSelection) &&
        !editor.isActive("codeBlock") &&
        editor.isEditable &&
        hasFocus(view, element),
    }),
    TableMenu.configure({
      pluginKey: "tableMenu",
      element: options.tableMenu,
      // Shown for a caret or a cell selection; a text range gets the text menu.
      shouldShow: ({ editor, view, element, state, from, to }) =>
        editor.isEditable &&
        editor.isActive("table") &&
        (from === to || state.selection instanceof CellSelection) &&
        hasFocus(view, element),
      options: { placement: "top-start", offset: 6 },
    }),
    LinkMenu.configure({
      pluginKey: "linkMenu",
      element: options.linkMenu,
      // Below the line, so it never covers the table menu above a table.
      shouldShow: ({ editor, view, element, from, to }) =>
        editor.isEditable && from === to && editor.isActive("link") && hasFocus(view, element),
      options: { placement: "bottom-start", offset: 6 },
    }),
    SlashCommand,
    Markdown,
  ];
}
