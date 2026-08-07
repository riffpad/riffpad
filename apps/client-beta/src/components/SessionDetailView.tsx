import { useCallback, useEffect, useRef, useState } from "react";
import { ensureIdentity } from "../lib/device";
import { openSessionSocket, type SessionSocket } from "../lib/sessionSocket";
import { useI18n } from "../lib/i18n";
import type { RiffpadEvent } from "../lib/types";
import EventItem from "./EventItem";
import ToolLog, { type ToolLine } from "./ToolLog";

interface Props {
  sid: string;
  name: string;
  cli?: string;
  cwd?: string;
  onLeave(): void;
  onReauth(): void;
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

function mergeTool(ex: ToolLine, line: ToolLine): ToolLine {
  const status = ex.status === "done" || ex.status === "fail" ? ex.status : line.status;
  return {
    key: ex.key,
    glyph: ex.glyph || line.glyph,
    status,
    detail: line.detail || ex.detail || "",
  };
}

export default function SessionDetailView({ sid, name, cli, cwd, onLeave, onReauth }: Props) {
  const { t } = useI18n();
  const [rows, setRows] = useState<Row[]>([]);
  const [tools, setTools] = useState<Record<string, ToolLine>>({});
  const [prompt, setPrompt] = useState("");
  const [err, setErr] = useState("");
  const [fatal, setFatal] = useState("");
  const [status, setStatus] = useState(t("connecting"));
  const [agentStatus, setAgentStatus] = useState("");
  const [meta, setMeta] = useState<{ cwd?: string; cli?: string }>({ cwd, cli });
  const [scroll, setScroll] = useState<{ can: boolean; top: boolean; bottom: boolean }>({ can: false, top: true, bottom: true });
  const sockRef = useRef<SessionSocket | null>(null);
  const listRef = useRef<HTMLDivElement | null>(null);
  const toolsRef = useRef<Record<string, ToolLine>>({});
  const rowsRef = useRef<Row[]>([]);

  const updateScroll = useCallback(() => {
    const el = listRef.current;
    if (!el) return;
    const can = el.scrollHeight > el.clientHeight + 4;
    const top = el.scrollTop < 8;
    const bottom = el.scrollHeight - el.scrollTop - el.clientHeight < 40;
    setScroll({ can, top, bottom });
  }, []);

  useEffect(() => {
    let cancelled = false;
    setRows([]);
    rowsRef.current = [];
    setTools({});
    toolsRef.current = {};
    setStatus(t("connecting"));
    setFatal("");
    setAgentStatus("");
    setMeta({ cwd, cli });
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
              const cur = toolsRef.current;
              const ex = cur[tool.key];
              let target = tool.key;
              let next: Record<string, ToolLine>;
              if (ex) {
                next = { ...cur, [tool.key]: mergeTool(ex, tool) };
              } else if (tool.key.startsWith("tool:")) {
                // A completed tool event without a path (e.g. FileChange)
                // merges into the path-keyed line already shown as running.
                const tname = tool.glyph.split(" ")[0].toLowerCase();
                const found = Object.entries(cur).find(
                  ([k, v]) => k.startsWith("path:") && v.glyph.toLowerCase().startsWith(tname),
                );
                if (found) {
                  target = found[0];
                  next = { ...cur, [found[0]]: mergeTool(found[1], tool) };
                } else {
                  next = { ...cur, [tool.key]: tool };
                }
              } else if (tool.key.startsWith("path:")) {
                // A bare file_change merges into the running tool line for the
                // same path when one exists.
                const path = tool.key.slice(5);
                const found = Object.entries(cur).find(
                  ([, v]) => v.key.startsWith("tool:") && v.status === "run" && v.glyph.includes(path),
                );
                if (found) {
                  target = found[0];
                  const glyph = found[1].glyph.includes(path) ? found[1].glyph : found[1].glyph + " " + path;
                  next = { ...cur, [found[0]]: mergeTool(found[1], { ...tool, glyph }) };
                } else {
                  next = { ...cur, [tool.key]: tool };
                }
              } else {
                next = { ...cur, [tool.key]: tool };
              }
              toolsRef.current = next;
              setTools(next);
              if (!rowsRef.current.some((r) => r.kind === "tool" && r.key === target)) {
                rowsRef.current = [...rowsRef.current, { id: target, kind: "tool", key: target }];
              }
              setRows([...rowsRef.current]);
              return;
            }
            rowsRef.current = [...rowsRef.current, { id: ev.id, kind: "event", ev }];
            setRows([...rowsRef.current]);
          },
          onError: (message) => setStatus(t("handshake_failed") + message),
          onFatal: (message) => setFatal(message),
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
  }, [sid, t, cwd, cli]);

  useEffect(() => {
    const el = listRef.current;
    if (!el) return;
    const nearBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 60;
    if (nearBottom || rows.length === 0) el.scrollTop = el.scrollHeight;
    updateScroll();
  }, [rows, tools, updateScroll]);

  function scrollToBottom() {
    const el = listRef.current;
    if (!el) return;
    el.scrollTop = el.scrollHeight;
    updateScroll();
  }

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

  const running = agentStatus === "running";
  const cwdPath = meta.cwd || cwd;
  const dir = cwdPath?.split("/").filter(Boolean).pop();
  const detailTitle = [meta.cli || cli, dir, sid.slice(0, 6)].filter(Boolean).join(" · ") || name || t("session_default");
  const fadeClass = !scroll.can ? "" : scroll.top ? (scroll.bottom ? "" : "fade-bottom") : scroll.bottom ? "fade-top" : "fade-both";

  return (
    <section id="detail" className="card detail-card">
      <div className="detail-head">
        <button className="ghost back" onClick={onLeave} aria-label={t("back")}>←</button>
        <div className="detail-title truncate">
          <span className="detail-name truncate">{detailTitle}</span>
        </div>
        <span id="session-conn" className={"conn-dot " + statusClass(status)} title={status} aria-label={status} />
      </div>
      <div id="events" ref={listRef} className={"events" + (fadeClass ? " " + fadeClass : "")} onScroll={updateScroll}>
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
      {rows.length === 0 && !running && (
          <div className="chat-skeleton" aria-hidden="true">
            {[0, 1, 2, 3, 4].map((i) => (
              <div
                key={i}
                className={"chat-skeleton-line" + (i % 2 === 1 ? " right" : "")}
                style={{ width: `${52 + ((i * 13) % 36)}%` }}
              />
            ))}
          </div>
        )}
      </div>
      {fatal && (
        <div id="fatal-banner" className="fatal-banner">
          <span>{fatal}</span>
          <div className="fatal-actions">
            <button className="ghost" onClick={onReauth}>{t("device_revoked_action")}</button>
            <button className="ghost" onClick={onLeave}>{t("back")}</button>
          </div>
        </div>
      )}
      {scroll.can && !scroll.bottom && (
        <button id="jump-bottom" className="jump-bottom" onClick={scrollToBottom}>↓ {t("jump_bottom")}</button>
      )}
      <form
        className="prompt-form"
        onSubmit={(e) => {
          e.preventDefault();
          void sendPrompt();
        }}
      >
        <div className="prompt-wrap">
          <input
            placeholder={t("prompt_ph")}
            value={prompt}
            onChange={(e) => setPrompt(e.target.value)}
          />
          {prompt.trim() && !running && (
            <button type="submit" id="send-btn" className="send-btn" aria-label={t("send")}>→</button>
          )}
        </div>
        {running ? (
          <button type="button" className="danger interrupt" onClick={() => void interrupt()}>
            <StopIcon />
            {t("interrupt")}
          </button>
        ) : null}
      </form>
      {err && <div className="err">{err}</div>}
    </section>
  );
}
