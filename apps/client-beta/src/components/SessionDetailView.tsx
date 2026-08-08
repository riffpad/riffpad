import { useCallback, useEffect, useRef, useState } from "react";
import { ensureIdentity } from "../lib/device";
import { openSessionSocket, type SessionSocket } from "../lib/sessionSocket";
import { useI18n } from "../lib/i18n";
import type { RiffpadEvent } from "../lib/types";
import DotMatrix from "./DotMatrix";
import EventItem, { type ApprovalOutcome } from "./EventItem";
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

const HISTORY_PAGE_SIZE = 100;

function statusClass(status: string): string {
  if (/未连接|未配对|握手失败|连接失败|失败|断开/.test(status)) return "bad";
  if (/连接中|重连|等待|离线|offline/i.test(status)) return "pending";
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
  const [offline, setOffline] = useState("");
  const [historyLoading, setHistoryLoading] = useState(false);
  const [hasMoreHistory, setHasMoreHistory] = useState(true);
  const [status, setStatus] = useState(t("connecting"));
  const [agentStatus, setAgentStatus] = useState("");
  const [meta, setMeta] = useState<{ cwd?: string; cli?: string }>({ cwd, cli });
  // Fate of queued approval_responses, keyed by approval requestId; consumed
  // by EventItem to move a card out of 待发送 (or mark it 已过期).
  const [approvalOutcomes, setApprovalOutcomes] = useState<Record<string, ApprovalOutcome>>({});
  // outbox event id -> approval requestId, recorded when send() queues.
  const queuedApprovalsRef = useRef(new Map<string, string>());
  const [scroll, setScroll] = useState<{ can: boolean; top: boolean; bottom: boolean }>({ can: false, top: true, bottom: true });
  const sockRef = useRef<SessionSocket | null>(null);
  const listRef = useRef<HTMLDivElement | null>(null);
  const toolsRef = useRef<Record<string, ToolLine>>({});
  const rowsRef = useRef<Row[]>([]);
  const anchorRef = useRef<number | null>(null);

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
    setHasMoreHistory(true);
    setHistoryLoading(false);
    setStatus(t("connecting"));
    setFatal("");
    setOffline("");
    setAgentStatus("");
    setMeta({ cwd, cli });
    setApprovalOutcomes({});
    queuedApprovalsRef.current.clear();
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
            if (ev.type === "notify") {
              // Daemon ack for a late/unknown approval_response: mark the card
              // 已过期. The notify itself still renders as a status line below.
              const rid = String((ev.payload || {}).requestId || "");
              if (rid) setApprovalOutcomes((m) => ({ ...m, [rid]: "expired" }));
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
                rowsRef.current = [...rowsRef.current, { id: ev.id, kind: "tool", key: target }];
              }
              setRows([...rowsRef.current]);
              return;
            }
            rowsRef.current = [...rowsRef.current, { id: ev.id, kind: "event", ev }];
            setRows([...rowsRef.current]);
          },
          onError: (message) => setStatus(t("handshake_failed") + message),
          onFatal: (message) => setFatal(message),
          onOffline: (message) => setOffline(message ?? ""),
          onHistory: (events) => applyHistory(events),
          onOutbox: (id, status) => {
            if (cancelled) return;
            const rid = queuedApprovalsRef.current.get(id);
            if (!rid) return;
            queuedApprovalsRef.current.delete(id);
            setApprovalOutcomes((m) => ({ ...m, [rid]: status }));
          },
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
    if (anchorRef.current != null) {
      el.scrollTop = el.scrollTop + (el.scrollHeight - anchorRef.current);
      anchorRef.current = null;
      return;
    }
    const nearBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 60;
    if (nearBottom || rows.length === 0) el.scrollTop = el.scrollHeight;
    updateScroll();
  }, [rows, tools, updateScroll]);

  function applyHistory(batch: RiffpadEvent[]) {
    if (batch.length === 0) {
      setHasMoreHistory(false);
      setHistoryLoading(false);
      return;
    }
    const el = listRef.current;
    const prevHeight = el?.scrollHeight ?? 0;
    const nearBottom = el ? el.scrollHeight - el.scrollTop - el.clientHeight < 60 : true;
    const prepend: Row[] = [];
    for (const ev of batch) {
      const tool = toolLineFromEvent(ev, t);
      if (tool) {
        const cur = toolsRef.current;
        const ex = cur[tool.key];
        const status = ex && (ex.status === "done" || ex.status === "fail") ? ex.status : tool.status;
        toolsRef.current = { ...cur, [tool.key]: { ...tool, glyph: ex ? ex.glyph : tool.glyph, detail: tool.detail || ex?.detail || "", status } };
        if (!rowsRef.current.some((r) => r.kind === "tool" && r.key === tool.key)) {
          prepend.push({ id: ev.id, kind: "tool", key: tool.key });
        }
      } else {
        prepend.push({ id: ev.id, kind: "event", ev });
      }
    }
    rowsRef.current = [...prepend, ...rowsRef.current];
    setTools({ ...toolsRef.current });
    setRows([...rowsRef.current]);
    setHasMoreHistory(batch.length >= HISTORY_PAGE_SIZE);
    setHistoryLoading(false);
    if (el && !nearBottom) {
      anchorRef.current = prevHeight;
    }
  }

  function handleScroll() {
    const el = listRef.current;
    if (!el) return;
    updateScroll();
    const anchor = rowsRef.current[0]?.id;
    if (
      el.scrollTop < 80 &&
      anchor &&
      !historyLoading &&
      hasMoreHistory &&
      sockRef.current
    ) {
      setHistoryLoading(true);
      const ok = sockRef.current.requestHistory(anchor, HISTORY_PAGE_SIZE);
      if (!ok) setHistoryLoading(false);
    }
  }

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
    const res = await sockRef.current.send("prompt", { text });
    setErr(res.status === "failed" ? t("send_failed") : "");
  }

  async function interrupt() {
    if (!sockRef.current) return;
    const res = await sockRef.current.send("control", { action: "stop" });
    setErr(res.status === "failed" ? t("send_failed") : "");
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
      <div id="events" ref={listRef} className={"events" + (fadeClass ? " " + fadeClass : "")} onScroll={handleScroll}>
        {historyLoading && (
          <div className="history-loading">
            <DotMatrix />
            {t("history_loading")}
          </div>
        )}
        {rows.map((row) =>
          row.kind === "tool" ? (
            <ToolLog key={row.key} line={tools[row.key]} />
          ) : (
            <EventItem
              key={row.id}
              ev={row.ev}
              outcomes={approvalOutcomes}
              send={async (type, payload) => {
                if (!sockRef.current) return { status: "failed" as const, id: "" };
                const res = await sockRef.current.send(type, payload);
                // Remember which approval requestId a queued outbox event
                // belongs to so onOutbox can resolve the card later.
                if (res.status === "queued" && type === "approval_response") {
                  queuedApprovalsRef.current.set(res.id, String(payload.requestId || ""));
                }
                return res;
              }}
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
      {offline && !fatal && (
        <div id="offline-banner" className="fatal-banner">
          <span>{offline}</span>
          <div className="fatal-actions">
            <button className="ghost" onClick={() => sockRef.current?.retry()}>{t("retry_now")}</button>
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
