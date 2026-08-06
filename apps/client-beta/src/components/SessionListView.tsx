import { useCallback, useEffect, useRef, useState } from "react";
import { api } from "../lib/store";
import type { SessionInfo } from "../lib/types";

interface Props {
  onOpen(sid: string, name: string): void;
  onLogout?: () => void;
}

export default function SessionListView({ onOpen, onLogout }: Props) {
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
    if (!res.ok) throw new Error(data.error || "启动失败");
    setName("");
    setPrompt("");
    setCwd("");
    setCli("claude");
    await refresh();
    onOpen(data.id, data.name);
  }

  return (
    <>
      <section className="card">
        <div className="row">
          <input placeholder="名称（可选）" value={name} onChange={(e) => setName(e.target.value)} />
          <input placeholder="初始指令（可选）" value={prompt} onChange={(e) => setPrompt(e.target.value)} />
        </div>
        <div className="row">
          <select value={cli} onChange={(e) => setCli(e.target.value)}>
            <option value="claude">Claude Code</option>
            <option value="kimi">Kimi Code</option>
            <option value="codex">Codex</option>
          </select>
          <input placeholder="工作目录（默认 daemon 当前目录）" value={cwd} onChange={(e) => setCwd(e.target.value)} />
        </div>
        <div className="row">
          <button className="primary" onClick={() => void create().catch((e) => alert(e instanceof Error ? e.message : String(e)))}>
            启动会话
          </button>
          <button className="ghost" onClick={() => void refresh()}>刷新</button>
          {onLogout && <button className="ghost" onClick={onLogout}>退出登录</button>}
        </div>
      </section>
      <ul id="session-list">
        {sessions.map((s) => (
          <li key={s.id} onClick={() => onOpen(s.id, s.name || s.id)}>
            <span
              className="truncate"
              title={`${s.cli || ""} ${s.cwd || ""} ${s.id}`}
            >
              {s.name ||
                [s.cli, s.cwd?.split("/").filter(Boolean).pop(), s.id.slice(0, 8)]
                  .filter(Boolean)
                  .join(" · ")}
            </span>
            <span className={"status " + (s.status || "")}>{s.status || ""}</span>
          </li>
        ))}
        {sessions.length === 0 && <li className="muted">暂无会话</li>}
      </ul>
    </>
  );
}
