import { useCallback, useEffect, useState } from "react";
import { api, deviceStore, isRelay } from "../lib/store";
import { deviceDisplayName } from "../lib/device";
import { useI18n } from "../lib/i18n";

interface Device {
  id: string;
  name?: string;
  createdAt?: string;
}

type ConfirmTarget =
  | { kind: "device"; id: string; name: string }
  | { kind: "all" }
  | null;

type TFunc = ReturnType<typeof useI18n>["t"];

function timeAgo(iso: string | undefined, t: TFunc): string {
  if (!iso) return "";
  const ms = Date.now() - new Date(iso).getTime();
  if (!(ms >= 0)) return "";
  const m = Math.floor(ms / 60000);
  if (m < 1) return t("time_just_now");
  if (m < 60) return t("time_min_ago", { n: m });
  const h = Math.floor(m / 60);
  if (h < 24) return t("time_hour_ago", { n: h });
  return t("time_day_ago", { n: Math.floor(h / 24) });
}

function RefreshIcon() {
  return (
    <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M21 12a9 9 0 1 1-2.64-6.36M21 3v6h-6" />
    </svg>
  );
}

interface Props {
  onCurrentRevoked?: () => void;
}

export default function DeviceManager({ onCurrentRevoked }: Props) {
  const { t } = useI18n();
  const [devices, setDevices] = useState<Device[]>([]);
  const [loading, setLoading] = useState(true);
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);
  const [revoking, setRevoking] = useState<string | null>(null);
  const [confirm, setConfirm] = useState<ConfirmTarget>(null);
  const currentId = deviceStore.get()?.deviceId;

  const refresh = useCallback(async () => {
    try {
      const res = await api("/api/devices");
      const data = await res.json();
      setDevices(data.devices || []);
    } catch {
      // transient; keep last list
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  function displayName(d: Device): string {
    return d.name && d.name !== "web" ? d.name : deviceDisplayName();
  }

  async function revoke(id: string) {
    setRevoking(id);
    setErr("");
    try {
      const res = await api("/api/devices/" + id, { method: "DELETE" });
      if (!res.ok) {
        const data = await res.json().catch(() => null);
        throw new Error(data?.error || t("revoke_failed", { status: res.status }));
      }
      await refresh();
      if (id === currentId) onCurrentRevoked?.();
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setRevoking(null);
    }
  }

  async function killAll() {
    setBusy(true);
    setErr("");
    try {
      if (isRelay) {
        for (const d of devices) {
          const res = await api("/api/devices/" + d.id, { method: "DELETE" });
          if (!res.ok) {
            const data = await res.json().catch(() => null);
            throw new Error(data?.error || t("revoke_failed", { status: res.status }));
          }
        }
        await refresh();
        if (devices.some((d) => d.id === currentId)) onCurrentRevoked?.();
        window.alert(t("kill_alert_relay"));
      } else {
        const res = await api("/api/killswitch", { method: "POST" });
        if (!res.ok) {
          const data = await res.json().catch(() => null);
          throw new Error(data?.error || t("revoke_failed", { status: res.status }));
        }
        await refresh();
        if (currentId) onCurrentRevoked?.();
      }
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  function confirmLabel(target: NonNullable<ConfirmTarget>): string {
    if (target.kind === "all") return t("confirm_kill");
    if (target.id === currentId) return t("confirm_revoke_self");
    return t("confirm_revoke", { name: target.name });
  }

  return (
    <>
      <p className="section-label"><span className="glyph">//</span>{t("nav_devices")}</p>
      {loading ? (
        <div id="device-skeleton" className="session-skeleton" aria-hidden="true">
          {[0, 1, 2].map((i) => (
            <div key={i} className="skeleton-card">
              <div className="skeleton-lines">
                <div className="skeleton-line w60" />
                <div className="skeleton-line w35" />
              </div>
              <span className="skeleton-dot" />
            </div>
          ))}
        </div>
      ) : (
        <section className="card device-card">
          <div className="device-head">
            <span className="device-status">■ {t("devices_status", { n: devices.length })}</span>
            <button id="devices-refresh" className="icon-btn" onClick={() => void refresh()} aria-label={t("refresh")} title={t("refresh")}>
              <RefreshIcon />
            </button>
          </div>
          {devices.length === 0 ? (
            <p className="muted empty">{t("no_devices")}</p>
          ) : (
            <ul id="device-list">
              {devices.map((d) => {
                const isCurrent = d.id === currentId;
                const deleting = revoking === d.id;
                return (
                  <li key={d.id} className="device-item">
                    <div className="device-main">
                      <div className="device-row1">
                        <span className="device-name truncate">{displayName(d)}</span>
                        {isCurrent && <span className="device-current">■ {t("this_device")}</span>}
                      </div>
                      <div className="device-row2 truncate">
                        {d.id.slice(0, 8)} · {t("device_active", { time: timeAgo(d.createdAt, t) })}
                      </div>
                    </div>
                    <button
                      className="ghost-danger device-revoke"
                      disabled={deleting || busy}
                      onClick={() => setConfirm({ kind: "device", id: d.id, name: displayName(d) })}
                    >
                      {deleting ? <><span className="spinner" />{t("revoking")}</> : t("revoke")}
                    </button>
                  </li>
                );
              })}
            </ul>
          )}
          <div className="device-danger">
            <button className="ghost-text-danger" disabled={busy || revoking !== null} onClick={() => setConfirm({ kind: "all" })}>
              {busy ? <><span className="spinner" />{t("revoking")}</> : `⚠ ${isRelay ? t("revoke_all") : t("kill_switch")}`}
            </button>
          </div>
          {err && <div className="err">{err}</div>}
        </section>
      )}
      {confirm && (
        <div className="modal-backdrop" onClick={() => setConfirm(null)}>
          <div className="modal" role="dialog" aria-modal="true" onClick={(e) => e.stopPropagation()}>
            <h3><span className="glyph">!</span>{t("danger_title")}</h3>
            <p className="muted">{confirmLabel(confirm)}</p>
            <div className="modal-actions">
              <button className="ghost" onClick={() => setConfirm(null)}>{t("cancel")}</button>
              <button
                className="danger"
                onClick={() => {
                  const target = confirm;
                  setConfirm(null);
                  if (target.kind === "all") void killAll();
                  else void revoke(target.id);
                }}
              >
                {t("confirm")}
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  );
}
