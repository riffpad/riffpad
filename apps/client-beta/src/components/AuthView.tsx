import { useState } from "react";
import { api } from "../lib/store";

export default function AuthView({ onAuthed }: { onAuthed: () => void }) {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [err, setErr] = useState("");

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
