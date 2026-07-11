"use client";

import { useState, useMemo } from "react";
import { motion } from "framer-motion";
import { ChevronDown, X, RefreshCw, Copy, MoreHorizontal } from "../Icons";
import { useLanguage } from "../LanguageProvider";
import { Logo } from "../Logo";

export function SaasWorkspaceMockup() {
  const { t } = useLanguage();
  const s = t.saasWorkspace;

  const files = useMemo(
    (): Record<string, { type: "md"; name: string; content: string }> => ({
      "docs/prd.md": {
        type: "md",
        name: s.filesContent.prd.name,
        content: s.filesContent.prd.content,
      },
      "docs/competitive-analysis.md": {
        type: "md",
        name: s.filesContent.analysis.name,
        content: s.filesContent.analysis.content,
      },
    }),
    [s]
  );

  const [docsOpen, setDocsOpen] = useState(true);
  const [openTabs, setOpenTabs] = useState<string[]>([
    "docs/competitive-analysis.md",
    "docs/prd.md",
  ]);
  const [activeTab, setActiveTab] = useState<string | null>(
    "docs/competitive-analysis.md"
  );
  const [view, setView] = useState<"editor" | "preview">("preview");

  const activeFile = activeTab ? files[activeTab] : null;

  const openFile = (path: string) => {
    setOpenTabs((tabs) => (tabs.includes(path) ? tabs : [...tabs, path]));
    setActiveTab(path);
  };

  const closeTab = (path: string) => {
    setOpenTabs((tabs) => {
      const idx = tabs.indexOf(path);
      const next = tabs.filter((p) => p !== path);
      if (activeTab === path) {
        const nextActive = next[idx] ?? next[idx - 1] ?? next[0] ?? null;
        setActiveTab(nextActive);
      }
      return next;
    });
  };

  return (
    <div className="mx-auto w-full max-w-6xl overflow-hidden rounded-xl border border-hairline bg-surface shadow-2xl">
      {/* App header — mac traffic lights only */}
      <div className="flex items-center gap-2 border-b border-hairline bg-surface-soft px-4 py-3">
        <span className="h-3 w-3 rounded-full bg-[#ff5f57] ring-1 ring-inset ring-black/5" />
        <span className="h-3 w-3 rounded-full bg-[#febc2e] ring-1 ring-inset ring-black/5" />
        <span className="h-3 w-3 rounded-full bg-[#28c840] ring-1 ring-inset ring-black/5" />
      </div>

      <div className="flex h-[560px] flex-col sm:flex-row">
        {/* Files pane */}
        <div className="flex w-full flex-col border-b border-hairline bg-surface sm:w-56 sm:border-b-0 sm:border-r">
          <div className="border-b border-hairline px-3 py-2 text-[10px] font-bold uppercase tracking-wider text-muted">
            {s.files}
          </div>
          <div className="flex-1 overflow-auto p-3">
            <button
              onClick={() => setDocsOpen((v) => !v)}
              className="flex w-full items-center gap-1.5 text-left text-sm font-semibold text-foreground transition hover:text-accent"
            >
              <ChevronDown
                className={`h-3.5 w-3.5 text-muted transition ${
                  docsOpen ? "" : "-rotate-90"
                }`}
              />
              <FolderIcon />
              {s.docs}
            </button>

            <motion.div
              initial={false}
              animate={{
                height: docsOpen ? "auto" : 0,
                opacity: docsOpen ? 1 : 0,
              }}
              className="overflow-hidden"
            >
              <div className="mt-1 space-y-0.5 border-l border-hairline-soft pl-5">
                <FileButton
                  name={files["docs/prd.md"].name}
                  active={activeTab === "docs/prd.md"}
                  onClick={() => openFile("docs/prd.md")}
                />
                <FileButton
                  name={files["docs/competitive-analysis.md"].name}
                  active={activeTab === "docs/competitive-analysis.md"}
                  onClick={() => openFile("docs/competitive-analysis.md")}
                />
              </div>
            </motion.div>
          </div>
        </div>

        {/* Editor / Preview pane */}
        <div className="flex flex-1 flex-col border-b border-hairline bg-surface sm:border-b-0 sm:border-r">
          {/* IDE-style tabs */}
          <div className="flex items-center justify-between border-b border-hairline bg-surface-soft">
            <div className="flex flex-1 items-end overflow-x-auto">
              {openTabs.map((path) => {
                const file = files[path];
                const isActive = activeTab === path;
                return (
                  <button
                    key={path}
                    onClick={() => setActiveTab(path)}
                    className={`group relative flex min-w-0 max-w-[160px] items-center gap-2 border-r border-hairline px-3 py-2 text-xs transition ${
                      isActive
                        ? "bg-surface font-semibold text-foreground"
                        : "bg-surface-soft text-body hover:bg-surface"
                    }`}
                  >
                    <FileIcon className="h-3.5 w-3.5 flex-shrink-0 text-muted" />
                    <span className="truncate">{file?.name ?? path}</span>
                    <span
                      onClick={(e) => {
                        e.stopPropagation();
                        closeTab(path);
                      }}
                      className={`ml-1 flex h-4 w-4 flex-shrink-0 items-center justify-center rounded transition ${
                        isActive
                          ? "text-muted hover:bg-hairline hover:text-foreground"
                          : "text-muted/60 opacity-0 group-hover:opacity-100 hover:bg-hairline hover:text-foreground"
                      }`}
                    >
                      <X className="h-3 w-3" />
                    </span>
                    {isActive && (
                      <span className="absolute bottom-0 left-0 right-0 h-[1px] bg-surface" />
                    )}
                  </button>
                );
              })}
            </div>

            {/* Editor / Preview icon toggle */}
            <button
              onClick={() => setView((v) => (v === "editor" ? "preview" : "editor"))}
              className="flex-shrink-0 p-2 text-muted transition hover:text-foreground"
              title={view === "editor" ? s.preview : s.editor}
            >
              {view === "editor" ? (
                <EyeIcon className="h-4 w-4" />
              ) : (
                <CodeIcon className="h-4 w-4" />
              )}
            </button>
          </div>

          <div className="flex-1 overflow-auto bg-surface-doc">
            {activeFile ? (
              view === "editor" ? (
                <MarkdownEditor content={activeFile.content} />
              ) : (
                <div className="p-5">
                  <MarkdownPreview content={activeFile.content} />
                </div>
              )
            ) : (
              <div className="flex h-full flex-col items-center justify-center gap-3 text-muted">
                <Logo className="h-16 w-16 opacity-20 grayscale" />
                <p className="text-sm">{s.empty}</p>
              </div>
            )}
          </div>
        </div>

        {/* Chat pane */}
        <div className="flex flex-1 flex-col bg-surface">
          <div className="border-b border-hairline px-4 py-2 text-[10px] font-bold uppercase tracking-wider text-muted">
            {s.chat}
          </div>

          <div className="flex-1 space-y-5 overflow-auto p-4 text-sm">
            <motion.div
              initial={{ opacity: 0, y: 12 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: 0.2 }}
              className="text-foreground"
            >
              <span className="mb-1 block text-[10px] font-bold uppercase tracking-wider text-muted">
                You
              </span>
              {s.userMessage}
            </motion.div>

            <motion.div
              initial={{ opacity: 0, y: 12 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: 0.5 }}
              className="space-y-2"
            >
              <span className="block text-[10px] font-bold uppercase tracking-wider text-muted">
                Agent
              </span>
              <StepDot label={s.stepThinking} />
              <StepDot label={s.stepSearch} />
              <StepDot label={s.stepWrite} />
            </motion.div>

            <motion.div
              initial={{ opacity: 0, y: 12 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: 2.1 }}
              className="text-body"
            >
              <p className="text-sm leading-relaxed">
                {s.assistantReply}{" "}
                <span className="font-mono font-semibold text-foreground">
                  {files["docs/competitive-analysis.md"].name}
                </span>
                {s.assistantReplySuffix}
              </p>
              <MessageActions />
            </motion.div>
          </div>

          <div className="px-4 py-3">
            <div className="flex items-center gap-2 rounded-full border border-hairline bg-surface px-3 py-2">
              <span className="text-sm text-muted">{s.inputPlaceholder}</span>
              <button
                type="button"
                className="ml-auto flex h-7 w-7 items-center justify-center rounded-full bg-accent text-on-accent transition hover:opacity-90"
              >
                <ArrowUpIcon className="h-4 w-4" />
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

function FileButton({
  name,
  active,
  onClick,
}: {
  name: string;
  active: boolean;
  onClick: () => void;
}) {
  return (
    <button
      onClick={onClick}
      className={`flex w-full items-center gap-1.5 rounded-md px-2 py-1 text-left text-xs transition ${
        active
          ? "bg-accent/10 font-semibold text-foreground"
          : "text-body hover:bg-surface-soft hover:text-foreground"
      }`}
    >
      <FileIcon className="h-3.5 w-3.5 text-muted" />
      {name}
    </button>
  );
}

function StepDot({ label }: { label: string }) {
  return (
    <div className="flex items-center gap-2 text-xs text-muted">
      <span className="h-1.5 w-1.5 rounded-full bg-accent" />
      {label}
    </div>
  );
}

function FolderIcon() {
  return (
    <svg
      className="h-4 w-4 text-accent"
      fill="currentColor"
      viewBox="0 0 24 24"
    >
      <path d="M4 20h16a2 2 0 0 0 2-2V8a2 2 0 0 0-2-2h-7.93a2 2 0 0 1-1.66-.9l-.82-1.2A2 2 0 0 0 7.93 2H4a2 2 0 0 0-2 2v13a2 2 0 0 0 2 2Z" />
    </svg>
  );
}

function FileIcon({ className }: { className?: string }) {
  return (
    <svg
      className={className}
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      viewBox="0 0 24 24"
    >
      <path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z" />
      <polyline points="14 2 14 8 20 8" />
    </svg>
  );
}

function CodeIcon({ className }: { className?: string }) {
  return (
    <svg
      className={className}
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      viewBox="0 0 24 24"
    >
      <polyline points="16 18 22 12 16 6" />
      <polyline points="8 6 2 12 8 18" />
    </svg>
  );
}

function EyeIcon({ className }: { className?: string }) {
  return (
    <svg
      className={className}
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      viewBox="0 0 24 24"
    >
      <path d="M2 12s3-7 10-7 10 7 10 7-3 7-10 7-10-7-10-7Z" />
      <circle cx="12" cy="12" r="3" />
    </svg>
  );
}

function ArrowUpIcon({ className }: { className?: string }) {
  return (
    <svg
      className={className}
      fill="none"
      stroke="currentColor"
      strokeWidth="2.5"
      strokeLinecap="round"
      strokeLinejoin="round"
      viewBox="0 0 24 24"
    >
      <path d="M12 19V5" />
      <path d="m5 12 7-7 7 7" />
    </svg>
  );
}

function MarkdownEditor({ content }: { content: string }) {
  const lines = content.split("\n");
  return (
    <div className="flex min-h-full font-mono text-xs leading-6">
      <div className="select-none border-r border-hairline bg-surface-soft px-3 py-4 text-right text-muted">
        {lines.map((_, i) => (
          <div key={i}>{i + 1}</div>
        ))}
      </div>
      <div className="flex-1 whitespace-pre-wrap p-4 text-body">{content}</div>
    </div>
  );
}

function MarkdownPreview({ content }: { content: string }) {
  return (
    <div className="prose prose-sm max-w-none text-body">
      {content.split("\n").map((line, i) => {
        const trimmed = line.trim();
        if (!trimmed) return <div key={i} className="h-3" />;
        if (trimmed.startsWith("# ")) {
          return (
            <h1 key={i} className="mb-2 mt-4 text-xl font-bold text-foreground">
              {trimmed.slice(2)}
            </h1>
          );
        }
        if (trimmed.startsWith("## ")) {
          return (
            <h2 key={i} className="mb-2 mt-4 text-base font-bold text-foreground">
              {trimmed.slice(3)}
            </h2>
          );
        }
        if (trimmed.startsWith("- ")) {
          return (
            <li key={i} className="ml-4 list-disc text-sm">
              {trimmed.slice(2)}
            </li>
          );
        }
        if (/^\d+\.\s/.test(trimmed)) {
          return (
            <li key={i} className="ml-4 list-decimal text-sm">
              {trimmed.replace(/^\d+\.\s/, "")}
            </li>
          );
        }
        return (
          <p key={i} className="text-sm leading-relaxed">
            {trimmed}
          </p>
        );
      })}
    </div>
  );
}

function MessageActions() {
  return (
    <div className="mt-2 flex items-center gap-1">
      <button
        type="button"
        className="flex h-7 w-7 items-center justify-center rounded-md text-muted transition hover:bg-surface-soft hover:text-foreground"
        aria-label="Regenerate"
      >
        <RefreshCw className="h-3.5 w-3.5" />
      </button>
      <button
        type="button"
        className="flex h-7 w-7 items-center justify-center rounded-md text-muted transition hover:bg-surface-soft hover:text-foreground"
        aria-label="Copy"
      >
        <Copy className="h-3.5 w-3.5" />
      </button>
      <button
        type="button"
        className="flex h-7 w-7 items-center justify-center rounded-md text-muted transition hover:bg-surface-soft hover:text-foreground"
        aria-label="More"
      >
        <MoreHorizontal className="h-3.5 w-3.5" />
      </button>
    </div>
  );
}
