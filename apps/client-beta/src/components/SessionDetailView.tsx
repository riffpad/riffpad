import { useEffect, useRef, useState } from "react";
import { ensureIdentity } from "../lib/device";
import { openSessionSocket, type SessionSocket } from "../lib/sessionSocket";
import { useI18n } from "../lib/i18n";
import type { RiffpadEvent } from "../lib/types";
import EventItem from "./EventItem";

interface Props {
  sid: string;
  name: string;
  onConn(label: string): void;
  onLeave(): void;
}

export default function SessionDetailView({ sid, name, onConn, onLeave }: Props) {
  const { t } = useI18n();
  const [events, setEvents] = useState<RiffpadEvent[]>([]);
  const [prompt, setPrompt] = useState("");
  const [stopping, setStopping] = useState(false);
  const [err, setErr] = useState("");
  const sockRef = useRef<SessionSocket | null>(null);
  const listRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    let cancelled = false;
    setEvents([]);
    onConn(t("connecting"));
    ensureIdentity()
      .then((dev) => {
        if (!dev.deviceId) {
          onConn(t("not_paired"));
          return null;
        }
        return openSessionSocket(sid, dev, {
          onConn,
          onEvent: (ev) => {
            if (!cancelled) setEvents((prev) => [...prev, ev]);
          },
          onError: (message) => onConn(t("handshake_failed") + message),
        });
      })
      .then((sock) => {
        if (cancelled) {
          sock?.close();
          return;
        }
        sockRef.current = sock;
      })
      .catch((e) => onConn(t("connect_failed") + (e instanceof Error ? e.message : String(e))));
    return () => {
      cancelled = true;
      sockRef.current?.close();
      sockRef.current = null;
    };
  }, [sid, onConn, t]);

  useEffect(() => {
    const el = listRef.current;
    if (el) el.scrollTop = el.scrollHeight;
  }, [events]);

  async function sendPrompt() {
    const text = prompt.trim();
    if (!text || !sockRef.current) return;
    setPrompt("");
    const sent = await sockRef.current.send("prompt", { text });
    setErr(sent ? "" : t("send_failed"));
  }

  async function stop() {
    if (!window.confirm(t("confirm_stop"))) return;
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
    <section id="detail" className="card detail-card">
      <div className="detail-head">
        <button className="ghost back" onClick={onLeave}>← {t("back")}</button>
        <div className="detail-title truncate">
          <span className="detail-name truncate">{name || t("session_default")}</span>
          <span className="detail-id">{sid.slice(0, 8)}</span>
        </div>
        <button className="danger" onClick={() => void stop()} disabled={stopping}>
          {stopping ? t("stopping") : t("stop")}
        </button>
      </div>
      <div id="events" ref={listRef} className="events">
        {events.map((ev, i) => (
          <EventItem
            key={ev.id + "-" + i}
            ev={ev}
            send={(type, payload) => sockRef.current ? sockRef.current.send(type, payload) : Promise.resolve(false)}
          />
        ))}
        {events.length === 0 && <div className="events-empty muted">{t("waiting_events")}</div>}
      </div>
      <form
        className="prompt-form"
        onSubmit={(e) => {
          e.preventDefault();
          void sendPrompt();
        }}
      >
        <input
          placeholder={t("prompt_ph")}
          value={prompt}
          onChange={(e) => setPrompt(e.target.value)}
        />
        <button type="submit" className="primary">{t("send")}</button>
      </form>
      {err && <div className="err">{err}</div>}
    </section>
  );
}
