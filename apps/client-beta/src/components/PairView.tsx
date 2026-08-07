import { useCallback, useEffect, useState } from "react";
import { pairDevice } from "../lib/device";
import { useI18n } from "../lib/i18n";
import PinInput from "./PinInput";
import ScanQR from "./ScanQR";

function PairIllustration() {
  return (
    <svg className="pair-illustration" viewBox="0 0 300 120" fill="none" aria-hidden="true">
      {/* terminal window */}
      <rect x="12" y="18" width="118" height="84" stroke="var(--hairline-strong)" strokeWidth="2" />
      <line x1="12" y1="34" x2="130" y2="34" stroke="var(--hairline-strong)" strokeWidth="2" />
      <circle cx="24" cy="26" r="3" fill="var(--danger)" />
      <circle cx="36" cy="26" r="3" fill="var(--warning)" />
      <circle cx="48" cy="26" r="3" fill="var(--success)" />
      <line x1="24" y1="48" x2="108" y2="48" stroke="var(--mute)" strokeWidth="2" />
      <line x1="24" y1="60" x2="92" y2="60" stroke="var(--mute)" strokeWidth="2" />
      <line x1="24" y1="72" x2="104" y2="72" stroke="var(--accent)" strokeWidth="2" />
      <line x1="24" y1="84" x2="78" y2="84" stroke="var(--mute)" strokeWidth="2" />
      {/* phone */}
      <rect x="182" y="10" width="74" height="104" stroke="var(--hairline-strong)" strokeWidth="2" />
      <rect x="192" y="22" width="54" height="76" stroke="var(--hairline)" strokeWidth="1.5" />
      <line x1="200" y1="32" x2="238" y2="32" stroke="var(--accent)" strokeWidth="2" />
      <rect x="200" y="42" width="38" height="18" stroke="var(--hairline-strong)" strokeWidth="1.5" />
      <rect x="200" y="66" width="38" height="10" stroke="var(--hairline)" strokeWidth="1.5" />
      <line x1="200" y1="82" x2="228" y2="82" stroke="var(--hairline)" strokeWidth="1.5" />
      <line x1="216" y1="106" x2="224" y2="106" stroke="var(--hairline-strong)" strokeWidth="2" />
      {/* sync connector */}
      <path d="M130 62 H182" stroke="var(--accent)" strokeWidth="2" strokeDasharray="4 4" />
      <circle className="sync-dot" cx="156" cy="62" r="4" fill="var(--accent)" />
    </svg>
  );
}

function QrIcon() {
  return (
    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="square" aria-hidden="true">
      <rect x="3" y="3" width="7" height="7" />
      <rect x="14" y="3" width="7" height="7" />
      <rect x="3" y="14" width="7" height="7" />
      <path d="M14 14h3v3h-3zM21 14v7M14 21h3" />
    </svg>
  );
}

export default function PairView({ onPaired }: { onPaired: () => void }) {
  const { t } = useI18n();
  const [code, setCode] = useState("");
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);
  const [scanning, setScanning] = useState(false);
  const [copied, setCopied] = useState(false);

  const submit = useCallback(async (value: string) => {
    const clean = (value || "").toUpperCase().replace(/[^A-Z0-9]/g, "").slice(0, 6);
    if (clean.length !== 6) return;
    setErr("");
    setBusy(true);
    try {
      await pairDevice(clean);
      onPaired();
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
      setBusy(false);
    }
  }, [onPaired]);

  useEffect(() => {
    const p = new URLSearchParams(location.search).get("pair");
    if (p) void submit(p);
  }, [submit]);

  async function copy() {
    try {
      await navigator.clipboard.writeText("riffpad pair");
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1500);
    } catch {
      // clipboard unavailable (non-secure context); ignore
    }
  }

  return (
    <>
      <section id="pair-view" className="pair-view">
        <div className="pair-hero">
          <PairIllustration />
          <h2>{t("pair_hero_title")}</h2>
          <p className="muted">{t("pair_hero_subtitle")}</p>
        </div>

        <button id="scan-btn" className="primary scan-btn" onClick={() => setScanning(true)}>
          <QrIcon />
          {t("scan_qr")}
        </button>

        <div className="divider muted">{t("or_manual")}</div>

        <PinInput
          value={code}
          onChange={setCode}
          onComplete={(v) => void submit(v)}
          disabled={busy}
          autoFocus
        />
        <div className="pair-actions">
          <button className="ghost" disabled={busy || code.length !== 6} onClick={() => void submit(code)}>
            {busy ? t("pairing") : t("pair_btn")}
          </button>
        </div>

        <div className="code-card">
          <code>riffpad pair</code>
          <button id="copy-btn" className="ghost" onClick={() => void copy()}>
            {copied ? t("copied") : t("copy_command")}
          </button>
        </div>
        <p className="muted pair-hint">{t("pair_hint")}</p>
        {err && <div id="pair-err" className="err">{err}</div>}
      </section>
      {scanning && (
        <ScanQR
          onClose={() => setScanning(false)}
          onCode={(c) => {
            setScanning(false);
            void submit(c);
          }}
        />
      )}
    </>
  );
}
