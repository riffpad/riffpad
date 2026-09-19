import { useI18n } from "../lib/i18n";
import type { SessionInfo } from "../lib/types";

type TFunc = ReturnType<typeof useI18n>["t"];

function timeAgo(iso: string | undefined, t: TFunc): string {
  if (!iso) return "";
  const ts = new Date(iso).getTime();
  // Go's zero time ("0001-01-01…") parses fine but is ~740k days ago;
  // treat any timestamp before 2000 as "missing" instead of a huge delta.
  if (!(ts >= 0) || ts < Date.UTC(2000, 0, 1)) return "";
  const ms = Date.now() - ts;
  const m = Math.floor(ms / 60000);
  if (m < 1) return t("time_just_now");
  if (m < 60) return t("time_min_ago", { n: m });
  const h = Math.floor(m / 60);
  if (h < 24) return t("time_hour_ago", { n: h });
  return t("time_day_ago", { n: Math.floor(h / 24) });
}


export function MenuIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
      <circle cx="5" cy="12" r="1.6" />
      <circle cx="12" cy="12" r="1.6" />
      <circle cx="19" cy="12" r="1.6" />
    </svg>
  );
}

export function statusTone(status?: string): string {
  switch (status) {
    case "waiting_input":
    case "waiting_approval": return "waiting";
    case "running": return "running";
    case "done": return "done";
    default: return status ? "idle" : "idle";
  }
}

export function statusLabel(status?: string): string {
  switch (status) {
    case "waiting_input": return "WAITING FOR INPUT";
    case "waiting_approval": return "NEEDS APPROVAL";
    case "running": return "RUNNING";
    case "done": return "DONE";
    default: return (status || "—").toUpperCase();
  }
}

interface Props {
  session: SessionInfo;
  onOpen(sid: string, name: string, cli?: string, cwd?: string): void;
  onMenu(): void;
}

// One row of the session list: title, meta line, live status light and the
// per-session menu button (#300).
export default function SessionListItem({ session: s, onOpen, onMenu }: Props) {
  const { t } = useI18n();
  const dir = s.cwd?.split("/").filter(Boolean).pop();
  const title = s.displayName || s.name || dir || "session-" + s.id.slice(0, 8);
  const meta = [s.cli, s.id.slice(0, 8), timeAgo(s.lastSeenAt, t)].filter(Boolean).join(" · ");
  return (
    <li className="session" onClick={() => onOpen(s.id, s.displayName || s.name || "", s.cli, s.cwd)}>
      <div className="session-main">
        <span className="session-name truncate">{title}</span>
        <span className="session-meta truncate" title={`${s.cwd || ""} ${s.id}`}>{meta}</span>
      </div>
      <span className={"session-light " + statusTone(s.status)}><span className="dot" />{statusLabel(s.status)}</span>
      <button className="session-menu-btn" aria-label={t("session_actions")} onClick={(e) => { e.stopPropagation(); onMenu(); }}>
        <MenuIcon />
      </button>
    </li>
  );
}
