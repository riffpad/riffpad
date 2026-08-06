import { useEffect, useState } from "react";
import { api, isRelay, relayStore } from "../lib/store";
import { useI18n } from "../lib/i18n";

export default function AuthView({ onAuthed }: { onAuthed: () => void }) {
  const { t } = useI18n();
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [err, setErr] = useState("");

  useEffect(() => {
    function onMessage(e: MessageEvent) {
      if (e.origin !== "https://api.riffpad.ai") return;
      const d = e.data;
      if (d?.type === "riffpad-oauth" && d.token) {
        relayStore.set({ token: d.token, username: d.user || "" });
        onAuthed();
      }
    }
    window.addEventListener("message", onMessage);
    return () => window.removeEventListener("message", onMessage);
  }, [onAuthed]);

  async function submit(path: string) {
    setErr("");
    try {
      const res = await api(path, {
        method: "POST",
        body: JSON.stringify({ username: username.trim(), password }),
      });
      const data = await res.json();
      if (!res.ok) throw new Error(data.error || t("request_failed"));
      localStorage.setItem("riffpad.relay", JSON.stringify({ token: data.token, username: data.user.username }));
      setPassword("");
      onAuthed();
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    }
  }

  return (
    <section id="auth-view" className="card">
      <h2><span className="glyph">//</span>{t("login_title")}</h2>
      {isRelay && (
        <>
          <button
            className="primary github"
            style={{ width: "100%" }}
            onClick={() => window.open("/api/auth/github/login", "_blank", "width=560,height=680")}
          >
            {t("github_login")}
          </button>
          <p className="muted divider">{t("or")}</p>
        </>
      )}
      <div className="row">
        <input
          placeholder={t("username_ph")}
          autoComplete="username"
          value={username}
          onChange={(e) => setUsername(e.target.value)}
        />
        <input
          type="password"
          placeholder={t("password_ph")}
          autoComplete="current-password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
        />
      </div>
      <div className="row">
        <button className="primary" onClick={() => void submit("/api/auth/login")}>{t("login_btn")}</button>
        <button className="ghost" onClick={() => void submit("/api/auth/register")}>{t("register_btn")}</button>
      </div>
      {err && <div id="auth-err" className="err">{err}</div>}
    </section>
  );
}
