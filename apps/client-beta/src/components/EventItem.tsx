import { useEffect, useState } from "react";
import { useI18n } from "../lib/i18n";
import type { SendResult } from "../lib/sessionSocket";
import type { ApprovalPayload, RiffpadEvent } from "../lib/types";
import Markdown from "./Markdown";

type TFunc = ReturnType<typeof useI18n>["t"];

// ApprovalOutcome is how a queued approval_response ended up, reported by the
// socket layer (SessionDetailView correlates the outbox event id back to the
// approval requestId):
// - flushed: written to the socket after a reconnect → safe to show 已批准
// - dropped: discarded on close() (tab killed while offline) → 未送达
// - expired: daemon answered the requestID is no longer pending → 已过期
export type ApprovalOutcome = "flushed" | "dropped" | "expired";

interface ApprovalState {
  st: "queued" | "done" | "failed" | "expired";
  decision?: "approve" | "reject";
}

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
  outcomes,
}: {
  ev: RiffpadEvent;
  send(type: string, payload: Record<string, unknown>): Promise<SendResult>;
  outcomes?: Record<string, ApprovalOutcome>;
}) {
  const { t } = useI18n();
  const [sending, setSending] = useState<string | null>(null);
  const [states, setStates] = useState<Record<string, ApprovalState>>({});
  const [err, setErr] = useState("");
  const p = ev.payload || {};
  const reqId = ev.type === "approval_request" ? String(p.requestId || "") : "";

  // The socket layer reports the fate of a queued approval asynchronously:
  // fold it into the card state. "expired" (daemon ack) always wins — even a
  // card already showing 已同意 was never actually applied.
  const outcome = reqId ? outcomes?.[reqId] : undefined;
  useEffect(() => {
    if (!reqId || !outcome) return;
    setStates((m) => {
      const cur = m[reqId];
      if (outcome === "expired") {
        if (cur?.st === "expired") return m;
        return { ...m, [reqId]: { st: "expired", decision: cur?.decision } };
      }
      if (!cur || cur.st !== "queued") return m;
      return {
        ...m,
        [reqId]: { st: outcome === "flushed" ? "done" : "failed", decision: cur.decision },
      };
    });
  }, [reqId, outcome]);

  async function approve(payload: ApprovalPayload, decision: "approve" | "reject") {
    setSending(decision);
    setErr("");
    try {
      const res = await send("approval_response", { requestId: payload.requestId, decision });
      setSending(null);
      if (res.status === "failed") {
        setErr(t("approval_send_failed"));
        return;
      }
      if (res.status === "queued") {
        // Offline: the tap is parked in the outbox. Show 待发送, never 已批准.
        setStates((m) => ({ ...m, [payload.requestId]: { st: "queued", decision } }));
        return;
      }
      setStates((m) => ({ ...m, [payload.requestId]: { st: "done", decision } }));
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
      return <div className={"status-line tone-" + level}>■ {String(p.message || "")}</div>;
    }

    case "approval_request": {
      const payload = p as unknown as ApprovalPayload;
      const st = states[payload.requestId];
      // "failed" unlocks the buttons so the user can retry; every other
      // terminal or in-flight state locks the card.
      const locked = sending !== null || (st !== undefined && st.st !== "failed");
      const cardState =
        st?.st === "done" ? " done" : st?.st === "queued" ? " pending" : st?.st === "expired" ? " expired" : "";
      return (
        <div className={"approval-card" + cardState}>
          <div className="approval-summary">{((payload.action ? payload.action + "：" : "") + (payload.summary || "")).trim()}</div>
          {payload.args && <pre className="tool-log-detail">{argsPreview(payload.args as Record<string, unknown>, t)}</pre>}
          <div className="approval-actions">
            <button
              className="approve"
              disabled={locked}
              onClick={() => void approve(payload, "approve")}
            >
              {sending === "approve" ? t("sending") : st?.st === "done" && st.decision === "approve" ? t("approved") : t("approve")}
            </button>
            <button
              className="reject"
              disabled={locked}
              onClick={() => void approve(payload, "reject")}
            >
              {sending === "reject" ? t("sending") : st?.st === "done" && st.decision === "reject" ? t("rejected") : t("reject")}
            </button>
          </div>
          {st?.st === "queued" && <div className="approval-note pending">{t("approval_pending")}</div>}
          {st?.st === "failed" && <div className="approval-note failed">{t("approval_not_delivered")}</div>}
          {st?.st === "expired" && <div className="approval-note expired">{t("approval_expired")}</div>}
          {err && <div className="err">{err}</div>}
        </div>
      );
    }

    default:
      return <div className="status-line">{t("event_" + ev.type) || ev.type}: {JSON.stringify(p)}</div>;
  }
}
