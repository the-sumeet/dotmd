import type { Editor, Range } from "@tiptap/core";
import {
  Code,
  Heading1,
  Heading2,
  Heading3,
  ImagePlus,
  List,
  ListChecks,
  ListOrdered,
  Minus,
  Pilcrow,
  Quote,
  Table,
} from "@lucide/svelte";

export interface SlashItem {
  title: string;
  description: string;
  icon: typeof Pilcrow;
  keywords: string[];
  command: (props: { editor: Editor; range: Range }) => void;
}

const run = (editor: Editor, range: Range) => editor.chain().focus().deleteRange(range);

export const slashItems: SlashItem[] = [
  {
    title: "Text",
    description: "Plain paragraph",
    icon: Pilcrow,
    keywords: ["paragraph", "p"],
    command: ({ editor, range }) => run(editor, range).setParagraph().run(),
  },
  {
    title: "Heading 1",
    description: "Large section heading",
    icon: Heading1,
    keywords: ["h1", "title"],
    command: ({ editor, range }) => run(editor, range).setHeading({ level: 1 }).run(),
  },
  {
    title: "Heading 2",
    description: "Medium section heading",
    icon: Heading2,
    keywords: ["h2", "subtitle"],
    command: ({ editor, range }) => run(editor, range).setHeading({ level: 2 }).run(),
  },
  {
    title: "Heading 3",
    description: "Small section heading",
    icon: Heading3,
    keywords: ["h3"],
    command: ({ editor, range }) => run(editor, range).setHeading({ level: 3 }).run(),
  },
  {
    title: "Bullet list",
    description: "Unordered list",
    icon: List,
    keywords: ["ul", "unordered", "bullets"],
    command: ({ editor, range }) => run(editor, range).toggleBulletList().run(),
  },
  {
    title: "Numbered list",
    description: "Ordered list",
    icon: ListOrdered,
    keywords: ["ol", "ordered", "numbers"],
    command: ({ editor, range }) => run(editor, range).toggleOrderedList().run(),
  },
  {
    title: "Task list",
    description: "Checklist with checkboxes",
    icon: ListChecks,
    keywords: ["todo", "checkbox", "check"],
    command: ({ editor, range }) => run(editor, range).toggleTaskList().run(),
  },
  {
    title: "Quote",
    description: "Block quotation",
    icon: Quote,
    keywords: ["blockquote", "citation"],
    command: ({ editor, range }) => run(editor, range).toggleBlockquote().run(),
  },
  {
    title: "Code block",
    description: "Fenced code",
    icon: Code,
    keywords: ["code", "pre", "fence"],
    command: ({ editor, range }) => run(editor, range).toggleCodeBlock().run(),
  },
  {
    title: "Divider",
    description: "Horizontal rule",
    icon: Minus,
    keywords: ["hr", "rule", "separator"],
    command: ({ editor, range }) => run(editor, range).setHorizontalRule().run(),
  },
  {
    title: "Image",
    description: "Add an image from your computer",
    icon: ImagePlus,
    keywords: ["img", "picture", "photo", "upload"],
    command: ({ editor, range }) => run(editor, range).pickImage().run(),
  },
  {
    title: "Table",
    description: "3×3 table with header",
    icon: Table,
    keywords: ["grid", "|"],
    command: ({ editor, range }) =>
      run(editor, range).insertTable({ rows: 3, cols: 3, withHeaderRow: true }).run(),
  },
];

export function filterSlashItems(query: string): SlashItem[] {
  const q = query.trim().toLowerCase();
  if (!q) return slashItems;
  return slashItems.filter(
    (item) => item.title.toLowerCase().includes(q) || item.keywords.some((k) => k.startsWith(q)),
  );
}
