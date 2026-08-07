import { useEffect, useRef, useState } from "react";
import { useI18n } from "../lib/i18n";

interface Props {
  onCode(code: string): void;
  onClose(): void;
}

interface Detector {
  detect(video: HTMLVideoElement): Promise<Array<{ rawValue: string }>>;
}

type DetectorCtor = new (opts: { formats: string[] }) => Detector;

export function codeFromQr(text: string): string | null {
  const m = text.match(/[?&]pair=([A-Za-z0-9]{6})/);
  if (m) return m[1].toUpperCase();
  const m2 = text.match(/\b([A-Z0-9]{6})\b/i);
  return m2 ? m2[1].toUpperCase() : null;
}

export default function ScanQR({ onCode, onClose }: Props) {
  const { t } = useI18n();
  const videoRef = useRef<HTMLVideoElement | null>(null);
  const streamRef = useRef<MediaStream | null>(null);
  const doneRef = useRef(false);
  const [state, setState] = useState<"starting" | "scanning" | "error">("starting");
  const [msg, setMsg] = useState("");

  useEffect(() => {
    let cancelled = false;
    let raf = 0;
    let detector: Detector | null = null;

    async function start() {
      try {
        const Ctor = (window as unknown as { BarcodeDetector?: DetectorCtor }).BarcodeDetector;
        if (!Ctor) {
          setState("error");
          setMsg(t("scan_unsupported"));
          return;
        }
        const stream = await navigator.mediaDevices.getUserMedia({
          video: { facingMode: "environment" },
          audio: false,
        });
        if (cancelled) {
          stream.getTracks().forEach((tr) => tr.stop());
          return;
        }
        streamRef.current = stream;
        const video = videoRef.current;
        if (!video) return;
        video.srcObject = stream;
        await video.play();
        detector = new Ctor({ formats: ["qr_code"] });
        setState("scanning");

        const tick = async () => {
          if (cancelled || doneRef.current || !detector || !video || video.readyState < 2) {
            if (!cancelled && !doneRef.current) raf = requestAnimationFrame(tick);
            return;
          }
          try {
            const codes = await detector.detect(video);
            for (const c of codes) {
              const code = codeFromQr(c.rawValue);
              if (code) {
                doneRef.current = true;
                onCode(code);
                return;
              }
            }
          } catch {
            // keep scanning
          }
          raf = requestAnimationFrame(tick);
        };
        raf = requestAnimationFrame(tick);
      } catch {
        if (!cancelled) {
          setState("error");
          setMsg(t("scan_denied"));
        }
      }
    }
    void start();

    return () => {
      cancelled = true;
      cancelAnimationFrame(raf);
      streamRef.current?.getTracks().forEach((tr) => tr.stop());
      streamRef.current = null;
    };
  }, [onCode, t]);

  return (
    <div id="scan-overlay" className="scan-overlay">
      <div className="scan-card">
        <div className="scan-head">
          <h3><span className="glyph">//</span>{t("scan_title")}</h3>
          <button className="ghost" onClick={onClose}>{t("scan_cancel")}</button>
        </div>
        {state === "starting" && <p className="muted scan-note">{t("scan_reading")}</p>}
        {state === "scanning" && <video ref={videoRef} className="scan-video" playsInline muted />}
        {state === "error" && <div className="err">{msg}</div>}
      </div>
    </div>
  );
}
