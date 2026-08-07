import { useEffect } from "react";
import { relayStore } from "../lib/store";
import { useI18n } from "../lib/i18n";

function GitHubIcon() {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
      <path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0 0 24 12c0-6.63-5.37-12-12-12z" />
    </svg>
  );
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

// isAllowedOrigin mirrors the relay's opener allowlist: production app/api
// origins plus loopback (local dev servers).
export function isAllowedOrigin(origin: string): boolean {
  try {
    const u = new URL(origin);
    if (u.protocol === "https:" && (u.host === "app.riffpad.ai" || u.host === "api.riffpad.ai")) return true;
    if (u.protocol === "http:" && (u.hostname === "localhost" || u.hostname === "127.0.0.1")) return true;
  } catch {
    // ignore malformed origins
  }
  return false;
}

interface Props {
  onAuthed(): void;
  theme: "light" | "dark";
  onToggleTheme(): void;
}

export default function AuthView({ onAuthed, theme, onToggleTheme }: Props) {
  const { t, lang, setLang } = useI18n();

  useEffect(() => {
    function onMessage(e: MessageEvent) {
      if (!isAllowedOrigin(e.origin)) return;
      const d = e.data;
      if (d?.type === "riffpad-oauth" && d.token) {
        relayStore.set({ token: d.token, username: d.user || "" });
        onAuthed();
      }
    }
    window.addEventListener("message", onMessage);
    return () => window.removeEventListener("message", onMessage);
  }, [onAuthed]);

  return (
    <section id="auth-view" className="card auth-card">
      <button
        id="github-login"
        className="auth-github github"
        style={{ width: "100%" }}
        onClick={() => window.open("/api/auth/github/login?opener=" + encodeURIComponent(location.origin), "_blank", "width=560,height=680")}
      >
        <GitHubIcon />
        {t("github_login")}
      </button>
      <div className="auth-toggles">
        <button
          id="theme-toggle"
          className="icon-btn"
          onClick={onToggleTheme}
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
    </section>
  );
}
