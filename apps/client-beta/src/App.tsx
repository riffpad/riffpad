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
  const [page, setPage] = useState<"sessions" | "devices">("sessions");
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

  const logout = useCallback(async () => {
    if (!isRelay) return;
    try {
      await api("/api/auth/logout", { method: "POST" });
    } catch {
      // ignore
    }
    relayStore.clear();
    setSidebarOpen(false);
    setPhase("auth");
  }, []);

  const handleCurrentRevoked = useCallback(() => {
    const dev = deviceStore.get();
    if (dev) deviceStore.set({ ...dev, deviceId: null, serverPub: null });
    setOpenSession(null);
    setPage("sessions");
    setSidebarOpen(false);
    setPhase("pair");
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
      {isRelay && (() => {
        const username = relayStore.get()?.username || "";
        if (!username) return null;
        return (
          <div className="sidebar-user">
            <img
              className="sidebar-avatar"
              src={`https://github.com/${encodeURIComponent(username)}.png?size=96`}
              alt=""
              onError={(e) => {
                e.currentTarget.style.display = "none";
              }}
            />
            <div className="sidebar-user-id">
              <span className="sidebar-user-name truncate">@{username}</span>
              <span className="sidebar-user-tag">GitHub</span>
            </div>
          </div>
        );
      })()}
      <nav className="topbar-nav">
        <button
          className={"nav-item" + (page === "sessions" ? " active" : "")}
          disabled={phase !== "sessions"}
          onClick={() => {
            setOpenSession(null);
            setPage("sessions");
            setSidebarOpen(false);
          }}
        >
          {t("nav_sessions")}
        </button>
        <button
          className={"nav-item" + (page === "devices" ? " active" : "")}
          disabled={phase !== "sessions"}
          onClick={() => {
            setOpenSession(null);
            setPage("devices");
            setSidebarOpen(false);
          }}
        >
          {t("nav_devices")}
        </button>
      </nav>
      <div className="topbar-actions">
        {isRelay && phase === "sessions" && (
          <button id="logout-btn" className="lang-toggle" onClick={() => void logout()}>
            {t("logout")}
          </button>
        )}
        <div className="sidebar-toggles">
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
        </div>
      </div>
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

  if (phase === "auth") {
    return (
      <main className="auth-stage">
        <AuthView
          onAuthed={() => void afterAuth()}
          theme={theme}
          onToggleTheme={() => setTheme(theme === "dark" ? "light" : "dark")}
        />
      </main>
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
        {phase === "pair" && (
            <PairView
            onPaired={() => {
              setPhase("sessions");
            }}
          />
        )}
        {phase === "sessions" && !openSession && (
          page === "sessions" ? (
            <SessionListView
              onOpen={(sid, name) => {
                setSidebarOpen(false);
                setOpenSession({ sid, name });
              }}
            />
          ) : (
            <DeviceManager onCurrentRevoked={handleCurrentRevoked} />
          )
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
