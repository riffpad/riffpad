"use client";

import { useState } from "react";
import { useTheme } from "next-themes";
import { Check, Copy } from "lucide-react";
import ReactMarkdown from "react-markdown";
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
import oneLight from "react-syntax-highlighter/dist/esm/styles/prism/one-light";
import oneDark from "react-syntax-highlighter/dist/esm/styles/prism/one-dark";
import remarkGfm from "remark-gfm";

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

export interface Citation {
  index: number;
  url: string;
  title: string;
}

interface MarkdownRendererProps {
  content: string;
  citations?: Citation[];
}

function applyCitations(content: string, citations: Citation[]): string {
  const maxIndex = citations.length;
  if (maxIndex === 0) return content;

  const validSet = new Set(citations.map((c) => c.index));
  const urlFor = (idx: number) => citations.find((c) => c.index === idx)?.url;

  // Protect URLs and code blocks so we don't accidentally turn digits inside
  // them into citation links.
  type Protected = { key: string; value: string };
  const protectedParts: Protected[] = [];
  let counter = 0;
  const protect = (value: string) => {
    const key = `__PROTECTED_${counter++}__`;
    protectedParts.push({ key, value });
    return key;
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
  };

  let text = content
    // Code blocks (fenced)
    .replace(/```[\s\S]*?```/g, protect)
    // Inline code
    .replace(/`[^`]*`/g, protect)
    // Markdown links and bare URLs
    .replace(/\[([^\]]*)\]\(([^)]+)\)/g, protect)
    .replace(/https?:\/\/[^\s\])]+/g, protect);

  // 1. Explicit bracketed citations [1], [2].
  text = text.replace(/\[(\d+)\]/g, (_, num) => {
    const idx = Number(num);
    const url = urlFor(idx);
    if (!url) return `[${num}]`;
    return `[${num}](${url})`;
  });

  // 2. Concatenated citations like "23" meaning [2] and [3]. Only when both
  //    digits are valid indices and there is no third digit.
  const concatRe = new RegExp(`(?<!\\d)([1-${maxIndex}])([1-${maxIndex}])(?!\\d)`, "g");
  text = text.replace(concatRe, (_, d1, d2) => {
    const idx1 = Number(d1);
    const idx2 = Number(d2);
    const url1 = urlFor(idx1);
    const url2 = urlFor(idx2);
    if (!url1 || !url2) return `${d1}${d2}`;
    return `[${d1}](${url1})[${d2}](${url2})`;
  });

  // 3. Standalone single-digit citations surrounded by word boundaries or
  //    punctuation. Avoid digits that are part of a word or inside links.
  const singleRe = new RegExp(
    `(?<!\\d)(?<!\\[)\\b([1-${maxIndex}])\\b(?!\\d)(?!\\])`,
    "g"
  );
  text = text.replace(singleRe, (match, num) => {
    const idx = Number(num);
    const url = urlFor(idx);
    if (!url) return match;
    return `[${num}](${url})`;
  });

  // Restore protected parts in reverse order so earlier keys don't mask later.
  for (let i = protectedParts.length - 1; i >= 0; i--) {
    const { key, value } = protectedParts[i];
    text = text.replace(key, () => value);
  }

  return text;
}

export function MarkdownRenderer({ content, citations }: MarkdownRendererProps) {
  const processed = citations?.length ? applyCitations(content, citations) : content;

  return (
    <ReactMarkdown
      remarkPlugins={[remarkGfm]}
      components={{
        h1: ({ children }) => (
          <h1 className="mb-2 mt-3 text-base font-bold text-ink">{children}</h1>
        ),
        h2: ({ children }) => (
          <h2 className="mb-2 mt-3 text-sm font-bold text-ink">{children}</h2>
        ),
        h3: ({ children }) => (
          <h3 className="mb-1 mt-2 text-sm font-semibold text-ink">{children}</h3>
        ),
        p: ({ children }) => (
          <p className="mb-2 text-sm leading-relaxed text-body last:mb-0">
            {children}
          </p>
        ),
        ul: ({ children }) => (
          <ul className="mb-2 ml-4 list-disc text-sm leading-relaxed text-body marker:text-mute">
            {children}
          </ul>
        ),
        ol: ({ children }) => (
          <ol className="mb-2 ml-4 list-decimal text-sm leading-relaxed text-body marker:text-mute">
            {children}
          </ol>
        ),
        li: ({ children }) => <li className="mb-0.5">{children}</li>,
        a: ({ children, href }) => (
          <a
            href={href}
            target="_blank"
            rel="noopener noreferrer"
            className="font-medium text-primary underline underline-offset-2 hover:text-primary-pressed"
          >
            {children}
          </a>
        ),
        code: Code,
        pre: ({ children }) => <>{children}</>,
        blockquote: ({ children }) => (
          <blockquote className="mb-2 border-l-2 border-hairline pl-3 italic text-mute">
            {children}
          </blockquote>
        ),
        hr: () => <hr className="my-3 border-hairline-soft" />,
        strong: ({ children }) => (
          <strong className="font-semibold text-ink">{children}</strong>
        ),
        em: ({ children }) => <em className="italic text-body">{children}</em>,
        table: ({ children }) => (
          <div className="my-2 overflow-x-auto">
            <table className="w-full border-collapse text-xs">{children}</table>
          </div>
        ),
        thead: ({ children }) => (
          <thead className="bg-card-soft text-ink">{children}</thead>
        ),
        th: ({ children }) => (
          <th className="border border-hairline px-2 py-1.5 text-left font-semibold">
            {children}
          </th>
        ),
        td: ({ children }) => (
          <td className="border border-hairline px-2 py-1.5 text-body">
            {children}
          </td>
        ),
      }}
    >
      {processed}
    </ReactMarkdown>
  );
}

function CopyButton({ text }: { text: string }) {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(text);
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    } catch {
      // ignore
    }
  };

  return (
    <button
      onClick={handleCopy}
      className="flex h-6 w-6 items-center justify-center rounded text-ash transition hover:bg-card hover:text-ink"
      aria-label={copied ? "Copied" : "Copy"}
      title={copied ? "Copied" : "Copy"}
    >
      {copied ? (
        <Check className="h-3 w-3" />
      ) : (
        <Copy className="h-3 w-3" />
      )}
    </button>
  );
}

function Code({
  children,
  className,
}: {
  children?: React.ReactNode;
  className?: string;
}) {
  const { resolvedTheme } = useTheme();
  const isInline = !className;
  const raw = Array.isArray(children) ? children.join("") : String(children ?? "");

  if (isInline) {
    return (
      <code className="rounded bg-card-soft px-1 py-0.5 text-xs font-mono text-ink">
        {children}
      </code>
    );
  }

  const match = /language-(\w+)/.exec(className ?? "");
  const language = match ? match[1] : "";
  const theme = resolvedTheme === "dark" ? oneDark : oneLight;

  return (
    <div className="my-2 overflow-hidden rounded-md border border-hairline bg-card">
      <div className="flex items-center justify-between border-b border-hairline bg-card-soft px-3 py-1.5">
        <span className="text-[10px] font-semibold uppercase tracking-wider text-mute">
          {language || "code"}
        </span>
        <CopyButton text={raw} />
      </div>
      <div className="max-h-96 overflow-auto">
        <SyntaxHighlighter
          language={language || "text"}
          style={theme}
          wrapLines={false}
          customStyle={{
            margin: 0,
            padding: "0.75rem",
            fontSize: "0.75rem",
            lineHeight: "1.6",
            background: "transparent",
            whiteSpace: "pre-wrap",
            wordBreak: "break-word",
          }}
          codeTagProps={{
            style: {
              fontFamily:
                'ui-monospace, SFMono-Regular, Menlo, Monaco, "JetBrains Mono", monospace',
              whiteSpace: "pre-wrap",
              wordBreak: "break-word",
            },
          }}
        >
          {raw}
        </SyntaxHighlighter>
      </div>
    </div>
  );
}
