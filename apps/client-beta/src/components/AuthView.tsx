import { useEffect, useState } from "react";
import { api, isRelay, relayStore } from "../lib/store";

export default function AuthView({ onAuthed }: { onAuthed: () => void }) {
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
      if (!res.ok) throw new Error(data.error || "请求失败");
      localStorage.setItem("riffpad.relay", JSON.stringify({ token: data.token, username: data.user.username }));
      setPassword("");
      onAuthed();
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    }
  }

  return (
    <section id="auth-view" className="card">
      <h2>登录 / 注册</h2>
      {isRelay && (
        <>
          <button
            className="primary"
            style={{ width: "100%" }}
            onClick={() => window.open("/api/auth/github/login", "_blank", "width=560,height=680")}
          >
            使用 GitHub 登录
          </button>
          <p className="muted" style={{ textAlign: "center" }}>或</p>
        </>
      )}
      <div className="row">
        <input
          placeholder="用户名"
          autoComplete="username"
          value={username}
          onChange={(e) => setUsername(e.target.value)}
        />
        <input
          type="password"
          placeholder="密码"
          autoComplete="current-password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
        />
      </div>
      <div className="row">
        <button className="primary" onClick={() => void submit("/api/auth/login")}>登录</button>
        <button className="ghost" onClick={() => void submit("/api/auth/register")}>注册</button>
      </div>
      {err && <div id="auth-err" className="err">{err}</div>}
    </section>
  );
}
