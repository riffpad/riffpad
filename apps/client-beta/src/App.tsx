import { useCallback, useEffect, useState } from "react";
import AuthView from "./components/AuthView";
import DeviceManager from "./components/DeviceManager";
import PairView from "./components/PairView";
import SessionDetailView from "./components/SessionDetailView";
import SessionListView from "./components/SessionListView";
import { api, deviceStore, isRelay, relayStore } from "./lib/store";
import { ensureIdentity } from "./lib/device";
import type { Device } from "./lib/types";

type Phase = "loading" | "auth" | "pair" | "sessions";

async function deviceStillValid(dev: Device): Promise<boolean> {
  if (!dev.deviceId) return false;
  try {
    const res = await api("/api/devices");
    const data = await res.json();
    const list = (data.devices || []) as { id?: string }[];
    return list.some((d) => d.id === dev.deviceId);
  } catch {
    // Network hiccup: don't force re-pair; the connect error will surface.
    return true;
  }
}

export default function App() {
  const [phase, setPhase] = useState<Phase>("loading");
  const [conn, setConn] = useState("离线");
  const [openSession, setOpenSession] = useState<{ sid: string; name: string } | null>(null);

  const afterAuth = useCallback(async () => {
    const dev = await ensureIdentity();
    if (dev.deviceId && !(await deviceStillValid(dev))) {
      deviceStore.set({ ...dev, deviceId: null, serverPub: null });
      setPhase("pair");
      return;
    }
    setPhase(dev.deviceId ? "sessions" : "pair");
  }, []);

  const boot = useCallback(async () => {
    if (isRelay) {
      const rel = relayStore.get();
      if (rel?.token) {
        const res = await api("/api/auth/me");
        if (res.ok) {
          await afterAuth();
          return;
        }
        relayStore.clear();
      }
      setPhase("auth");
      return;
    }
    const dev = await ensureIdentity();
    if (dev.deviceId && !(await deviceStillValid(dev))) {
      deviceStore.set({ ...dev, deviceId: null, serverPub: null });
      setPhase("pair");
      return;
    }
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
            onAuthed={() => void afterAuth()}
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
          <>
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
            <DeviceManager />
          </>
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
