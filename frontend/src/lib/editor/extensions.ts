import type { Extensions } from "@tiptap/core";
import StarterKit from "@tiptap/starter-kit";
import Image from "@tiptap/extension-image";
import { TaskItem, TaskList } from "@tiptap/extension-list";
import { TableKit } from "@tiptap/extension-table";
import { Placeholder } from "@tiptap/extensions";
import { BubbleMenu } from "@tiptap/extension-bubble-menu";
import { Markdown } from "@tiptap/markdown";
import { InlineImageParagraph } from "./paragraph";
import { SlashCommand } from "./slash-command.svelte";

/**
 * Only nodes and marks that have a plain markdown representation are enabled,
 * so serialization never has to fall back to HTML.
 */
export function createExtensions(bubbleMenuElement: HTMLElement): Extensions {
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
    TableKit.configure({ table: { resizable: false } }),
    Placeholder.configure({ placeholder: "Write something, or type '/' for commands…" }),
    BubbleMenu.configure({
      element: bubbleMenuElement,
      shouldShow: ({ editor, from, to }) =>
        from !== to && !editor.isActive("codeBlock") && editor.isEditable,
    }),
    SlashCommand,
    Markdown,
  ];
}
