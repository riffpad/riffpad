"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { Sparkles } from "lucide-react";

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

export default function Home() {
  const [prompt, setPrompt] = useState("");
  const [workspaceId, setWorkspaceId] = useState<string | null>(null);
  const [events, setEvents] = useState<AgentEvent[]>([]);
  const [connected, setConnected] = useState(false);
  const wsRef = useRef<WebSocket | null>(null);

  const connect = useCallback((id: string) => {
    const wsUrl = API_URL.replace(/^http/, "ws");
    const ws = new WebSocket(`${wsUrl}/ws/workspaces/${id}`);
    wsRef.current = ws;

    ws.onopen = () => setConnected(true);
    ws.onclose = () => setConnected(false);
    ws.onerror = (err) => {
      console.error("WebSocket error:", err);
      setConnected(false);
    };
    ws.onmessage = (msg) => {
      try {
        const event: AgentEvent = JSON.parse(msg.data);
        setEvents((prev) => [...prev, event]);
      } catch {
        console.error("Failed to parse event:", msg.data);
      }
    };
  }, []);

  const createWorkspace = async () => {
    const res = await fetch(`${API_URL}/api/v1/workspaces`, { method: "POST" });
    if (!res.ok) throw new Error("Failed to create workspace");
    const data = await res.json();
    setWorkspaceId(data.id);
    connect(data.id);
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

    wsRef.current?.send(
      JSON.stringify({ type: "prompt", content: prompt })
    );
    setPrompt("");
  };

  useEffect(() => {
    return () => {
      wsRef.current?.close();
    };
  }, []);

  return (
    <main className="min-h-screen flex flex-col px-4 sm:px-6 py-8">
      <div className="w-full max-w-3xl mx-auto space-y-6">
        <div className="text-center space-y-2">
          <h1 className="text-3xl sm:text-5xl font-bold tracking-tight">Riffpad</h1>
          <p className="text-muted-foreground text-base sm:text-lg">
            AI-native sketchbook for code. Describe an idea, make it run.
          </p>
          {workspaceId && (
            <p className="text-xs text-muted-foreground">
              Workspace: {workspaceId} · {connected ? "connected" : "disconnected"}
            </p>
          )}
        </div>

        <form onSubmit={handleSubmit} className="relative">
          <textarea
            value={prompt}
            onChange={(e) => setPrompt(e.target.value)}
            placeholder="Describe your idea..."
            className="w-full min-h-[120px] rounded-xl border bg-background p-4 pr-14 text-base shadow-sm resize-none focus:outline-none focus:ring-2 focus:ring-ring"
          />
          <button
            type="submit"
            disabled={!prompt.trim()}
            className="absolute bottom-3 right-3 inline-flex items-center justify-center rounded-lg bg-primary px-3 py-2 text-sm font-medium text-primary-foreground shadow transition-colors hover:bg-primary/90 disabled:opacity-50 disabled:pointer-events-none"
          >
            <Sparkles className="mr-1.5 h-4 w-4" />
            Spark
          </button>
        </form>

        <div className="space-y-2">
          <h2 className="text-sm font-medium text-muted-foreground">Events</h2>
          <div className="rounded-xl border bg-background p-4 min-h-[200px] max-h-[50vh] overflow-y-auto space-y-2 font-mono text-sm">
            {events.length === 0 && (
              <p className="text-muted-foreground">No events yet.</p>
            )}
            {events.map((event, idx) => (
              <div key={idx} className="border-b pb-2 last:border-0">
                <span className="text-xs text-muted-foreground">
                  {new Date(event.timestamp).toLocaleTimeString()}
                </span>{" "}
                <span className="font-semibold">{event.type}</span>
                {event.content && (
                  <p className="text-muted-foreground whitespace-pre-wrap">{event.content}</p>
                )}
                {event.toolName && (
                  <p className="text-xs text-muted-foreground">tool: {event.toolName}</p>
                )}
                {event.path && (
                  <p className="text-xs text-muted-foreground">path: {event.path}</p>
                )}
              </div>
            ))}
          </div>
        </div>
      </div>
    </main>
  );
}
