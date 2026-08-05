import { useCallback, useEffect, useState } from "react";
import AuthView from "./components/AuthView";
import PairView from "./components/PairView";
import SessionDetailView from "./components/SessionDetailView";
import SessionListView from "./components/SessionListView";
import { api, isRelay, relayStore } from "./lib/store";
import { ensureIdentity } from "./lib/device";

type Phase = "loading" | "auth" | "pair" | "sessions";

export default function App() {
  const [phase, setPhase] = useState<Phase>("loading");
  const [conn, setConn] = useState("离线");
  const [openSession, setOpenSession] = useState<{ sid: string; name: string } | null>(null);

  const boot = useCallback(async () => {
    if (isRelay) {
      const rel = relayStore.get();
      if (rel?.token) {
        const res = await api("/api/auth/me");
        if (res.ok) {
          const dev = await ensureIdentity();
          setPhase(dev.deviceId ? "sessions" : "pair");
          return;
        }
        relayStore.clear();
      }
      setPhase("auth");
      return;
    }
    const dev = await ensureIdentity();
    setPhase(dev.deviceId ? "sessions" : "pair");
  }, []);

  useEffect(() => {
    boot().catch((e) => alert(e instanceof Error ? e.message : String(e)));
  }, [boot]);

  useEffect(() => {
    if (isRelay || openSession) return;
    let alive = true;
    const tick = async () => {
      try {
        const res = await fetch("/api/status");
        if (!alive) return;
        setConn(res.ok ? "服务在线" : "离线");
      } catch {
        if (alive) setConn("离线");
      }
    };
    void tick();
    const timer = setInterval(tick, 5000);
    return () => {
      alive = false;
      clearInterval(timer);
    };
  }, [openSession, phase]);

  return (
    <>
      <header>
        <h1>Riffpad</h1>
        <span id="conn" className="muted">{conn}</span>
      </header>
      <main>
        {phase === "loading" && null}
        {phase === "auth" && (
          <AuthView
            onAuthed={() => {
              setPhase("sessions");
            }}
          />
        )}
        {phase === "pair" && (
          <PairView
            onPaired={() => {
              setPhase("sessions");
            }}
          />
        )}
        {phase === "sessions" && !openSession && (
          <SessionListView
            onOpen={(sid, name) => {
              setConn("离线");
              setOpenSession({ sid, name });
            }}
            onLogout={
              isRelay
                ? async () => {
                    try {
                      await api("/api/auth/logout", { method: "POST" });
                    } catch {
                      // ignore
                    }
                    relayStore.clear();
                    setPhase("auth");
                  }
                : undefined
            }
          />
        )}
        {openSession && (
          <SessionDetailView
            sid={openSession.sid}
            name={openSession.name}
            onConn={setConn}
            onLeave={() => {
              setConn("离线");
              setOpenSession(null);
            }}
          />
        )}
      </main>
    </>
  );
}
