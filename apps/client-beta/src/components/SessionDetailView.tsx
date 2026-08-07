import { useEffect, useRef, useState } from "react";
import { ensureIdentity } from "../lib/device";
import { openSessionSocket, type SessionSocket } from "../lib/sessionSocket";
import { useI18n } from "../lib/i18n";
import type { RiffpadEvent } from "../lib/types";
import EventItem from "./EventItem";
import ToolLog, { type ToolLine } from "./ToolLog";

interface Props {
  sid: string;
  name: string;
  onLeave(): void;
}

type Row =
  | { id: string; kind: "event"; ev: RiffpadEvent }
  | { id: string; kind: "tool"; key: string };

function statusClass(status: string): string {
  if (/未连接|未配对|握手失败|连接失败|失败|断开/.test(status)) return "bad";
  if (/连接中|重连|等待/.test(status)) return "pending";
  return "good";
}

function StopIcon() {
  return (
    <svg width="13" height="13" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
      <rect x="5" y="5" width="14" height="14" />
    </svg>
  );
}

function toolLineFromEvent(ev: RiffpadEvent, t: ReturnType<typeof useI18n>["t"]): ToolLine | null {
  const p = ev.payload || {};
  if (ev.type === "tool_call") {
    const tool = String(p.tool || "");
    const status = String(p.status || "started");
    const args = p.args as Record<string, unknown> | undefined;
    const path = args && typeof args.path === "string" ? String(args.path) : "";
    const key = path ? "path:" + path : "tool:" + tool + ":" + String(p.summary || "");
    const glyph = path ? `${tool} ${path}` : `${tool} ${String(p.summary || "")}`.trim();
    const st: ToolLine["status"] = status === "completed" ? "done" : status === "failed" ? "fail" : "run";
    let detail = "";
    if (args) {
      if (typeof args.content === "string") {
        detail = t("content_preview") + "\n" + String(args.content).slice(0, 800) + (String(args.content).length > 800 ? "\n" + t("truncated") : "");
      } else {
        detail = JSON.stringify(args, null, 2).slice(0, 1200);
      }
    }
    return { key, glyph, status: st, detail };
  }
  if (ev.type === "file_change") {
    const path = String(p.path || "");
    return { key: "path:" + path, glyph: "FileChange " + path, status: "done", detail: String(p.summary || path) };
  }
  if (ev.type === "command") {
    const cmd = String(p.command || "");
    const exit = p.exitCode as number | undefined;
    return {
      key: "cmd:" + cmd,
      glyph: "$ " + cmd,
      status: exit === undefined ? "run" : exit === 0 ? "done" : "fail",
      detail: [String(p.output || ""), exit !== undefined ? "exit code: " + exit : ""].filter(Boolean).join("\n"),
    };
  }
  return null;
}

export default function SessionDetailView({ sid, name, onLeave }: Props) {
  const { t } = useI18n();
  const [rows, setRows] = useState<Row[]>([]);
  const [tools, setTools] = useState<Record<string, ToolLine>>({});
  const [prompt, setPrompt] = useState("");
  const [stopping, setStopping] = useState(false);
  const [err, setErr] = useState("");
  const [status, setStatus] = useState(t("connecting"));
  const [agentStatus, setAgentStatus] = useState("");
  const [meta, setMeta] = useState<{ cwd?: string; cli?: string }>({});
  const sockRef = useRef<SessionSocket | null>(null);
  const listRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    let cancelled = false;
    setRows([]);
    setTools({});
    setStatus(t("connecting"));
    setAgentStatus("");
    setMeta({});
    ensureIdentity()
      .then((dev) => {
        if (!dev.deviceId) {
          setStatus(t("not_paired"));
          return null;
        }
        return openSessionSocket(sid, dev, {
          onConn: setStatus,
          onEvent: (ev) => {
            if (cancelled) return;
            if (ev.type === "agent_status") {
              const st = String((ev.payload || {}).status || "");
              if (st) setAgentStatus(st);
              return;
            }
            if (ev.type === "session_start") {
              const p = ev.payload || {};
              setMeta({ cwd: String(p.cwd || ""), cli: String(p.cli || "") });
            }
            const tool = toolLineFromEvent(ev, t);
            if (tool) {
              setTools((prev) => {
                const ex = prev[tool.key];
                const status = ex && (ex.status === "done" || ex.status === "fail") ? ex.status : tool.status;
                return { ...prev, [tool.key]: { ...tool, glyph: ex ? ex.glyph : tool.glyph, detail: tool.detail || ex?.detail || "", status } };
              });
              setRows((prev) =>
                prev.some((r) => r.kind === "tool" && r.key === tool.key)
                  ? prev
                  : [...prev, { id: tool.key, kind: "tool", key: tool.key }],
              );
              return;
            }
            setRows((prev) => [...prev, { id: ev.id, kind: "event", ev }]);
          },
          onError: (message) => setStatus(t("handshake_failed") + message),
        });
      })
      .then((sock) => {
        if (cancelled) {
          sock?.close();
          return;
        }
        sockRef.current = sock;
      })
      .catch((e) => setStatus(t("connect_failed") + (e instanceof Error ? e.message : String(e))));
    return () => {
      cancelled = true;
      sockRef.current?.close();
      sockRef.current = null;
    };
  }, [sid, t]);

  useEffect(() => {
    const el = listRef.current;
    if (el) el.scrollTop = el.scrollHeight;
  }, [rows, tools]);

  async function sendPrompt() {
    const text = prompt.trim();
    if (!text || !sockRef.current) return;
    setPrompt("");
    const sent = await sockRef.current.send("prompt", { text });
    setErr(sent ? "" : t("send_failed"));
  }

  async function interrupt() {
    if (!sockRef.current) return;
    const sent = await sockRef.current.send("control", { action: "stop" });
    setErr(sent ? "" : t("send_failed"));
  }

  async function stop() {
    if (!window.confirm(t("confirm_stop"))) return;
    setStopping(true);
    try {
      await fetch("/api/sessions/" + sid + "/stop", { method: "POST" });
    } catch {
      // ignore
    }
    sockRef.current?.close();
    await new Promise((r) => setTimeout(r, 200));
    setStopping(false);
    onLeave();
  }

  const running = agentStatus === "running";
  const dir = meta.cwd?.split("/").filter(Boolean).pop();

  return (
    <section id="detail" className="card detail-card">
      <div className="detail-head">
        <button className="ghost back" onClick={onLeave}>← {t("back")}</button>
        <div className="detail-title truncate">
          <span className="detail-name truncate">{name || t("session_default")}</span>
          <span className="detail-id truncate">
            {[meta.cli, dir, sid.slice(0, 8)].filter(Boolean).join(" · ")}
          </span>
        </div>
        <span id="session-conn" className={"conn " + statusClass(status)}>{status}</span>
        <button
          className={"ghost stop-btn" + (running ? " armed" : "")}
          onClick={() => void stop()}
          disabled={stopping}
        >
          <StopIcon />
          {stopping ? t("stopping") : t("stop")}
        </button>
      </div>
      <div id="events" ref={listRef} className="events">
        {rows.map((row) =>
          row.kind === "tool" ? (
            <ToolLog key={row.key} line={tools[row.key]} />
          ) : (
            <EventItem
              key={row.id}
              ev={row.ev}
              send={(type, payload) => sockRef.current ? sockRef.current.send(type, payload) : Promise.resolve(false)}
            />
          ),
        )}
        {running && (
          <div className="agent-running">
            <span className="dot" />
            {t("agent_running")}
          </div>
        )}
        {rows.length === 0 && !running && <div className="events-empty muted">{t("waiting_events")}</div>}
      </div>
      <form
        className="prompt-form"
        onSubmit={(e) => {
          e.preventDefault();
          void sendPrompt();
        }}
      >
        <input
          placeholder={t("prompt_ph")}
          value={prompt}
          onChange={(e) => setPrompt(e.target.value)}
        />
        {running ? (
          <button type="button" className="danger interrupt" onClick={() => void interrupt()}>
            <StopIcon />
            {t("interrupt")}
          </button>
        ) : (
          <button type="submit" className="primary">{t("send")}</button>
        )}
      </form>
      {err && <div className="err">{err}</div>}
    </section>
  );
}
