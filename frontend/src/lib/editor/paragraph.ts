import Paragraph from "@tiptap/extension-paragraph";

/**
 * Paragraph whose markdown parser keeps a lone image inside the paragraph.
 *
 * The stock parser unwraps `![alt](src)` on its own line into a block-level
 * image node. dotmd registers Image as inline (markdown images are inline),
 * so that unwrapped node would be invalid at the top level of the document.
 */
export const InlineImageParagraph = Paragraph.extend({
  parseMarkdown(token, helpers) {
    const tokens = token.tokens ?? [];
    if (tokens.length === 1 && tokens[0].type === "image") {
      return helpers.createNode("paragraph", undefined, helpers.parseInline(tokens));
    }
    return this.parent!(token, helpers);
  },
});
