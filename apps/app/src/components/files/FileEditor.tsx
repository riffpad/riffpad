"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { Code, Eye, FileCode, X } from "lucide-react";
import { ScrollArea } from "@/components/ui/scroll-area";
import { CodeHighlighter } from "@/components/ui/CodeHighlighter";
import { MarkdownRenderer } from "@/components/chat/MarkdownRenderer";

const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

interface FileEditorProps {
  workspaceId: string;
  selectedFile: string | null;
  onSelectFile: (path: string | null) => void;
  emptyHint?: string;
}

type ViewMode = "raw" | "preview";

function isMarkdown(path: string) {
  return /\.(md|mdx|markdown)$/i.test(path);
}

function fileName(path: string) {
  return path.split("/").pop() ?? path;
}

const LANG_MAP: Record<string, string> = {
  ts: "typescript",
  tsx: "tsx",
  js: "javascript",
  jsx: "jsx",
  json: "json",
  py: "python",
  go: "go",
  rs: "rust",
  css: "css",
  html: "html",
  htm: "html",
  md: "markdown",
  mdx: "markdown",
  yml: "yaml",
  yaml: "yaml",
  sh: "bash",
  bash: "bash",
};

function languageFromPath(path: string) {
  const ext = path.split(".").pop()?.toLowerCase();
  return ext ? LANG_MAP[ext] ?? ext : "text";
}

export function FileEditor({
  workspaceId,
  selectedFile,
  onSelectFile,
  emptyHint,
}: FileEditorProps) {
  const [openFiles, setOpenFiles] = useState<string[]>([]);
  const [activeFile, setActiveFile] = useState<string | null>(null);
  const [contents, setContents] = useState<Record<string, string>>({});
  const [viewModes, setViewModes] = useState<Record<string, ViewMode>>({});
  const loadedRef = useRef<Set<string>>(new Set());

  const fetchContent = useCallback(async (path: string) => {
    if (loadedRef.current.has(path)) return;
    loadedRef.current.add(path);
    try {
      const res = await fetch(
        `${API_URL}/api/v1/workspaces/${workspaceId}/file?path=${encodeURIComponent(path)}`
      );
      if (res.ok) {
        const data = await res.json();
        setContents((prev) => ({ ...prev, [path]: data.content ?? "" }));
      }
    } catch (err) {
      console.error("Failed to fetch file:", err);
    }
  }, [workspaceId]);

  const openFile = useCallback((path: string) => {
    setOpenFiles((prev) => (prev.includes(path) ? prev : [...prev, path]));
    setActiveFile(path);
    onSelectFile(path);
    if (isMarkdown(path)) {
      setViewModes((prev) => ({
        ...prev,
        [path]: prev[path] ?? "preview",
      }));
    }
  }, [onSelectFile]);

  // Open and activate the file selected from the file tree.
  useEffect(() => {
    if (selectedFile && selectedFile !== activeFile) {
      openFile(selectedFile);
    }
  }, [selectedFile, activeFile, openFile]);

  const closeFile = useCallback((path: string) => {
    setOpenFiles((prev) => {
      const idx = prev.indexOf(path);
      const next = prev.filter((p) => p !== path);
      if (activeFile === path) {
        const nextActive = next[idx] ?? next[idx - 1] ?? next[0] ?? null;
        setActiveFile(nextActive);
        onSelectFile(nextActive);
      }
      return next;
    });
  }, [activeFile, onSelectFile]);

  const toggleView = useCallback((path: string) => {
    setViewModes((prev) => ({
      ...prev,
      [path]: prev[path] === "raw" ? "preview" : "raw",
    }));
  }, []);

  useEffect(() => {
    if (activeFile) {
      void fetchContent(activeFile);
    }
  }, [activeFile, fetchContent]);

  const activeContent = activeFile ? contents[activeFile] ?? "" : "";
  const activeView = activeFile ? viewModes[activeFile] ?? "raw" : "raw";

  return (
    <div className="flex flex-col h-full min-h-0 bg-canvas">
      {/* IDE-style tabs */}
      <div className="flex items-center justify-between border-b border-hairline bg-card-soft/70 backdrop-blur">
        <div className="flex flex-1 items-end overflow-x-auto scrollbar-none">
          {openFiles.map((path) => {
            const isActive = activeFile === path;
            return (
              <button
                key={path}
                onClick={() => openFile(path)}
                className={`group relative flex min-w-0 max-w-[160px] items-center gap-1.5 border-r border-hairline px-3 py-2 text-xs transition ${
                  isActive
                    ? "bg-canvas font-semibold text-ink"
                    : "bg-card-soft/50 text-body hover:bg-card-soft hover:text-ink"
                }`}
              >
                <FileCode className="h-3.5 w-3.5 shrink-0 text-mute" />
                <span className="truncate">{fileName(path)}</span>
                <span
                  onClick={(e) => {
                    e.stopPropagation();
                    closeFile(path);
                  }}
                  className={`ml-1 flex h-4 w-4 shrink-0 items-center justify-center rounded transition ${
                    isActive
                      ? "text-mute hover:bg-hairline hover:text-ink"
                      : "text-mute/60 opacity-0 group-hover:opacity-100 hover:bg-hairline hover:text-ink"
                  }`}
                  aria-label="Close tab"
                >
                  <X className="h-3 w-3" />
                </span>
                {isActive && (
                  <span className="absolute top-0 left-0 right-0 h-[2px] bg-primary" />
                )}
              </button>
            );
          })}
        </div>

        {activeFile && isMarkdown(activeFile) && (
          <button
            onClick={() => toggleView(activeFile)}
            className="flex shrink-0 items-center gap-1 px-3 py-2 text-xs font-medium text-mute transition hover:text-ink"
            title={activeView === "raw" ? "Preview" : "Raw"}
            aria-label={activeView === "raw" ? "Preview" : "Raw"}
          >
            {activeView === "raw" ? (
              <>
                <Eye className="h-3.5 w-3.5" />
                <span className="hidden sm:inline">Preview</span>
              </>
            ) : (
              <>
                <Code className="h-3.5 w-3.5" />
                <span className="hidden sm:inline">Raw</span>
              </>
            )}
          </button>
        )}
      </div>

      {/* Editor / Preview content */}
      <div data-testid="file-editor-content" className="flex-1 min-h-0 overflow-hidden bg-card-doc/30">
        {activeFile ? (
          <ScrollArea className="h-full">
            {isMarkdown(activeFile) && activeView === "preview" ? (
              <div className="p-4">
                <MarkdownRenderer content={activeContent} />
              </div>
            ) : (
              <div className="p-4 text-sm leading-relaxed">
                <CodeHighlighter
                  content={activeContent}
                  language={languageFromPath(activeFile)}
                />
              </div>
            )}
          </ScrollArea>
        ) : (
          <div className="h-full flex flex-col items-center justify-center text-mute text-sm">
            <FileCode className="h-8 w-8 mb-2 opacity-40" />
            <p>{emptyHint ?? "Select a file to view"}</p>
          </div>
        )}
      </div>
    </div>
  );
}
