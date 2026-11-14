import { Extension } from "@tiptap/core";
import { Fragment, type Node as PMNode, type Schema } from "@tiptap/pm/model";
import { TextSelection } from "@tiptap/pm/state";

/** Leaf inline nodes (images, hard breaks) count as one non-space character. */
const LEAF = "￼";

/**
 * Typing a markdown table header row (`| a | b |`) and pressing Enter at the
 * end of it turns the paragraph into a table with those cells as headers and
 * one empty body row, with the cursor in the first body cell.
 *
 * Only top-level paragraphs convert (GFM has no tables inside lists or
 * quotes), and only rows with at least two cells, so a stray `|x|` line still
 * splits normally. Cell formatting is kept; `\|` and pipes inside inline code
 * do not split cells.
 */
export const TableShortcut = Extension.create({
  name: "tableShortcut",
  // Ahead of the core Enter handling (splitBlock), but below SlashCommand so
  // Enter in an open slash menu still picks the highlighted item.
  priority: 110,

  addKeyboardShortcuts() {
    return {
      Enter: ({ editor }) => {
        const { state } = editor;
        const { selection, schema } = state;
        const { $from } = selection;
        const para = $from.parent;
        if (!selection.empty || $from.depth !== 1 || para.type.name !== "paragraph") return false;
        if ($from.parentOffset !== para.content.size) return false;

        const cells = splitRow(para, schema);
        if (!cells || cells.length < 2) return false;

        const { table, tableRow, tableHeader, tableCell, paragraph } = schema.nodes;
        if (!table || !tableRow || !tableHeader || !tableCell) return false;

        const header = tableRow.create(
          null,
          cells.map((content) => tableHeader.create(null, paragraph.create(null, content))),
        );
        const body = tableRow.create(
          null,
          cells.map(() => tableCell.create(null, paragraph.create())),
        );

        const start = $from.before();
        const tr = state.tr.replaceWith(start, $from.after(), table.create(null, [header, body]));
        // table > body row > first cell > paragraph
        const cursor = start + 1 + header.nodeSize + 1 + 1 + 1;
        tr.setSelection(TextSelection.near(tr.doc.resolve(cursor))).scrollIntoView();
        editor.view.dispatch(tr);
        return true;
      },
    };
  },
});

/**
 * Split a `| a | b |` paragraph into the inline content of each cell, or
 * return null when the paragraph is not shaped like a table row.
 */
function splitRow(para: PMNode, schema: Schema): Fragment[] | null {
  const size = para.content.size;
  // One character per content position, so string indexes are content offsets.
  const text = para.textBetween(0, size, "", LEAF);
  if (text.length !== size) return null;

  const inCode = new Array<boolean>(size).fill(false);
  para.forEach((node, offset) => {
    if (node.isText && node.marks.some((m) => m.type.name === "code")) {
      inCode.fill(true, offset, offset + node.nodeSize);
    }
  });

  const pipes: number[] = [];
  for (let i = 0; i < size; i++) {
    if (inCode[i]) continue;
    if (text[i] === "\\" && i + 1 < size && !inCode[i + 1]) {
      i++; // a backslash escapes the next character: `\|` is a literal pipe, `\\|` a divider
      continue;
    }
    if (text[i] === "|") pipes.push(i);
  }

  const first = text.length - text.trimStart().length;
  const last = text.trimEnd().length - 1;
  if (pipes.length < 2 || pipes[0] !== first || pipes[pipes.length - 1] !== last) return null;

  const cells: Fragment[] = [];
  for (let k = 0; k + 1 < pipes.length; k++) {
    let from = pipes[k] + 1;
    let to = pipes[k + 1];
    while (from < to && /\s/.test(text[from])) from++;
    while (to > from && /\s/.test(text[to - 1])) to--;
    cells.push(unescapePipes(para.content.cut(from, to), schema));
  }
  return cells;
}

/** `\|` in a markdown cell is a literal pipe; store it as one. */
function unescapePipes(content: Fragment, schema: Schema): Fragment {
  const nodes: PMNode[] = [];
  content.forEach((node) => {
    const isCode = node.marks.some((m) => m.type.name === "code");
    nodes.push(
      // Pairs match left to right, so `\\` stays and only a real `\|` unescapes.
      node.isText && !isCode
        ? schema.text(node.text!.replace(/\\([\\|])/g, (m, c) => (c === "|" ? "|" : m)), node.marks)
        : node,
    );
  });
  return Fragment.from(nodes);
}
