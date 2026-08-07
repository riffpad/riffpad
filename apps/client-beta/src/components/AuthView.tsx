import { useEffect, useState } from "react";
import { relayStore } from "../lib/store";
import { useI18n } from "../lib/i18n";

function GitHubIcon() {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
      <path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0 0 24 12c0-6.63-5.37-12-12-12z" />
    </svg>
  );
}

function CopyIcon() {
  return (
    <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="square" aria-hidden="true">
      <rect x="9" y="9" width="12" height="12" />
      <path d="M5 15H3V3h12v2" />
    </svg>
  );
}

function CheckIcon() {
  return (
    <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="square" aria-hidden="true">
      <path d="M4 12l5 5L20 6" />
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

const INSTALL_CMD = "curl -fsSL https://riffpad.ai/install.sh | sh";

export default function AuthView({ onAuthed }: { onAuthed: () => void }) {
  const { t } = useI18n();
  const [cloud, setCloud] = useState<"checking" | "online" | "offline">("checking");
  const [copied, setCopied] = useState(false);

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

  useEffect(() => {
    let alive = true;
    fetch("/api/status")
      .then((r) => {
        if (alive) setCloud(r.ok ? "online" : "offline");
      })
      .catch(() => {
        if (alive) setCloud("offline");
      });
    return () => {
      alive = false;
    };
  }, []);

  async function copyInstall() {
    try {
      await navigator.clipboard.writeText(INSTALL_CMD);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1500);
    } catch {
      // clipboard unavailable; ignore
    }
  }

  const cloudLabel = cloud === "online" ? t("cloud_online") : cloud === "offline" ? t("cloud_offline") : t("cloud_checking");

  return (
    <>
      <section id="auth-view" className="card auth-card">
        <div className="auth-terminal">{t("auth_terminal")}</div>
        <div id="cloud-status" className={"auth-status " + cloud}>
          <span className="dot" />
          {cloudLabel}
        </div>
        <h2><span className="glyph">//</span>{t("auth_required")}</h2>
        <p className="muted auth-desc">{t("auth_desc")}</p>
        <button
          id="github-login"
          className="auth-github github"
          style={{ width: "100%" }}
          onClick={() => window.open("/api/auth/github/login?opener=" + encodeURIComponent(location.origin), "_blank", "width=560,height=680")}
        >
          <GitHubIcon />
          {t("github_login")}
        </button>
        <p className="muted install-label">{t("cli_install_label")}</p>
        <div className="install-command">
          <code><span className="prompt">$</span> {INSTALL_CMD}</code>
          <button id="install-copy" className="icon-btn" onClick={() => void copyInstall()} aria-label={t("copy_command")}>
            {copied ? <CheckIcon /> : <CopyIcon />}
          </button>
        </div>
      </section>
      <a className="back-home" href="https://riffpad.ai">{t("back_home")}</a>
    </>
  );
}
