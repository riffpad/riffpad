import { useCallback, useEffect, useRef, useState } from "react";
import { api, isRelay } from "../lib/store";
import { applySessionMeta, updateSessionMeta } from "../lib/sessionMeta";
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

// sortSessions returns a stable, meaningful order for the session list:
// most recently active first, missing/zero timestamps last, id as a
// tiebreaker. Backends range over Go maps, so the raw API order is random
// and would reshuffle the list on every 5s poll (#249).
function sortSessions(list: SessionInfo[]): SessionInfo[] {
  return [...list].sort((a, b) => {
    const ta = new Date(a.lastSeenAt || 0).getTime();
    const tb = new Date(b.lastSeenAt || 0).getTime();
    if (ta !== tb) return tb - ta;
    return a.id < b.id ? -1 : a.id > b.id ? 1 : 0;
  });
}

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

function MenuIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
      <circle cx="5" cy="12" r="1.6" />
      <circle cx="12" cy="12" r="1.6" />
      <circle cx="19" cy="12" r="1.6" />
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
  const [offline, setOffline] = useState(false);
  // relay mode: the request succeeded but no host is connected — the list is
  // empty because the daemon is offline, not because there are no sessions
  // (#174). Local mode never sets this: a 200 there means the daemon is up.
  const [hostOffline, setHostOffline] = useState(false);
  const timer = useRef<number | null>(null);
  // Session management: menu / rename / delete bottom sheets. These are
  // client-view operations — the host agent keeps running untouched (#251).
  const [action, setAction] = useState<{ session: SessionInfo; mode: "menu" | "rename" | "delete" } | null>(null);
  const [renameText, setRenameText] = useState("");
  const [metaBusy, setMetaBusy] = useState(false);
  const [metaErr, setMetaErr] = useState("");

  const refresh = useCallback(async () => {
    try {
      const res = await api("/api/sessions");
      const data = await res.json();
      setSessions(sortSessions(applySessionMeta(data.sessions || [])));
      setOffline(!res.ok);
      if (res.ok) setHostOffline(isRelay && data.hostOnline === false);
    } catch {
      // transient network glitch; poll again later
      setOffline(true);
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

  async function saveRename() {
    if (!action || action.mode !== "rename") return;
    setMetaBusy(true);
    setMetaErr("");
    try {
      await updateSessionMeta(action.session, { displayName: renameText.trim() });
      setAction(null);
      await refresh();
    } catch (e) {
      setMetaErr(e instanceof Error ? e.message : String(e));
    } finally {
      setMetaBusy(false);
    }
  }

  async function confirmDelete() {
    if (!action || action.mode !== "delete") return;
    setMetaBusy(true);
    setMetaErr("");
    try {
      await updateSessionMeta(action.session, { hidden: true });
      setAction(null);
      await refresh();
    } catch (e) {
      setMetaErr(e instanceof Error ? e.message : String(e));
    } finally {
      setMetaBusy(false);
    }
  }

  const visibleSessions = sessions.filter((s) => !s.hidden);

  return (
    <>
      <p className="section-label"><span className="glyph">//</span>{t("sessions_label")}</p>
      {offline && <div id="offline-banner" className="offline-banner">■ {t("list_offline")}</div>}
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
          {visibleSessions.map((s) => {
            const dir = s.cwd?.split("/").filter(Boolean).pop();
            const title = s.displayName || s.name || dir || "session-" + s.id.slice(0, 8);
            const meta = [s.cli, s.id.slice(0, 8), timeAgo(s.lastSeenAt, t)].filter(Boolean).join(" · ");
            const tone = statusTone(s.status);
            return (
              <li key={s.id} className="session" onClick={() => onOpen(s.id, s.displayName || s.name || "", s.cli, s.cwd)}>
                <div className="session-main">
                  <span className="session-name truncate">
                    {title}
                  </span>
                  <span className="session-meta truncate" title={`${s.cwd || ""} ${s.id}`}>{meta}</span>
                </div>
                <span className={"session-light " + tone}><span className="dot" />{statusLabel(s.status)}</span>
                <button
                  className="session-menu-btn"
                  aria-label={t("session_actions")}
                  onClick={(e) => {
                    e.stopPropagation();
                    setRenameText(s.displayName || s.name || "");
                    setMetaErr("");
                    setAction({ session: s, mode: "menu" });
                  }}
                >
                  <MenuIcon />
                </button>
              </li>
            );
          })}
          {visibleSessions.length === 0 && <li className="empty muted">{t(hostOffline ? "offline_title" : "no_sessions")}</li>}
        </ul>
      )}

      {!loading && visibleSessions.length === 0 && hostOffline && (
        <section className="card empty-card">
          <h3><span className="glyph">//</span>{t("offline_title")}</h3>
          <p className="muted">{t("offline_hint")}</p>
        </section>
      )}

      {!loading && visibleSessions.length === 0 && !hostOffline && (
        <section className="card empty-card">
          <h3><span className="glyph">//</span>{t("empty_title")}</h3>
          <p className="muted">{t("empty_run_hint")}</p>
          <code className="empty-cmd">riffpad run codex</code>
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

      {action?.mode === "menu" && (
        <>
          <div className="sheet-backdrop" onClick={() => setAction(null)} />
          <div className="bottom-sheet">
            <div className="sheet-handle" />
            <h2>{action.session.displayName || action.session.name || action.session.cwd?.split("/").filter(Boolean).pop() || t("session_default")}</h2>
            <div className="sheet-actions">
              <button className="ghost" onClick={() => setAction({ session: action.session, mode: "rename" })}>
                {t("session_rename")}
              </button>
              <button className="ghost-danger" onClick={() => setAction({ session: action.session, mode: "delete" })}>
                {t("session_delete")}
              </button>
            </div>
          </div>
        </>
      )}

      {action?.mode === "rename" && (
        <>
          <div className="sheet-backdrop" onClick={() => setAction(null)} />
          <div className="bottom-sheet">
            <div className="sheet-handle" />
            <h2>{t("rename_title")}</h2>
            <form
              onSubmit={(e) => {
                e.preventDefault();
                void saveRename();
              }}
            >
              <input
                value={renameText}
                onChange={(e) => setRenameText(e.target.value)}
                placeholder={t("rename_ph")}
                autoFocus
              />
              <button className="primary sheet-start" disabled={metaBusy} type="submit">
                {metaBusy ? t("saving") : t("rename_save")}
              </button>
            </form>
            {metaErr && <div className="err">{metaErr}</div>}
          </div>
        </>
      )}

      {action?.mode === "delete" && (
        <>
          <div className="sheet-backdrop" onClick={() => setAction(null)} />
          <div className="bottom-sheet">
            <div className="sheet-handle" />
            <h2>{t("delete_title")}</h2>
            <p className="muted">{t("delete_hint")}</p>
            <button className="danger sheet-start" disabled={metaBusy} onClick={() => void confirmDelete()}>
              {metaBusy ? t("deleting") : t("session_delete")}
            </button>
            {metaErr && <div className="err">{metaErr}</div>}
          </div>
        </>
      )}
    </>
  );
}
