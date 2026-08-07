import { useCallback, useEffect, useState } from "react";
import AuthView from "./components/AuthView";
import DeviceAuthView from "./components/DeviceAuthView";
import DeviceManager from "./components/DeviceManager";
import PairView from "./components/PairView";
import SessionDetailView from "./components/SessionDetailView";
import SessionListView from "./components/SessionListView";
import { api, deviceStore, isRelay, relayStore } from "./lib/store";
import { ensureIdentity } from "./lib/device";
import { useI18n } from "./lib/i18n";
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

function connClass(conn: string): string {
  if (/离线|未连接|未配对|握手失败|连接失败|失败/.test(conn)) return "bad";
  if (/连接中|等待/.test(conn)) return "pending";
  return "good";
}

export default function App() {
  const { t, lang, setLang } = useI18n();
  const [phase, setPhase] = useState<Phase>("loading");
  const [conn, setConn] = useState(t("offline"));
  const [openSession, setOpenSession] = useState<{ sid: string; name: string } | null>(null);
  const isDevicePage = window.location.pathname.startsWith("/device");

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
    if (window.location.pathname.startsWith("/device")) return;
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
    if (isRelay || openSession || window.location.pathname.startsWith("/device")) return;
    let alive = true;
    const tick = async () => {
      try {
        const res = await fetch("/api/status");
        if (!alive) return;
        setConn(res.ok ? t("online") : t("offline"));
      } catch {
        if (alive) setConn(t("offline"));
      }
    };
    void tick();
    const timer = setInterval(tick, 5000);
    return () => {
      alive = false;
      clearInterval(timer);
    };
  }, [openSession, phase, t]);

  if (isDevicePage) {
    return (
      <>
        <header>
          <h1>riffpad</h1>
          <span className="muted">{t("cli_auth")}</span>
        </header>
        <main>
          <DeviceAuthView />
        </main>
      </>
    );
  }

  return (
    <>
      <header>
        <h1>riffpad</h1>
        <span id="conn" className={"conn " + connClass(conn)}>{conn}</span>
        <button
          id="lang-toggle"
          className="lang-toggle"
          onClick={() => setLang(lang === "zh" ? "en" : "zh")}
          title={lang === "zh" ? "English" : "中文"}
        >
          {t("lang_toggle")}
        </button>
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
                setConn(t("offline"));
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
              setConn(t("offline"));
              setOpenSession(null);
            }}
          />
        )}
      </main>
    </>
  );
}
