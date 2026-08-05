import { useState } from "react";
import { EVENT_LABELS, type ApprovalPayload, type RiffpadEvent } from "../lib/types";

function eventText(ev: RiffpadEvent): string {
  const p = ev.payload || {};
  switch (ev.type) {
    case "agent_message":
    case "user_message":
      return String(p.text || "");
    case "tool_call":
      return String((p.tool || "") + " " + (p.summary || ""));
    case "file_change":
      return String(p.path || "");
    case "command":
      return String(p.command || "");
    case "agent_status":
      return String(p.status || "");
    case "notify":
      return String(p.message || "");
    case "session_end":
      return "原因: " + String(p.reason || "结束");
    default:
      return JSON.stringify(p);
  }
}

function argsPreview(args?: Record<string, unknown>): string {
  if (!args) return "";
  if (typeof args.content === "string") {
    const c = args.content as string;
    return "内容预览：\n" + c.slice(0, 500) + (c.length > 500 ? "\n…[截断]" : "");
  }
  return JSON.stringify(args, null, 2).slice(0, 800);
}

export default function EventItem({
  ev,
  send,
}: {
  ev: RiffpadEvent;
  send(type: string, payload: Record<string, unknown>): Promise<boolean>;
}) {
  const [sending, setSending] = useState<string | null>(null);
  const [done, setDone] = useState<Record<string, string>>({});

  async function approve(p: ApprovalPayload, decision: "approve" | "reject") {
    setSending(decision);
    try {
      const sent = await send("approval_response", { requestId: p.requestId, decision });
      if (!sent) {
        setSending(null);
        return;
      }
      setDone((d) => ({ ...d, [p.requestId]: decision === "approve" ? "已同意" : "已拒绝" }));
    } catch {
      setSending(null);
    }
  }

  const label = EVENT_LABELS[ev.type] || ev.type;
  if (ev.type === "approval_request") {
    const p = ev.payload as unknown as ApprovalPayload;
    const res = done[p.requestId];
    return (
      <div className={"ev " + ev.type}>
        <div className="ev-head">{label}</div>
        <div className="ev-body">{((p.action ? p.action + "：" : "") + (p.summary || "")).trim()}</div>
        <div className="row">
          <button
            disabled={sending !== null || !!res}
            onClick={() => void approve(p, "approve")}
          >
            {sending === "approve" ? "发送中…" : res === "已同意" ? "已同意" : "同意"}
          </button>
          <button
            className="danger"
            disabled={sending !== null || !!res}
            onClick={() => void approve(p, "reject")}
          >
            {sending === "reject" ? "发送中…" : res === "已拒绝" ? "已拒绝" : "拒绝"}
          </button>
        </div>
        {p.args && <pre className="ev-body">{argsPreview(p.args as Record<string, unknown>)}</pre>}
      </div>
    );
  }
  return (
    <div className={"ev " + ev.type}>
      <div className="ev-head">{label}</div>
      <div className="ev-body">{eventText(ev)}</div>
    </div>
  );
}
