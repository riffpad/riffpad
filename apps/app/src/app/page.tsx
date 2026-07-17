"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import Image from "next/image";
import { useTheme } from "next-themes";
import { FileCode, Folder, Moon, Send, Sun, Terminal } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Textarea } from "@/components/ui/textarea";
import { useI18n } from "@/lib/i18n";

const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

interface AgentEvent {
  type: string;
  content?: string;
  name?: string;
  args?: unknown;
  result?: unknown;
  path?: string;
  toolCallId?: string;
  toolName?: string;
  isError?: boolean;
  message?: Record<string, unknown>;
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
  const { theme, setTheme, resolvedTheme } = useTheme();
  const [mounted, setMounted] = useState(false);

  const [prompt, setPrompt] = useState("");
  const [workspaceId, setWorkspaceId] = useState<string | null>(null);
  const [workspaceSlug, setWorkspaceSlug] = useState<string | null>(null);
  const [events, setEvents] = useState<AgentEvent[]>([]);
  const [connected, setConnected] = useState(false);
  const [files, setFiles] = useState<FileInfo[]>([]);
  const [selectedFile, setSelectedFile] = useState<string | null>(null);
  const [fileContent, setFileContent] = useState<string>("");

  const wsRef = useRef<WebSocket | null>(null);
  const eventsEndRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    setMounted(true);
  }, []);

  const fetchFiles = useCallback(async (id: string) => {
    try {
      const res = await fetch(`${API_URL}/api/v1/workspaces/${id}/files`);
      if (res.ok) {
        const data = await res.json();
        setFiles(data);
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
          setEvents((prev) => [...prev, event]);
          if (
            event.type === "file_change" ||
            event.type === "tool_execution_end"
          ) {
            void fetchFiles(id);
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

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!prompt.trim()) return;

    let id = workspaceId;
    if (!id) {
      const ws = await createWorkspace();
      id = ws.id;
    }

    wsRef.current?.send(JSON.stringify({ type: "prompt", content: prompt }));
    setPrompt("");
  };

  useEffect(() => {
    eventsEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [events]);

  useEffect(() => {
    return () => {
      wsRef.current?.close();
    };
  }, []);

  const handleFileClick = (file: FileInfo) => {
    if (!workspaceId || file.isDir) return;
    setSelectedFile(file.path);
    void fetchFile(workspaceId, file.path);
  };

  const toggleLocale = () => {
    setLocale(locale === "zh" ? "en" : "zh");
  };

  const toggleTheme = () => {
    if (!theme) return;
    if (theme === "system") {
      setTheme(resolvedTheme === "dark" ? "light" : "dark");
    } else {
      setTheme(theme === "dark" ? "light" : "dark");
    }
  };

  const isDark = resolvedTheme === "dark";

  return (
    <div className="min-h-screen bg-canvas text-body flex flex-col">
      {/* Header */}
      <header className="border-b border-hairline bg-card px-4 lg:px-6 h-14 flex items-center justify-between sticky top-0 z-40">
        <div className="flex items-center gap-3">
          <Image
            src="/logo.png"
            alt="Riffpad"
            width={32}
            height={32}
            className="rounded-md"
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

        <div className="flex items-center gap-2">
          {workspaceId ? (
            <div className="hidden sm:flex items-center gap-2 text-xs text-mute mr-2">
              <span className="font-medium text-ink">
                {workspaceSlug || workspaceId}
              </span>
              <span
                className={`inline-block h-2 w-2 rounded-full ${
                  connected ? "bg-accent-green" : "bg-accent-red"
                }`}
              />
            </div>
          ) : null}

          <Button
            variant="ghost"
            size="sm"
            onClick={toggleLocale}
            className="text-xs font-semibold uppercase tracking-wide h-8 px-2"
          >
            {locale === "zh" ? "EN" : "中"}
          </Button>

          <Button
            variant="ghost"
            size="icon"
            onClick={toggleTheme}
            className="h-8 w-8"
            aria-label="toggle theme"
          >
            {mounted && isDark ? (
              <Sun className="h-4 w-4" />
            ) : (
              <Moon className="h-4 w-4" />
            )}
          </Button>

          {!workspaceId && (
            <Button
              onClick={() => void createWorkspace()}
              size="sm"
              className="bg-primary text-primary-foreground hover:bg-primary-pressed h-8 text-xs font-bold rounded-md"
            >
              {t("workspace.new")}
            </Button>
          )}
        </div>
      </header>

      {/* Workspace layout */}
      {workspaceId ? (
        <div className="flex-1 grid grid-cols-1 lg:grid-cols-[260px_1fr_320px] overflow-hidden">
          {/* File tree */}
          <aside className="border-r border-hairline bg-card hidden lg:flex flex-col">
            <CardHeader className="py-3 px-4">
              <CardTitle className="text-xs font-bold uppercase tracking-wide text-mute flex items-center gap-2">
                <Folder className="h-4 w-4" />
                {t("files.title")}
              </CardTitle>
            </CardHeader>
            <ScrollArea className="flex-1 px-2">
              {files.length === 0 && (
                <p className="text-xs text-mute px-2 py-1">
                  {t("files.empty")}
                </p>
              )}
              <ul className="space-y-0.5 pb-4">
                {files.map((file) => (
                  <li key={file.path}>
                    <button
                      onClick={() => handleFileClick(file)}
                      className={`w-full flex items-center gap-2 rounded-md px-2 py-1.5 text-sm text-left transition-colors ${
                        selectedFile === file.path
                          ? "bg-canvas text-ink"
                          : "hover:bg-card-soft text-body"
                      }`}
                    >
                      {file.isDir ? (
                        <Folder className="h-4 w-4 text-mute" />
                      ) : (
                        <FileCode className="h-4 w-4 text-mute" />
                      )}
                      <span className="truncate">{file.name}</span>
                    </button>
                  </li>
                ))}
              </ul>
            </ScrollArea>
          </aside>

          {/* Center */}
          <section className="flex flex-col min-w-0 border-r border-hairline">
            <div className="p-4 border-b border-hairline bg-card">
              <form onSubmit={handleSubmit} className="flex gap-3">
                <Textarea
                  value={prompt}
                  onChange={(e) => setPrompt(e.target.value)}
                  placeholder={t("prompt.placeholder")}
                  className="min-h-[72px] resize-none bg-card border-hairline text-ink placeholder:text-ash"
                />
                <Button
                  type="submit"
                  disabled={!prompt.trim()}
                  className="self-end h-10 bg-primary text-primary-foreground hover:bg-primary-pressed font-bold rounded-md disabled:opacity-50"
                >
                  <Send className="h-4 w-4 mr-1.5" />
                  {t("prompt.send")}
                </Button>
              </form>
            </div>

            <div className="flex-1 p-4 min-h-0 overflow-hidden bg-canvas">
              {selectedFile ? (
                <Card className="h-full flex flex-col border-hairline bg-card">
                  <CardHeader className="py-3 px-4 border-b border-hairline-soft">
                    <CardTitle className="text-sm font-semibold text-ink flex items-center gap-2">
                      <FileCode className="h-4 w-4 text-mute" />
                      {selectedFile}
                    </CardTitle>
                  </CardHeader>
                  <CardContent className="flex-1 min-h-0 p-0">
                    <ScrollArea className="h-full">
                      <pre className="p-4 text-sm font-mono text-body whitespace-pre-wrap">
                        {fileContent}
                      </pre>
                    </ScrollArea>
                  </CardContent>
                </Card>
              ) : (
                <div className="h-full flex items-center justify-center text-mute text-sm">
                  {t("viewer.empty")}
                </div>
              )}
            </div>
          </section>

          {/* Events */}
          <aside className="flex flex-col bg-card min-w-0">
            <CardHeader className="py-3 px-4 border-b border-hairline-soft">
              <CardTitle className="text-xs font-bold uppercase tracking-wide text-mute flex items-center gap-2">
                <Terminal className="h-4 w-4" />
                {t("events.title")}
              </CardTitle>
            </CardHeader>
            <ScrollArea className="flex-1 px-3">
              <div className="space-y-2 py-3">
                {events.length === 0 && (
                  <p className="text-xs text-mute">{t("events.empty")}</p>
                )}
                {events.map((event, idx) => (
                  <div
                    key={idx}
                    className="rounded-md border border-hairline bg-card-doc p-2.5 text-xs"
                  >
                    <div className="flex items-center gap-2 mb-1">
                      <span
                        className={`inline-flex items-center px-1.5 py-0.5 rounded-xs text-[10px] font-bold uppercase tracking-wide ${
                          event.isError
                            ? "bg-accent-red-soft text-accent-red"
                            : event.type === "tool_execution_end"
                            ? "bg-accent-green-soft text-accent-green"
                            : event.type === "message_end"
                            ? "bg-accent-blue-soft text-accent-blue"
                            : "bg-card-soft text-mute"
                        }`}
                      >
                        {event.type}
                      </span>
                      <span className="text-[10px] text-ash">
                        {new Date(event.timestamp).toLocaleTimeString()}
                      </span>
                    </div>
                    {event.content && (
                      <p className="text-body whitespace-pre-wrap leading-relaxed">
                        {event.content}
                      </p>
                    )}
                    {event.toolName && (
                      <p className="text-ash mt-1">tool: {event.toolName}</p>
                    )}
                    {event.path && (
                      <p className="text-ash">path: {event.path}</p>
                    )}
                  </div>
                ))}
                <div ref={eventsEndRef} />
              </div>
            </ScrollArea>
          </aside>
        </div>
      ) : (
        /* Empty state */
        <div className="flex-1 flex flex-col items-center justify-center p-6 text-center">
          <div className="w-20 h-20 mb-6 bg-primary/10 rounded-2xl flex items-center justify-center">
            <Image
              src="/logo.png"
              alt="Riffpad"
              width={56}
              height={56}
              className="rounded-md"
            />
          </div>
          <h2 className="text-2xl font-extrabold text-ink mb-2">
            {t("app.name")}
          </h2>
          <p className="text-body max-w-md mb-8 leading-relaxed">
            {t("app.tagline")}
          </p>
          <Button
            onClick={() => void createWorkspace()}
            className="bg-primary text-primary-foreground hover:bg-primary-pressed font-bold h-10 px-6 rounded-md"
          >
            {t("workspace.new")}
          </Button>
        </div>
      )}
    </div>
  );
}
