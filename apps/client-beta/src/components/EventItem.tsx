import { useState } from "react";
import { useI18n } from "../lib/i18n";
import type { ApprovalPayload, RiffpadEvent } from "../lib/types";
import Markdown from "./Markdown";

type TFunc = ReturnType<typeof useI18n>["t"];

function argsPreview(args: Record<string, unknown> | undefined, t: TFunc): string {
  if (!args) return "";
  if (typeof args.content === "string") {
    const c = args.content as string;
    return t("content_preview") + "\n" + c.slice(0, 800) + (c.length > 800 ? "\n" + t("truncated") : "");
  }
  return JSON.stringify(args, null, 2).slice(0, 1200);
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
  const p = ev.payload || {};

  async function approve(payload: ApprovalPayload, decision: "approve" | "reject") {
    setSending(decision);
    try {
      const sent = await send("approval_response", { requestId: payload.requestId, decision });
      if (!sent) {
        setSending(null);
        setErr(t("approval_send_failed"));
        return;
      }
      setDone((d) => ({ ...d, [payload.requestId]: decision === "approve" ? t("approved") : t("rejected") }));
    } catch {
      setSending(null);
      setErr(t("approval_send_retry"));
    }
  }

  switch (ev.type) {
    case "user_message":
      return <div className="msg user">{String(p.text || "")}</div>;

    case "agent_message":
      return (
        <div className="msg agent">
          <Markdown text={String(p.text || "")} />
        </div>
      );

    case "session_start": {
      const cwd = String(p.cwd || "");
      const cli = String(p.cli || "");
      return (
        <div className="status-line">
          <span className="badge">cwd: {cwd || "—"}</span>
          <span className="badge">cli: {cli || "—"}</span>
        </div>
      );
    }

    case "session_end":
      return <div className="status-line end">—— {t("event_session_end")} · {t("session_end_reason")}{String(p.reason || "end")} ——</div>;

    case "notify": {
      const level = String(p.level || "info");
      return <div className={"status-line tone-" + level}>● {String(p.message || "")}</div>;
    }

    case "approval_request": {
      const payload = p as unknown as ApprovalPayload;
      const res = done[payload.requestId];
      return (
        <div className={"approval-card" + (res ? " done" : "")}>
          <div className="approval-summary">{((payload.action ? payload.action + "：" : "") + (payload.summary || "")).trim()}</div>
          {payload.args && <pre className="tool-log-detail">{argsPreview(payload.args as Record<string, unknown>, t)}</pre>}
          <div className="approval-actions">
            <button
              className="approve"
              disabled={sending !== null || !!res}
              onClick={() => void approve(payload, "approve")}
            >
              {sending === "approve" ? t("sending") : res === t("approved") ? t("approved") : t("approve")}
            </button>
            <button
              className="reject"
              disabled={sending !== null || !!res}
              onClick={() => void approve(payload, "reject")}
            >
              {sending === "reject" ? t("sending") : res === t("rejected") ? t("rejected") : t("reject")}
            </button>
          </div>
          {err && <div className="err">{err}</div>}
        </div>
      );
    }

    default:
      return <div className="status-line">{t("event_" + ev.type) || ev.type}: {JSON.stringify(p)}</div>;
  }
}
