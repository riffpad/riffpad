"use client";

import { useTheme } from "next-themes";
import { PrismLight as SyntaxHighlighter } from "react-syntax-highlighter";
import ts from "react-syntax-highlighter/dist/esm/languages/prism/typescript";
import js from "react-syntax-highlighter/dist/esm/languages/prism/javascript";
import jsx from "react-syntax-highlighter/dist/esm/languages/prism/jsx";
import tsx from "react-syntax-highlighter/dist/esm/languages/prism/tsx";
import json from "react-syntax-highlighter/dist/esm/languages/prism/json";
import python from "react-syntax-highlighter/dist/esm/languages/prism/python";
import bash from "react-syntax-highlighter/dist/esm/languages/prism/bash";
import shell from "react-syntax-highlighter/dist/esm/languages/prism/shell-session";
import css from "react-syntax-highlighter/dist/esm/languages/prism/css";
import html from "react-syntax-highlighter/dist/esm/languages/prism/markup";
import yaml from "react-syntax-highlighter/dist/esm/languages/prism/yaml";
import go from "react-syntax-highlighter/dist/esm/languages/prism/go";
import rust from "react-syntax-highlighter/dist/esm/languages/prism/rust";
import markdown from "react-syntax-highlighter/dist/esm/languages/prism/markdown";
import oneLight from "react-syntax-highlighter/dist/esm/styles/prism/one-light";
import oneDark from "react-syntax-highlighter/dist/esm/styles/prism/one-dark";

SyntaxHighlighter.registerLanguage("typescript", ts);
SyntaxHighlighter.registerLanguage("ts", ts);
SyntaxHighlighter.registerLanguage("javascript", js);
SyntaxHighlighter.registerLanguage("js", js);
SyntaxHighlighter.registerLanguage("jsx", jsx);
SyntaxHighlighter.registerLanguage("tsx", tsx);
SyntaxHighlighter.registerLanguage("json", json);
SyntaxHighlighter.registerLanguage("python", python);
SyntaxHighlighter.registerLanguage("py", python);
SyntaxHighlighter.registerLanguage("bash", bash);
SyntaxHighlighter.registerLanguage("sh", bash);
SyntaxHighlighter.registerLanguage("shell", shell);
SyntaxHighlighter.registerLanguage("css", css);
SyntaxHighlighter.registerLanguage("html", html);
SyntaxHighlighter.registerLanguage("yaml", yaml);
SyntaxHighlighter.registerLanguage("yml", yaml);
SyntaxHighlighter.registerLanguage("go", go);
SyntaxHighlighter.registerLanguage("rust", rust);
SyntaxHighlighter.registerLanguage("markdown", markdown);
SyntaxHighlighter.registerLanguage("md", markdown);

interface CodeHighlighterProps {
  content: string;
  language?: string;
  className?: string;
}

const MAX_HIGHLIGHT_CHARS = 100_000;

export function CodeHighlighter({
  content,
  language = "text",
  className,
}: CodeHighlighterProps) {
  const { resolvedTheme } = useTheme();
  const theme = resolvedTheme === "dark" ? oneDark : oneLight;

  // Fall back to plain pre for very large files to avoid blocking the UI.
  if (content.length > MAX_HIGHLIGHT_CHARS) {
    return (
      <pre
        className={`text-sm font-mono text-body whitespace-pre-wrap ${className ?? ""}`}
      >
        {content}
      </pre>
    );
  }

  return (
    <SyntaxHighlighter
      language={language || "text"}
      style={theme}
      wrapLines={false}
      customStyle={{
        margin: 0,
        padding: 0,
        background: "transparent",
        whiteSpace: "pre-wrap",
        wordBreak: "break-word",
      }}
      codeTagProps={{
        style: {
          fontFamily:
            'ui-monospace, SFMono-Regular, Menlo, Monaco, "JetBrains Mono", "Noto Sans Mono CJK SC", "PingFang SC", "Microsoft YaHei", monospace',
          whiteSpace: "pre-wrap",
          wordBreak: "break-word",
        },
      }}
      className={className}
    >
      {content}
    </SyntaxHighlighter>
  );
}
