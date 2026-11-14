import { Extension } from "@tiptap/core";
import { Plugin, PluginKey } from "@tiptap/pm/state";
import type { EditorView } from "@tiptap/pm/view";

export interface LinkOpenOptions {
  /** Called with the link's raw href (e.g. "../Plan.md"), as written in the note. */
  onOpen: (href: string) => void;
}

/** The link under a Cmd/Ctrl+click inside the editor, or null. */
function modClickedLink(view: EditorView, event: MouseEvent): HTMLAnchorElement | null {
  if (event.button !== 0 || !(event.metaKey || event.ctrlKey)) return null;
  const link = (event.target as Element | null)?.closest?.("a");
  return link && view.dom.contains(link) ? link : null;
}

/**
 * Cmd/Ctrl+click follows a link. A plain click keeps placing the caret,
 * since the editor is always editable.
 */
export const LinkOpen = Extension.create<LinkOpenOptions>({
  name: "linkOpen",

  addOptions() {
    return { onOpen: () => {} };
  },

  addProseMirrorPlugins() {
    const { onOpen } = this.options;
    return [
      new Plugin({
        key: new PluginKey("linkOpen"),
        props: {
          // handleClick runs before ProseMirror applies its own Cmd+click
          // (select the whole paragraph); returning true skips that.
          handleClick(view, _pos, event) {
            const link = modClickedLink(view, event);
            // The attribute, not `link.href`: the browser would resolve a
            // relative path against the app's own URL.
            const href = link?.getAttribute("href");
            if (!href) return false;
            onOpen(href);
            return true;
          },
          handleDOMEvents: {
            // The native click comes after; never let the webview navigate.
            click(view, event) {
              if (modClickedLink(view, event)) event.preventDefault();
              return false;
            },
          },
        },
      }),
    ];
  },
});
