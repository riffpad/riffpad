import { useCallback, useEffect, useRef, useState } from "react";
import { pairDevice, PairingCodeUsedError } from "../lib/device";
import { useI18n } from "../lib/i18n";
import DotMatrix from "./DotMatrix";
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

function ScanIcon() {
  return (
    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="square" aria-hidden="true">
      <path d="M4 8V5a1 1 0 0 1 1-1h3M16 4h3a1 1 0 0 1 1 1v3M20 16v3a1 1 0 0 1-1 1h-3M8 20H5a1 1 0 0 1-1-1v-3" />
      <line x1="6" y1="12" x2="18" y2="12" />
    </svg>
  );
}

function CopyIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="square" aria-hidden="true">
      <rect x="9" y="9" width="12" height="12" />
      <path d="M5 15H3V3h12v2" />
    </svg>
  );
}

function CheckIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="square" aria-hidden="true">
      <path d="M4 12l5 5L20 6" />
    </svg>
  );
}

const INSTALL_CMD = "curl -fsSL https://riffpad.ai/install.sh | sh";

export default function PairView({ onPaired }: { onPaired: () => void }) {
  const { t } = useI18n();
  const [code, setCode] = useState("");
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);
  const [scanning, setScanning] = useState(false);
  const [copied, setCopied] = useState(false);
  const [installCopied, setInstallCopied] = useState(false);
  const [installOpen, setInstallOpen] = useState(false);
  const copyTimer = useRef<number | null>(null);

  useEffect(() => {
    return () => {
      if (copyTimer.current) window.clearTimeout(copyTimer.current);
    };
  }, []);

  const submit = useCallback(async (value: string) => {
    const clean = (value || "").toUpperCase().replace(/[^A-Z0-9]/g, "").slice(0, 6);
    if (clean.length !== 6) return;
    setErr("");
    setBusy(true);
    try {
      await pairDevice(clean);
      onPaired();
    } catch (e) {
      setErr(e instanceof PairingCodeUsedError ? t("pair_code_used") : e instanceof Error ? e.message : String(e));
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
      if (copyTimer.current) window.clearTimeout(copyTimer.current);
      copyTimer.current = window.setTimeout(() => setCopied(false), 1500);
    } catch {
      // clipboard unavailable (non-secure context); ignore
    }
  }

  async function copyInstall() {
    try {
      await navigator.clipboard.writeText(INSTALL_CMD);
      setInstallCopied(true);
      if (copyTimer.current) window.clearTimeout(copyTimer.current);
      copyTimer.current = window.setTimeout(() => setInstallCopied(false), 1500);
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

        <p className="muted pair-above">{t("pair_above")}</p>
        <div
          id="code-card"
          className="code-card"
          role="button"
          tabIndex={0}
          aria-label={t("copy_command")}
          onClick={() => void copy()}
          onKeyDown={(e) => {
            if (e.key === "Enter" || e.key === " ") {
              e.preventDefault();
              void copy();
            }
          }}
        >
          <div className="code-card-head">
            <span>{t("bash_label")}</span>
            <button id="copy-btn" className="icon-btn copy-btn" onClick={() => void copy()} aria-label={t("copy_command")}>
              {copied ? <CheckIcon /> : <CopyIcon />}
            </button>
          </div>
          <code>
            <span className="cmd">riffpad</span> pair
          </code>
        </div>
        <p className="muted pair-hint">{t("pair_hint", { cmd: "riffpad pair" })}</p>

        <button id="scan-btn" className="primary scan-btn" onClick={() => setScanning(true)}>
          <ScanIcon />
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
        {busy && (
          <div className="pair-loading">
            <DotMatrix />
            {t("pairing")}
          </div>
        )}
        <button id="install-cli-link" className="install-cli-link" onClick={() => setInstallOpen(true)}>
          {t("install_cli_link")} <span className="chevron">▾</span>
        </button>
        <p className="muted pair-hint">{t("device_stale_hint")}</p>
        {err && <div id="pair-err" className="err">{err}</div>}
      </section>
      {installOpen && (
        <>
          <div className="sheet-backdrop" onClick={() => setInstallOpen(false)} />
          <div className="bottom-sheet">
            <div className="sheet-handle" />
            <h2><span className="glyph">//</span>{t("install_cli_title")}</h2>
            <div
              className="code-card"
              role="button"
              tabIndex={0}
              aria-label={t("copy_command")}
              onClick={() => void copyInstall()}
              onKeyDown={(e) => {
                if (e.key === "Enter" || e.key === " ") {
                  e.preventDefault();
                  void copyInstall();
                }
              }}
            >
              <div className="code-card-head">
                <span>{t("bash_label")}</span>
                <button id="install-copy" className="icon-btn copy-btn" onClick={() => void copyInstall()} aria-label={t("copy_command")}>
                  {installCopied ? <CheckIcon /> : <CopyIcon />}
                </button>
              </div>
              <code><span className="prompt">$</span> {INSTALL_CMD}</code>
            </div>
          </div>
        </>
      )}
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
