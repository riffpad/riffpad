import { useState } from "react";
import { useI18n } from "../lib/i18n";
import type { ApprovalPayload, RiffpadEvent } from "../lib/types";

type TFunc = ReturnType<typeof useI18n>["t"];

function eventText(ev: RiffpadEvent, t: TFunc): string {
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
      return t("session_end_reason") + String(p.reason || "end");
    default:
      return JSON.stringify(p);
  }
}

function argsPreview(args: Record<string, unknown> | undefined, t: TFunc): string {
  if (!args) return "";
  if (typeof args.content === "string") {
    const c = args.content as string;
    return t("content_preview") + "\n" + c.slice(0, 500) + (c.length > 500 ? "\n" + t("truncated") : "");
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
  const { t } = useI18n();
  const [sending, setSending] = useState<string | null>(null);
  const [done, setDone] = useState<Record<string, string>>({});
  const [err, setErr] = useState("");

  async function approve(p: ApprovalPayload, decision: "approve" | "reject") {
    setSending(decision);
    try {
      const sent = await send("approval_response", { requestId: p.requestId, decision });
      if (!sent) {
        setSending(null);
        setErr(t("approval_send_failed"));
        return;
      }
      setDone((d) => ({ ...d, [p.requestId]: decision === "approve" ? t("approved") : t("rejected") }));
    } catch {
      setSending(null);
      setErr(t("approval_send_retry"));
    }
  }

  const label = t("event_" + ev.type) || ev.type;
  if (ev.type === "approval_request") {
    const p = ev.payload as unknown as ApprovalPayload;
    const res = done[p.requestId];
    return (
      <div className={"ev " + ev.type + (res ? " ev-done" : "")}>
        <div className="ev-head"><span className="glyph">!</span>{label}</div>
        <div className="ev-body">{((p.action ? p.action + "：" : "") + (p.summary || "")).trim()}</div>
        <div className="row approval-actions">
          <button
            className={"approve" + (res === t("approved") ? " done" : "")}
            disabled={sending !== null || !!res}
            onClick={() => void approve(p, "approve")}
          >
            {sending === "approve" ? t("sending") : res === t("approved") ? t("approved") : t("approve")}
          </button>
          <button
            className={"danger reject" + (res === t("rejected") ? " done" : "")}
            disabled={sending !== null || !!res}
            onClick={() => void approve(p, "reject")}
          >
            {sending === "reject" ? t("sending") : res === t("rejected") ? t("rejected") : t("reject")}
          </button>
        </div>
        {p.args && <pre className="ev-body args">{argsPreview(p.args as Record<string, unknown>, t)}</pre>}
        {err && <div className="err">{err}</div>}
      </div>
    );
  }
  return (
    <div className={"ev " + ev.type}>
      <div className="ev-head">{label}</div>
      <div className="ev-body">{eventText(ev, t)}</div>
    </div>
  );
}
