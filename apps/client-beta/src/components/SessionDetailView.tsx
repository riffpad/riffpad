import { useEffect, useRef, useState } from "react";
import { ensureIdentity } from "../lib/device";
import { openSessionSocket, type SessionSocket } from "../lib/sessionSocket";
import type { RiffpadEvent } from "../lib/types";
import EventItem from "./EventItem";

interface Props {
  sid: string;
  name: string;
  onConn(label: string): void;
  onLeave(): void;
}

export default function SessionDetailView({ sid, name, onConn, onLeave }: Props) {
  const [events, setEvents] = useState<RiffpadEvent[]>([]);
  const [prompt, setPrompt] = useState("");
  const [stopping, setStopping] = useState(false);
  const [err, setErr] = useState("");
  const sockRef = useRef<SessionSocket | null>(null);
  const listRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    let cancelled = false;
    setEvents([]);
    onConn("连接中…");
    ensureIdentity()
      .then((dev) => {
        if (!dev.deviceId) {
          onConn("未配对：请刷新页面并重新输入配对码");
          return null;
        }
        return openSessionSocket(sid, dev, {
          onConn,
          onEvent: (ev) => {
            if (!cancelled) setEvents((prev) => [...prev, ev]);
          },
          onError: (message) => onConn("握手失败：" + message),
        });
      })
      .then((sock) => {
        if (cancelled) {
          sock?.close();
          return;
        }
        sockRef.current = sock;
      })
      .catch((e) => onConn("连接失败：" + (e instanceof Error ? e.message : String(e))));
    return () => {
      cancelled = true;
      sockRef.current?.close();
      sockRef.current = null;
    };
  }, [sid, onConn]);

  useEffect(() => {
    const el = listRef.current;
    if (el) el.scrollTop = el.scrollHeight;
  }, [events]);

  async function sendPrompt() {
    const text = prompt.trim();
    if (!text || !sockRef.current) return;
    setPrompt("");
    const sent = await sockRef.current.send("prompt", { text });
    setErr(sent ? "" : "未连接：设备可能已失效或会话已结束，请刷新页面重试");
  }

  async function stop() {
    if (!window.confirm("确定停止这个会话？agent 进程会被终止。")) return;
    setStopping(true);
    try {
      await fetch("/api/sessions/" + sid + "/stop", { method: "POST" });
    } catch {
      // ignore
    }
    sockRef.current?.close();
    await new Promise((r) => setTimeout(r, 200));
    setStopping(false);
    onLeave();
  }

  return (
    <section id="detail" className="card">
      <div className="row">
        <h3>{name || "会话"} · {sid.slice(0, 8)}</h3>
        <button className="danger" onClick={() => void stop()} disabled={stopping}>
          {stopping ? "停止中…" : "停止"}
        </button>
        <button className="ghost" onClick={onLeave}>返回</button>
      </div>
      <div id="events" ref={listRef} className="events">
        {events.map((ev, i) => (
          <EventItem
            key={ev.id + "-" + i}
            ev={ev}
            send={(type, payload) => sockRef.current ? sockRef.current.send(type, payload) : Promise.resolve(false)}
          />
        ))}
        {events.length === 0 && <div className="muted">等待事件…</div>}
      </div>
      <form
        onSubmit={(e) => {
          e.preventDefault();
          void sendPrompt();
        }}
      >
        <div className="row">
          <input
            placeholder="下达指令…"
            value={prompt}
            onChange={(e) => setPrompt(e.target.value)}
          />
          <button type="submit" className="primary">发送</button>
        </div>
      </form>
      {err && <div className="err">{err}</div>}
    </section>
  );
}
