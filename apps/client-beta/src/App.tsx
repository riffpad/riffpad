import { useCallback, useEffect, useState } from "react";
import AuthView from "./components/AuthView";
import DeviceAuthView from "./components/DeviceAuthView";
import DeviceManager from "./components/DeviceManager";
import Logo from "./components/Logo";
import PairView from "./components/PairView";
import SessionDetailView from "./components/SessionDetailView";
import SessionListView from "./components/SessionListView";
import { api, deviceStore, isRelay, relayStore } from "./lib/store";
import { ensureIdentity } from "./lib/device";
import { useI18n } from "./lib/i18n";
import type { Device } from "./lib/types";

type Phase = "loading" | "auth" | "pair" | "sessions";
type Theme = "light" | "dark";

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

function initTheme(): Theme {
  const saved = localStorage.getItem("riffpad.theme");
  if (saved === "light" || saved === "dark") return saved;
  return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
}

function SunIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" aria-hidden="true">
      <circle cx="12" cy="12" r="4" />
      <path d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M4.93 19.07l1.41-1.41M17.66 6.34l1.41-1.41" />
    </svg>
  );
}

function MoonIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z" />
    </svg>
  );
}

function MenuIcon() {
  return (
    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" aria-hidden="true">
      <path d="M4 7h16M4 12h16M4 17h16" />
    </svg>
  );
}

export default function App() {
  const { t, lang, setLang } = useI18n();
  const [phase, setPhase] = useState<Phase>("loading");
  const [theme, setTheme] = useState<Theme>(initTheme);
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [paired, setPaired] = useState<boolean | null>(null);
  const [openSession, setOpenSession] = useState<{ sid: string; name: string } | null>(null);
  const isDevicePage = window.location.pathname.startsWith("/device");

  useEffect(() => {
    document.documentElement.dataset.theme = theme;
    localStorage.setItem("riffpad.theme", theme);
  }, [theme]);

  useEffect(() => {
    const onHash = () => setSidebarOpen(false);
    window.addEventListener("hashchange", onHash);
    return () => window.removeEventListener("hashchange", onHash);
  }, []);

  const afterAuth = useCallback(async () => {
    const dev = await ensureIdentity();
    if (dev.deviceId && !(await deviceStillValid(dev))) {
      deviceStore.set({ ...dev, deviceId: null, serverPub: null });
      setPaired(false);
      setPhase("pair");
      return;
    }
    setPaired(!!dev.deviceId);
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
      setPaired(false);
      setPhase("auth");
      return;
    }
    const dev = await ensureIdentity();
    if (dev.deviceId && !(await deviceStillValid(dev))) {
      deviceStore.set({ ...dev, deviceId: null, serverPub: null });
      setPaired(false);
      setPhase("pair");
      return;
    }
    setPaired(!!dev.deviceId);
    setPhase(dev.deviceId ? "sessions" : "pair");
  }, []);

  const logout = useCallback(async () => {
    if (!isRelay) return;
    try {
      await api("/api/auth/logout", { method: "POST" });
    } catch {
      // ignore
    }
    relayStore.clear();
    setPaired(false);
    setSidebarOpen(false);
    setPhase("auth");
  }, []);

  useEffect(() => {
    boot().catch((e) => alert(e instanceof Error ? e.message : String(e)));
  }, [boot]);

  const topbar = (
    <header id="topbar" className={"topbar" + (sidebarOpen ? " open" : "")}>
      <div className="brand">
        <Logo />
        <span className="brand-name">riffpad</span>
      </div>
      <div className="topbar-actions">
        {phase !== "loading" && (
          <span id="conn" className={"conn " + (paired ? "good" : "bad")}>
            {paired ? t("paired") : t("unpaired")}
          </span>
        )}
        <button
          id="theme-toggle"
          className="icon-btn"
          onClick={() => setTheme(theme === "dark" ? "light" : "dark")}
          title={theme === "dark" ? t("theme_light") : t("theme_dark")}
          aria-label={theme === "dark" ? t("theme_light") : t("theme_dark")}
        >
          {theme === "dark" ? <SunIcon /> : <MoonIcon />}
        </button>
        <button
          id="lang-toggle"
          className="lang-toggle"
          onClick={() => setLang(lang === "zh" ? "en" : "zh")}
          title={lang === "zh" ? "English" : "中文"}
        >
          {t("lang_toggle")}
        </button>
        {isRelay && phase === "sessions" && (
          <button id="logout-btn" className="lang-toggle" onClick={() => void logout()}>
            {t("logout")}
          </button>
        )}
      </div>
      <button id="sidebar-close" className="icon-btn sidebar-close" onClick={() => setSidebarOpen(false)} aria-label={t("sidebar_close")}>
        ×
      </button>
    </header>
  );

  if (isDevicePage) {
    return (
      <>
        <header className="topbar topbar-static">
          <div className="brand">
            <Logo />
            <span className="brand-name">riffpad</span>
          </div>
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
      <button id="menu-toggle" className="icon-btn menu-toggle" onClick={() => setSidebarOpen(true)} aria-label={t("sidebar_open")}>
        <MenuIcon />
      </button>
      {topbar}
      {sidebarOpen && <div id="sidebar-backdrop" className="sidebar-backdrop" onClick={() => setSidebarOpen(false)} />}
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
              setPaired(true);
              setPhase("sessions");
            }}
          />
        )}
        {phase === "sessions" && !openSession && (
          <>
            <SessionListView
              onOpen={(sid, name) => {
                setSidebarOpen(false);
                setOpenSession({ sid, name });
              }}
            />
            <DeviceManager />
          </>
        )}
        {openSession && (
          <SessionDetailView
            sid={openSession.sid}
            name={openSession.name}
            onLeave={() => {
              setOpenSession(null);
            }}
          />
        )}
      </main>
    </>
  );
}
