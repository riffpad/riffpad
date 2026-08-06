import { useState } from "react";
import { useI18n } from "../lib/i18n";

export default function DeviceAuthView() {
  const { t } = useI18n();
  const code = new URLSearchParams(window.location.search).get("code") || "";
  const [err, setErr] = useState("");

  return (
    <section id="device-auth-view" className="card">
      <h2><span className="glyph">//</span>{t("cli_auth_title")}</h2>
      <p className="muted">{t("cli_auth_desc", { cmd: "riffpad relay login --github" })}</p>
      <p className="pair-code">{t("auth_code")}<b>{code || "—"}</b></p>
      <button
        className="primary github"
        style={{ width: "100%" }}
        disabled={!code}
        onClick={() => {
          if (!code) return;
          setErr("");
          window.location.href = "/api/auth/github/login?device=" + encodeURIComponent(code);
        }}
      >
        {t("github_login")}
      </button>
      {!code && <div className="err">{t("cli_auth_missing")}</div>}
      {err && <div className="err">{err}</div>}
    </section>
  );
}
