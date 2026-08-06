import { useCallback, useEffect, useState } from "react";
import { api, isRelay } from "../lib/store";
import { useI18n } from "../lib/i18n";

interface Device {
  id: string;
  name?: string;
  createdAt?: string;
}

export default function DeviceManager() {
  const { t } = useI18n();
  const [devices, setDevices] = useState<Device[]>([]);
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);

  const refresh = useCallback(async () => {
    try {
      const res = await api("/api/devices");
      const data = await res.json();
      setDevices(data.devices || []);
    } catch {
      // transient; keep last list
    }
  }, []);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  async function revoke(id: string) {
    setErr("");
    try {
      await api("/api/devices/" + id, { method: "DELETE" });
      await refresh();
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    }
  }

  async function kill() {
    if (!window.confirm(t("confirm_kill"))) return;
    setBusy(true);
    setErr("");
    try {
      if (isRelay) {
        for (const d of devices) {
          await api("/api/devices/" + d.id, { method: "DELETE" });
        }
        await refresh();
        window.alert(t("kill_alert_relay"));
      } else {
        await api("/api/killswitch", { method: "POST" });
        await refresh();
      }
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <section className="card device-card">
      <div className="row device-head">
        <h3><span className="glyph">//</span>{t("devices")}</h3>
        <button className="ghost" onClick={() => void refresh()}>{t("refresh")}</button>
        <button className="danger" onClick={() => void kill()} disabled={busy}>
          {isRelay ? t("revoke_all") : t("kill_switch")}
        </button>
      </div>
      {devices.length === 0 ? (
        <p className="muted empty">{t("no_devices")}</p>
      ) : (
        <ul id="device-list">
          {devices.map((d) => (
            <li key={d.id}>
              <span className="truncate">{(d.name || t("device")) + " · " + d.id.slice(0, 8)}</span>
              <button className="danger ghost-danger" onClick={() => void revoke(d.id)}>{t("revoke")}</button>
            </li>
          ))}
        </ul>
      )}
      {err && <div className="err">{err}</div>}
    </section>
  );
}
