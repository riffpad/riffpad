import { useMemo } from "react";
import { marked } from "marked";
import { markedHighlight } from "marked-highlight";
import hljs from "highlight.js/lib/core";
import javascript from "highlight.js/lib/languages/javascript";
import typescript from "highlight.js/lib/languages/typescript";
import python from "highlight.js/lib/languages/python";
import bash from "highlight.js/lib/languages/bash";
import json from "highlight.js/lib/languages/json";
import go from "highlight.js/lib/languages/go";
import rust from "highlight.js/lib/languages/rust";
import diff from "highlight.js/lib/languages/diff";
import xml from "highlight.js/lib/languages/xml";
import plaintext from "highlight.js/lib/languages/plaintext";
import DOMPurify from "dompurify";
import "highlight.js/styles/github-dark.css";

hljs.registerLanguage("javascript", javascript);
hljs.registerLanguage("typescript", typescript);
hljs.registerLanguage("python", python);
hljs.registerLanguage("bash", bash);
hljs.registerLanguage("json", json);
hljs.registerLanguage("go", go);
hljs.registerLanguage("rust", rust);
hljs.registerLanguage("diff", diff);
hljs.registerLanguage("xml", xml);
hljs.registerLanguage("plaintext", plaintext);

marked.use(
  markedHighlight({
    langPrefix: "hljs language-",
    highlight(code, lang) {
      // Unknown or empty language tags fall back to plain text. Without this
      // module, hljs.highlight throws "Unknown language: plaintext" and the
      // exception tears down the whole React tree (#228).
      const language = hljs.getLanguage(lang) ? lang : "plaintext";
      return hljs.highlight(code, { language }).value;
    },
  }),
);
marked.setOptions({ gfm: true, breaks: true });

export function markdownToHtml(text: string): string {
  const raw = marked.parse(text, { async: false }) as string;
  return DOMPurify.sanitize(raw);
}

export default function Markdown({ text }: { text: string }) {
  const html = useMemo(() => markdownToHtml(text), [text]);
  return <div className="md" dangerouslySetInnerHTML={{ __html: html }} />;
}
