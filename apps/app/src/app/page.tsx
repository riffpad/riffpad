"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { flushSync } from "react-dom";
import Image from "next/image";
import { useTheme } from "next-themes";
import {
  ChevronDown,
  Code,
  Eye,
  FileCode,
  Folder,
  Languages,
  MessageSquare,
  Moon,
  PanelLeft,
  Send,
  Sun,
  X,
} from "lucide-react";

import { Button } from "@/components/ui/button";
import { CardHeader, CardTitle } from "@/components/ui/card";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Textarea } from "@/components/ui/textarea";
import { ChatPanel } from "@/components/chat/ChatPanel";
import type { ChatItem } from "@/components/chat/types";
import { useI18n } from "@/lib/i18n";

const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

interface AgentMessage {
  role: "user" | "assistant" | "tool";
  content?: string;
  tool_calls?: { id: string; function: { name: string; arguments: string } }[];
  timestamp: number;
}

interface AgentEvent {
  type: string;
  content?: string;
  delta?: string;
  name?: string;
  args?: unknown;
  result?: unknown;
  path?: string;
  toolCallId?: string;
  toolName?: string;
  isError?: boolean;
  message?: AgentMessage;
  timestamp: number;
}

interface FileInfo {
  name: string;
  path: string;
  isDir: boolean;
  size: number;
}

export default function Home() {
  const { t, locale, setLocale } = useI18n();
  const { setTheme, resolvedTheme } = useTheme();
  const [mounted, setMounted] = useState(false);

  const [prompt, setPrompt] = useState("");
  const [workspaceId, setWorkspaceId] = useState<string | null>(null);
  const [workspaceSlug, setWorkspaceSlug] = useState<string | null>(null);
  const [chatItems, setChatItems] = useState<ChatItem[]>([]);
  const [connected, setConnected] = useState(false);
  const [files, setFiles] = useState<FileInfo[]>([]);
  const [selectedFile, setSelectedFile] = useState<string | null>(null);
  const [fileContent, setFileContent] = useState<string>("");
  const [activeTab, setActiveTab] = useState<"preview" | "code">("preview");
  const [filesOpen, setFilesOpen] = useState(false);
  const [langOpen, setLangOpen] = useState(false);

  const wsRef = useRef<WebSocket | null>(null);
  const chatEndRef = useRef<HTMLDivElement>(null);
  const langRef = useRef<HTMLDivElement>(null);
  const chatIdRef = useRef(0);
  const assistantIdRef = useRef<string | null>(null);

  useEffect(() => {
    setMounted(true);
  }, []);

  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (langRef.current && !langRef.current.contains(e.target as Node)) {
        setLangOpen(false);
      }
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, []);

  useEffect(() => {
    if (filesOpen) {
      document.body.style.overflow = "hidden";
    } else {
      document.body.style.overflow = "";
    }
    return () => {
      document.body.style.overflow = "";
    };
  }, [filesOpen]);

  const fetchFiles = useCallback(async (id: string) => {
    try {
      const res = await fetch(`${API_URL}/api/v1/workspaces/${id}/files`);
      if (res.ok) {
        const data = await res.json();
        setFiles(data ?? []);
      }
    } catch (err) {
      console.error("Failed to fetch files:", err);
    }
  }, []);

  const fetchFile = useCallback(async (id: string, path: string) => {
    try {
      const res = await fetch(
        `${API_URL}/api/v1/workspaces/${id}/file?path=${encodeURIComponent(path)}`
      );
      if (res.ok) {
        const data = await res.json();
        setFileContent(data.content);
      }
    } catch (err) {
      console.error("Failed to fetch file:", err);
    }
  }, []);

  const connect = useCallback(
    (id: string) => {
      const wsUrl = API_URL.replace(/^http/, "ws");
      const ws = new WebSocket(`${wsUrl}/ws/workspaces/${id}`);
      wsRef.current = ws;

      ws.onopen = () => setConnected(true);
      ws.onclose = () => setConnected(false);
      ws.onerror = () => setConnected(false);
      ws.onmessage = (msg) => {
        try {
          const event: AgentEvent = JSON.parse(msg.data);
          const makeId = () => `${++chatIdRef.current}`;

          switch (event.type) {
            case "agent_start": {
              assistantIdRef.current = makeId();
              setChatItems((prev) => [
                ...prev,
                {
                  type: "assistant",
                  id: assistantIdRef.current!,
                  content: "",
                  isStreaming: true,
                  timestamp: event.timestamp,
                },
              ]);
              break;
            }
            case "message_start": {
              if (!assistantIdRef.current) {
                assistantIdRef.current = makeId();
                setChatItems((prev) => [
                  ...prev,
                  {
                    type: "assistant",
                    id: assistantIdRef.current!,
                    content: "",
                    isStreaming: true,
                    timestamp: event.timestamp,
                  },
                ]);
              }
              break;
            }
            case "message_delta": {
              if (event.delta && assistantIdRef.current) {
                setChatItems((prev) =>
                  prev.map((item) =>
                    item.id === assistantIdRef.current && item.type === "assistant"
                      ? { ...item, content: item.content + event.delta! }
                      : item
                  )
                );
              }
              break;
            }
            case "reasoning_delta": {
              if (event.delta && assistantIdRef.current) {
                setChatItems((prev) =>
                  prev.map((item) =>
                    item.id === assistantIdRef.current && item.type === "assistant"
                      ? { ...item, reasoning: (item.reasoning ?? "") + event.delta! }
                      : item
                  )
                );
              }
              break;
            }
            case "message_end": {
              if (event.message?.role === "assistant" && assistantIdRef.current) {
                setChatItems((prev) =>
                  prev.map((item) =>
                    item.id === assistantIdRef.current && item.type === "assistant"
                      ? { ...item, content: event.message?.content ?? "", isStreaming: false }
                      : item
                  )
                );
                assistantIdRef.current = null;
              }
              break;
            }
            case "tool_execution_start": {
              flushSync(() => {
                setChatItems((prev) => [
                  ...prev,
                  {
                    type: "tool",
                    id: `tool-${event.toolCallId ?? makeId()}`,
                    toolName: event.toolName ?? "tool",
                    args: event.args,
                    isPartial: true,
                    timestamp: event.timestamp,
                  },
                ]);
              });
              break;
            }
            case "tool_execution_end": {
              setChatItems((prev) =>
                prev.map((item) =>
                  item.type === "tool" && item.id === `tool-${event.toolCallId}`
                    ? {
                        ...item,
                        isPartial: false,
                        result: event.result,
                        isError: event.isError,
                      }
                    : item
                )
              );
              void fetchFiles(id);
              break;
            }
            case "file_change": {
              if (event.path) {
                setChatItems((prev) => [
                  ...prev,
                  {
                    type: "file",
                    id: makeId(),
                    path: event.path!,
                    timestamp: event.timestamp,
                  },
                ]);
              }
              void fetchFiles(id);
              break;
            }
            case "agent_end": {
              assistantIdRef.current = null;
              break;
            }
            case "error": {
              setChatItems((prev) => [
                ...prev,
                {
                  type: "assistant",
                  id: makeId(),
                  content: event.content ?? "Agent error",
                  timestamp: event.timestamp,
                },
              ]);
              break;
            }
          }
        } catch {
          console.error("Failed to parse event:", msg.data);
        }
      };
    },
    [fetchFiles]
  );

  const createWorkspace = async () => {
    const res = await fetch(`${API_URL}/api/v1/workspaces`, { method: "POST" });
    if (!res.ok) throw new Error("Failed to create workspace");
    const data = await res.json();
    setWorkspaceId(data.id);
    setWorkspaceSlug(data.slug);
    connect(data.id);
    await fetchFiles(data.id);
    return data;
  };

  const handleSubmit = async (e: React.FormEvent | React.KeyboardEvent) => {
    e.preventDefault();
    if (!prompt.trim()) return;

    let id = workspaceId;
    if (!id) {
      const ws = await createWorkspace();
      id = ws.id;
    }

    setChatItems((prev) => [
      ...prev,
      {
        id: `${++chatIdRef.current}`,
        type: "user",
        content: prompt,
        timestamp: Date.now(),
      },
    ]);

    wsRef.current?.send(JSON.stringify({ type: "prompt", content: prompt }));
    setPrompt("");
  };

  useEffect(() => {
    chatEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [chatItems]);

  useEffect(() => {
    return () => {
      wsRef.current?.close();
    };
  }, []);

  const handleFileClick = (file: FileInfo) => {
    if (!workspaceId || file.isDir) return;
    setSelectedFile(file.path);
    setActiveTab("code");
    void fetchFile(workspaceId, file.path);
  };

  const toggleTheme = () => {
    setTheme(resolvedTheme === "dark" ? "light" : "dark");
  };

  const isDark = resolvedTheme === "dark";

  const fileTree = (
    <>
      <CardHeader className="py-3 px-4 flex flex-row items-center justify-between">
        <CardTitle className="text-xs font-bold uppercase tracking-wide text-mute flex items-center gap-2">
          <Folder className="h-4 w-4" />
          {t("files.title")}
        </CardTitle>
        <button
          onClick={() => setFilesOpen(false)}
          className="lg:hidden inline-flex h-8 w-8 items-center justify-center rounded-md text-mute hover:text-ink hover:bg-card-soft"
          aria-label={t("files.close")}
        >
          <X className="h-4 w-4" />
        </button>
      </CardHeader>
      <ScrollArea className="flex-1 px-2">
        {files.length === 0 && (
          <p className="text-xs text-mute px-2 py-1">{t("files.empty")}</p>
        )}
        <ul className="space-y-0.5 pb-4">
          {files.map((file) => (
            <li key={file.path}>
              <button
                onClick={() => handleFileClick(file)}
                className={`w-full flex items-center gap-2 rounded-md px-2 py-1.5 text-sm text-left transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/50 ${
                  selectedFile === file.path
                    ? "bg-canvas text-ink shadow-sm"
                    : "hover:bg-card-soft text-body"
                }`}
              >
                {file.isDir ? (
                  <Folder className="h-4 w-4 text-mute shrink-0" />
                ) : (
                  <FileCode className="h-4 w-4 text-mute shrink-0" />
                )}
                <span className="truncate">{file.name}</span>
              </button>
            </li>
          ))}
        </ul>
      </ScrollArea>
    </>
  );

  const centerTabs = (
    <div className="flex items-center gap-1 border-b border-hairline bg-card/70 backdrop-blur px-2">
      <button
        onClick={() => setActiveTab("preview")}
        className={`flex items-center gap-1.5 px-3 py-2 text-xs font-semibold transition ${
          activeTab === "preview"
            ? "text-ink border-b-2 border-primary"
            : "text-mute hover:text-ink"
        }`}
      >
        <Eye className="h-3.5 w-3.5" />
        {t("preview.title")}
      </button>
      <button
        onClick={() => setActiveTab("code")}
        className={`flex items-center gap-1.5 px-3 py-2 text-xs font-semibold transition ${
          activeTab === "code"
            ? "text-ink border-b-2 border-primary"
            : "text-mute hover:text-ink"
        }`}
      >
        <Code className="h-3.5 w-3.5" />
        {t("code.title")}
      </button>
    </div>
  );

  const previewPane = (
    <div className="flex-1 flex flex-col items-center justify-center p-6 text-center min-h-0">
      <div className="w-16 h-16 mb-4 bg-primary/10 rounded-2xl flex items-center justify-center">
        <Eye className="h-8 w-8 text-primary" />
      </div>
      <h3 className="text-sm font-semibold text-ink mb-1">
        {t("preview.empty")}
      </h3>
      <p className="text-xs text-mute max-w-xs">
        {t("preview.hint")}
      </p>
    </div>
  );

  const codePane = (
    <div className="flex-1 min-h-0 overflow-hidden bg-card-doc/50">
      {selectedFile ? (
        <ScrollArea className="h-full">
          <pre className="p-4 text-sm font-mono text-body whitespace-pre-wrap">
            {fileContent}
          </pre>
        </ScrollArea>
      ) : (
        <div className="h-full flex items-center justify-center text-mute text-sm">
          {t("code.empty")}
        </div>
      )}
    </div>
  );

  return (
    <div className="h-dvh bg-canvas text-body flex flex-col relative overflow-hidden">
      {/* Header */}
      <header className="relative border-b border-hairline bg-card/80 backdrop-blur px-4 lg:px-6 h-14 flex items-center justify-between sticky top-0 z-40">
        <div className="flex items-center gap-3">
          <Image
            src="/logo.png"
            alt="Riffpad"
            width={32}
            height={32}
            className="riffpad-logo rounded-md"
          />
          <div>
            <h1 className="text-[15px] font-bold text-ink leading-tight">
              {t("app.name")}
            </h1>
            <p className="text-[11px] text-mute hidden sm:block">
              {t("app.tagline")}
            </p>
          </div>
        </div>

        <div className="flex items-center gap-1.5">
          {workspaceId ? (
            <div className="hidden sm:flex items-center gap-2 text-xs text-mute mr-2">
              <span className="font-medium text-ink">
                {workspaceSlug || workspaceId}
              </span>
              <span
                className={`inline-block h-2 w-2 rounded-full ${
                  connected ? "bg-accent-green" : "bg-accent-red"
                }`}
                title={connected ? t("workspace.connected") : t("workspace.disconnected")}
              />
            </div>
          ) : null}

          {workspaceId && (
            <Button
              variant="ghost"
              size="sm"
              onClick={() => setFilesOpen(true)}
              className="lg:hidden h-8 px-2 text-xs font-semibold"
            >
              <PanelLeft className="h-4 w-4 mr-1.5" />
              {t("files.title")}
            </Button>
          )}

          {!workspaceId && (
            <Button
              onClick={() => void createWorkspace()}
              size="sm"
              className="bg-primary text-primary-foreground hover:bg-primary-pressed h-8 text-xs font-bold rounded-md"
            >
              {t("workspace.new")}
            </Button>
          )}

          <div ref={langRef} className="relative">
            <Button
              variant="ghost"
              size="sm"
              onClick={() => setLangOpen((v) => !v)}
              className="text-xs font-semibold h-8 px-2 gap-1"
              aria-expanded={langOpen}
              aria-haspopup="listbox"
            >
              <Languages className="h-3.5 w-3.5" />
              <span className="hidden sm:inline">{t(`lang.${locale}`)}</span>
              <ChevronDown
                className={`h-3.5 w-3.5 transition-transform ${
                  langOpen ? "rotate-180" : ""
                }`}
              />
            </Button>
            {langOpen && (
              <div
                className="absolute right-0 top-full z-50 mt-1 min-w-[120px] overflow-hidden rounded-md border border-hairline bg-card shadow-lg"
                role="listbox"
              >
                {(["zh", "en"] as const).map((code) => (
                  <button
                    key={code}
                    onClick={() => {
                      setLocale(code);
                      setLangOpen(false);
                    }}
                    className={`flex min-h-[36px] w-full items-center justify-between px-3 py-2 text-left text-sm transition ${
                      locale === code
                        ? "bg-card-soft font-semibold text-ink"
                        : "text-body hover:bg-card-soft hover:text-ink"
                    }`}
                    role="option"
                    aria-selected={locale === code}
                  >
                    {t(`lang.${code}`)}
                    {locale === code && (
                      <span className="h-1.5 w-1.5 rounded-full bg-primary" />
                    )}
                  </button>
                ))}
              </div>
            )}
          </div>

          <Button
            variant="ghost"
            size="icon"
            onClick={toggleTheme}
            className="h-8 w-8"
            aria-label="toggle theme"
          >
            {mounted && isDark ? (
              <Moon className="h-4 w-4" />
            ) : (
              <Sun className="h-4 w-4" />
            )}
          </Button>
        </div>
      </header>

      {/* IDE layout */}
      {workspaceId ? (
        <div className="relative flex-1 grid grid-cols-1 lg:grid-cols-[260px_1fr_320px] overflow-hidden h-full min-h-0">
          {/* File tree */}
          <aside className="border-r border-hairline bg-card/70 backdrop-blur hidden lg:flex flex-col h-full min-h-0">
            {fileTree}
          </aside>

          {/* Mobile file drawer */}
          {filesOpen && (
            <>
              <div
                className="fixed inset-0 z-40 bg-ink/20 backdrop-blur-sm lg:hidden"
                onClick={() => setFilesOpen(false)}
              />
              <aside className="fixed left-0 top-14 bottom-0 z-50 w-[260px] border-r border-hairline bg-card shadow-2xl lg:hidden flex flex-col animate-fade-in-up">
                {fileTree}
              </aside>
            </>
          )}

          {/* Center: Preview / Code */}
          <section className="flex flex-col min-w-0 border-r border-hairline bg-card/30 h-full min-h-0">
            {centerTabs}
            {activeTab === "preview" ? previewPane : codePane}
          </section>

          {/* Right: Chat */}
          <aside className="flex flex-col bg-card/70 backdrop-blur min-w-0 h-full min-h-0">
            <CardHeader className="py-3 px-4 border-b border-hairline-soft">
              <CardTitle className="text-xs font-bold uppercase tracking-wide text-mute flex items-center gap-2">
                <MessageSquare className="h-4 w-4" />
                {t("chat.title")}
              </CardTitle>
            </CardHeader>

            <ChatPanel items={chatItems} emptyHint={t("chat.empty")} scrollRef={chatEndRef} />

            <div className="p-3 border-t border-hairline bg-card/80 backdrop-blur">
              <form onSubmit={handleSubmit} className="flex items-center gap-2 rounded-full border border-hairline bg-card px-3 py-2 shadow-sm">
                <Textarea
                  value={prompt}
                  onChange={(e) => setPrompt(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === "Enter" && !e.shiftKey) {
                      e.preventDefault();
                      if (prompt.trim()) {
                        handleSubmit(e);
                      }
                    }
                  }}
                  placeholder={t("chat.placeholder")}
                  rows={1}
                  className="flex-1 resize-none bg-transparent border-0 text-ink placeholder:text-ash focus-visible:ring-0 focus-visible:ring-offset-0 min-h-0 py-1 px-0 max-h-32"
                />
                <Button
                  type="submit"
                  disabled={!prompt.trim()}
                  aria-label={t("prompt.send")}
                  className="h-8 w-8 shrink-0 rounded-full bg-primary text-primary-foreground hover:bg-primary-pressed disabled:opacity-50 p-0"
                >
                  <Send className="h-4 w-4" />
                </Button>
              </form>
            </div>
          </aside>
        </div>
      ) : (
        /* Empty state */
        <div className="relative flex-1 flex flex-col items-center justify-center p-6 text-center">
          <div className="relative mb-8">
            <div className="relative w-24 h-24 bg-card border border-hairline rounded-3xl shadow-xl flex items-center justify-center">
              <Image
                src="/logo.png"
                alt="Riffpad"
                width={64}
                height={64}
                className="riffpad-logo rounded-lg"
              />
            </div>
          </div>

          <h2 className="text-3xl sm:text-4xl font-extrabold text-ink mb-3 tracking-tight">
            {t("app.name")}
          </h2>
          <p className="text-body max-w-md mb-8 leading-relaxed text-base sm:text-lg">
            {t("app.tagline")}
          </p>

          <Button
            onClick={() => void createWorkspace()}
            className="bg-primary text-primary-foreground hover:bg-primary-pressed font-bold h-11 px-8 rounded-lg shadow-lg shadow-primary/20 transition hover:shadow-primary/30 hover:-translate-y-0.5"
          >
            {t("workspace.new")}
          </Button>

          <p className="mt-4 text-xs text-mute">{t("workspace.hint")}</p>
        </div>
      )}
    </div>
  );
}
