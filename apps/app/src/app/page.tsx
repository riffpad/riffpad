"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { FileCode, Folder, Send, Sparkles, Terminal } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Textarea } from "@/components/ui/textarea";

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

  return (
    <main className="min-h-screen bg-background text-foreground flex flex-col">
      {/* Header */}
      <header className="border-b bg-card px-4 py-3 flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Sparkles className="h-5 w-5 text-primary" />
          <h1 className="font-semibold tracking-tight">Riffpad</h1>
        </div>
        {workspaceId ? (
          <div className="flex items-center gap-2 text-xs text-muted-foreground">
            <span>{workspaceSlug || workspaceId}</span>
            <Badge variant={connected ? "default" : "secondary"}>
              {connected ? "connected" : "disconnected"}
            </Badge>
          </div>
        ) : (
          <Badge variant="outline">no workspace</Badge>
        )}
      </header>

      {/* Main workspace */}
      <div className="flex-1 grid grid-cols-1 md:grid-cols-[260px_1fr_320px] overflow-hidden">
        {/* File tree */}
        <aside className="border-r bg-card hidden md:flex flex-col">
          <CardHeader className="py-3">
            <CardTitle className="text-sm font-medium flex items-center gap-2">
              <Folder className="h-4 w-4" /> Files
            </CardTitle>
          </CardHeader>
          <ScrollArea className="flex-1 px-3">
            {files.length === 0 && (
              <p className="text-xs text-muted-foreground px-3">
                No files yet.
              </p>
            )}
            <ul className="space-y-1 pb-4">
              {files.map((file) => (
                <li key={file.path}>
                  <button
                    onClick={() => handleFileClick(file)}
                    className={`w-full flex items-center gap-2 rounded-md px-2 py-1.5 text-sm text-left hover:bg-accent hover:text-accent-foreground ${
                      selectedFile === file.path ? "bg-accent text-accent-foreground" : ""
                    }`}
                  >
                    {file.isDir ? (
                      <Folder className="h-4 w-4 text-muted-foreground" />
                    ) : (
                      <FileCode className="h-4 w-4 text-muted-foreground" />
                    )}
                    <span className="truncate">{file.name}</span>
                  </button>
                </li>
              ))}
            </ul>
          </ScrollArea>
        </aside>

        {/* Center: prompt + file viewer */}
        <section className="flex flex-col min-w-0 border-r">
          <div className="p-4 border-b">
            <form onSubmit={handleSubmit} className="flex gap-2">
              <Textarea
                value={prompt}
                onChange={(e) => setPrompt(e.target.value)}
                placeholder="Describe your idea... e.g. Write a Python script that prints the current time"
                className="min-h-[80px] resize-none"
              />
              <Button
                type="submit"
                disabled={!prompt.trim()}
                className="self-end h-10"
              >
                <Send className="h-4 w-4 mr-1" />
                Spark
              </Button>
            </form>
          </div>

          <div className="flex-1 p-4 min-h-0 overflow-hidden">
            {selectedFile ? (
              <Card className="h-full flex flex-col">
                <CardHeader className="py-3">
                  <CardTitle className="text-sm font-medium flex items-center gap-2">
                    <FileCode className="h-4 w-4" />
                    {selectedFile}
                  </CardTitle>
                </CardHeader>
                <CardContent className="flex-1 min-h-0 p-0">
                  <ScrollArea className="h-full">
                    <pre className="p-4 text-sm font-mono whitespace-pre-wrap">
                      {fileContent}
                    </pre>
                  </ScrollArea>
                </CardContent>
              </Card>
            ) : (
              <div className="h-full flex items-center justify-center text-muted-foreground text-sm">
                Select a file from the sidebar to view its contents.
              </div>
            )}
          </div>
        </section>

        {/* Right: events / agent log */}
        <aside className="flex flex-col bg-card min-w-0">
          <CardHeader className="py-3">
            <CardTitle className="text-sm font-medium flex items-center gap-2">
              <Terminal className="h-4 w-4" /> Agent Events
            </CardTitle>
          </CardHeader>
          <ScrollArea className="flex-1 px-3">
            <div className="space-y-2 pb-4">
              {events.length === 0 && (
                <p className="text-xs text-muted-foreground">
                  No events yet. Send a prompt to start.
                </p>
              )}
              {events.map((event, idx) => (
                <div
                  key={idx}
                  className="rounded-md border bg-background p-2 text-xs"
                >
                  <div className="flex items-center gap-2 mb-1">
                    <Badge
                      variant={event.isError ? "destructive" : "outline"}
                      className="text-[10px] h-5"
                    >
                      {event.type}
                    </Badge>
                    <span className="text-muted-foreground">
                      {new Date(event.timestamp).toLocaleTimeString()}
                    </span>
                  </div>
                  {event.content && (
                    <p className="text-muted-foreground whitespace-pre-wrap">
                      {event.content}
                    </p>
                  )}
                  {event.toolName && (
                    <p className="text-muted-foreground">
                      tool: {event.toolName}
                    </p>
                  )}
                  {event.path && (
                    <p className="text-muted-foreground">path: {event.path}</p>
                  )}
                </div>
              ))}
              <div ref={eventsEndRef} />
            </div>
          </ScrollArea>
        </aside>
      </div>
    </main>
  );
}
