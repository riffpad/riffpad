import { useCallback, useEffect, useRef, useState } from "react";
import { api, isRelay } from "../lib/store";
import { useI18n } from "../lib/i18n";
import type { SessionInfo } from "../lib/types";

interface Props {
  onOpen(sid: string, name: string): void;
  onLogout?: () => void;
}

export default function SessionListView({ onOpen, onLogout }: Props) {
  const { t } = useI18n();
  const [sessions, setSessions] = useState<SessionInfo[]>([]);
  const [name, setName] = useState("");
  const [prompt, setPrompt] = useState("");
  const [cli, setCli] = useState("claude");
  const [cwd, setCwd] = useState("");
  const timer = useRef<number | null>(null);

  const refresh = useCallback(async () => {
    try {
      const res = await api("/api/sessions");
      const data = await res.json();
      setSessions(data.sessions || []);
    } catch {
      // transient network glitch; poll again later
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
    const res = await api("/api/sessions", {
      method: "POST",
      body: JSON.stringify({ name, prompt, cli, cwd }),
    });
    const data = await res.json();
    if (!res.ok) throw new Error(data.error || t("start_failed"));
    setName("");
    setPrompt("");
    setCwd("");
    setCli("claude");
    await refresh();
    onOpen(data.id, data.name);
  }

  return (
    <>
      <section className="card create-card">
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
          <input placeholder={t("session_cwd_ph")} value={cwd} onChange={(e) => setCwd(e.target.value)} />
        </div>
        <div className="row">
          <button className="primary" onClick={() => void create().catch((e) => alert(e instanceof Error ? e.message : String(e)))}>
            {t("start_session")}
          </button>
          <button className="ghost" onClick={() => void refresh()}>{t("refresh")}</button>
          {onLogout && <button className="ghost" onClick={onLogout}>{t("logout")}</button>}
        </div>
      </section>
      <p className="section-label"><span className="glyph">//</span>{t("sessions_label")}</p>
      <ul id="session-list">
        {sessions.map((s) => (
          <li key={s.id} className="session" onClick={() => onOpen(s.id, s.name || s.id)}>
            <div className="session-main">
              <span className="session-name truncate">
                {s.name ||
                  [s.cli, s.cwd?.split("/").filter(Boolean).pop(), s.id.slice(0, 8)]
                    .filter(Boolean)
                    .join(" · ")}
              </span>
              <span className="session-meta truncate" title={`${s.cli || ""} ${s.cwd || ""} ${s.id}`}>
                {[s.cli, s.cwd].filter(Boolean).join(" · ") || s.id.slice(0, 8)}
              </span>
            </div>
            <span className={"status " + (s.status || "")}>{s.status || "—"}</span>
          </li>
        ))}
        {sessions.length === 0 && <li className="empty muted">{t("no_sessions")}</li>}
      </ul>
      {sessions.length === 0 && (
        <section className="card empty-card">
          <h3><span className="glyph">//</span>{t("empty_title")}</h3>
          <ol className="steps">
            <li>
              {isRelay ? (
                <>{t("empty_step1_relay", { cmd: "curl -fsSL https://riffpad.ai/install.sh | sh" })}</>
              ) : (
                t("empty_step1_local")
              )}
            </li>
            <li>{t("empty_step2")}</li>
            <li>{t("empty_step3")}</li>
          </ol>
        </section>
      )}
    </>
  );
}
