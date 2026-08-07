import { useCallback, useEffect, useRef, useState } from "react";
import { api } from "../lib/store";
import { useI18n } from "../lib/i18n";
import type { SessionInfo } from "../lib/types";

interface Props {
  onOpen(sid: string, name: string, cli?: string, cwd?: string): void;
}

const CWD_HISTORY_KEY = "riffpad.cwdHistory";

function loadCwdHistory(): string[] {
  try {
    const v = JSON.parse(localStorage.getItem(CWD_HISTORY_KEY) || "[]");
    return Array.isArray(v) ? v.filter((x): x is string => typeof x === "string") : [];
  } catch {
    return [];
  }
}

function saveCwdHistory(list: string[]) {
  try {
    localStorage.setItem(CWD_HISTORY_KEY, JSON.stringify(list.slice(0, 5)));
  } catch {
    // ignore
  }
}

type TFunc = ReturnType<typeof useI18n>["t"];

function timeAgo(iso: string | undefined, t: TFunc): string {
  if (!iso) return "";
  const ms = Date.now() - new Date(iso).getTime();
  if (!(ms >= 0)) return "";
  const m = Math.floor(ms / 60000);
  if (m < 1) return t("time_just_now");
  if (m < 60) return t("time_min_ago", { n: m });
  const h = Math.floor(m / 60);
  if (h < 24) return t("time_hour_ago", { n: h });
  return t("time_day_ago", { n: Math.floor(h / 24) });
}

function statusTone(status?: string): string {
  switch (status) {
    case "waiting_input": return "waiting";
    case "running": return "running";
    case "done": return "done";
    case "error": return "error";
    case "restored": return "restored";
    default: return "muted";
  }
}

function statusLabel(status?: string): string {
  switch (status) {
    case "waiting_input": return "WAITING FOR INPUT";
    case "running": return "RUNNING";
    case "done": return "DONE";
    case "error": return "ERROR";
    case "restored": return "RESTORED";
    default: return (status || "—").toUpperCase();
  }
}

function PlusIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="square" aria-hidden="true">
      <path d="M12 5v14M5 12h14" />
    </svg>
  );
}

export default function SessionListView({ onOpen }: Props) {
  const { t } = useI18n();
  const [sessions, setSessions] = useState<SessionInfo[]>([]);
  const [sheetOpen, setSheetOpen] = useState(false);
  const [name, setName] = useState("");
  const [prompt, setPrompt] = useState("");
  const [cli, setCli] = useState("claude");
  const [cwd, setCwd] = useState("");
  const [cwdHistory, setCwdHistory] = useState<string[]>(loadCwdHistory);
  const [busy, setBusy] = useState(false);
  const [loading, setLoading] = useState(true);
  const timer = useRef<number | null>(null);

  const refresh = useCallback(async () => {
    try {
      const res = await api("/api/sessions");
      const data = await res.json();
      setSessions(data.sessions || []);
    } catch {
      // transient network glitch; poll again later
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void refresh();
    timer.current = window.setInterval(() => void refresh(), 5000);
    return () => {
      if (timer.current) window.clearInterval(timer.current);
    };
  }, [refresh]);

  async function create() {
    setBusy(true);
    try {
      const res = await api("/api/sessions", {
        method: "POST",
        body: JSON.stringify({ name, prompt, cli, cwd }),
      });
      const data = await res.json();
      if (!res.ok) throw new Error(data.error || t("start_failed"));
      const trimmed = cwd.trim();
      if (trimmed) {
        const next = [trimmed, ...cwdHistory.filter((x) => x !== trimmed)];
        saveCwdHistory(next);
        setCwdHistory(next);
      }
      const finalName = data.name || "session-" + String(data.id).slice(0, 8);
      setName("");
      setPrompt("");
      setCwd("");
      setCli("claude");
      setSheetOpen(false);
      await refresh();
      onOpen(data.id, finalName);
    } catch (e) {
      alert(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <>
      <p className="section-label"><span className="glyph">//</span>{t("sessions_label")}</p>
      {loading && sessions.length === 0 ? (
        <div id="session-skeleton" className="session-skeleton" aria-hidden="true">
          {[0, 1, 2].map((i) => (
            <div key={i} className="skeleton-card">
              <div className="skeleton-lines">
                <div className="skeleton-line w60" />
                <div className="skeleton-line w35" />
              </div>
              <span className="skeleton-dot" />
            </div>
          ))}
        </div>
      ) : (
        <ul id="session-list">
          {sessions.map((s) => {
            const dir = s.cwd?.split("/").filter(Boolean).pop();
            const title = s.name || dir || "session-" + s.id.slice(0, 8);
            const meta = [s.cli, s.id.slice(0, 8), timeAgo(s.lastSeenAt, t)].filter(Boolean).join(" · ");
            const tone = statusTone(s.status);
            return (
              <li key={s.id} className="session" onClick={() => onOpen(s.id, s.name || "", s.cli, s.cwd)}>
                <div className="session-main">
                  <span className="session-name truncate">
                    {title}
                  </span>
                  <span className="session-meta truncate" title={`${s.cwd || ""} ${s.id}`}>{meta}</span>
                </div>
                <span className={"session-light " + tone}><span className="dot" />{statusLabel(s.status)}</span>
              </li>
            );
          })}
          {sessions.length === 0 && <li className="empty muted">{t("no_sessions")}</li>}
        </ul>
      )}

      {!loading && sessions.length === 0 && (
        <section className="card empty-card">
          <h3><span className="glyph">//</span>{t("empty_title")}</h3>
          <p className="muted">{t("empty_run_hint")}</p>
          <code className="empty-cmd">riffpad run --cli codex</code>
        </section>
      )}

      {!loading && (
        <button id="new-session" className="ghost new-session" onClick={() => setSheetOpen(true)}>
          <PlusIcon />
          {t("new_session")}
        </button>
      )}

      {sheetOpen && (
        <>
          <div className="sheet-backdrop" onClick={() => setSheetOpen(false)} />
          <div className="bottom-sheet">
            <div className="sheet-handle" />
            <h2><span className="glyph">//</span>{t("start_session")}</h2>
            <div className="row">
              <input placeholder={t("session_name_ph")} value={name} onChange={(e) => setName(e.target.value)} />
              <input placeholder={t("session_prompt_ph")} value={prompt} onChange={(e) => setPrompt(e.target.value)} />
            </div>
            <div className="row">
              <select value={cli} onChange={(e) => setCli(e.target.value)}>
                <option value="claude">Claude Code</option>
                <option value="kimi">Kimi Code</option>
                <option value="codex">Codex</option>
              </select>
              <input
                placeholder={t("session_cwd_ph")}
                value={cwd}
                list="cwd-history"
                onChange={(e) => setCwd(e.target.value)}
              />
              <datalist id="cwd-history">
                {cwdHistory.map((d) => <option key={d} value={d} />)}
              </datalist>
            </div>
            <button className="primary sheet-start" disabled={busy} onClick={() => void create()}>
              {busy ? t("starting_session") : t("start_session")}
            </button>
          </div>
        </>
      )}
    </>
  );
}
