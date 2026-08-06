import { useCallback, useEffect, useState } from "react";
import { api, isRelay } from "../lib/store";

interface Device {
  id: string;
  name?: string;
  createdAt?: string;
}

export default function DeviceManager() {
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
    if (!window.confirm("熔断将停止所有会话并撤销所有设备，确定继续？")) return;
    setBusy(true);
    setErr("");
    try {
      if (isRelay) {
        // Cloud side has no daemon control; revoke all devices via relay.
        for (const d of devices) {
          await api("/api/devices/" + d.id, { method: "DELETE" });
        }
        await refresh();
        window.alert("已撤销全部云端设备。要停止电脑上的 agent，请在电脑端执行 riffpad kill。");
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
    <section className="card">
      <div className="row">
        <h3>设备</h3>
        <button className="ghost" onClick={() => void refresh()}>刷新</button>
        <button className="danger" onClick={() => void kill()} disabled={busy}>
          {isRelay ? "撤销全部设备" : "熔断"}
        </button>
      </div>
      {devices.length === 0 ? (
        <p className="muted">暂无已配对设备</p>
      ) : (
        <ul id="device-list">
          {devices.map((d) => (
            <li key={d.id}>
              <span className="truncate">{(d.name || "设备") + " · " + d.id.slice(0, 8)}</span>
              <button className="danger" onClick={() => void revoke(d.id)}>撤销</button>
            </li>
          ))}
        </ul>
      )}
      {err && <div className="err">{err}</div>}
    </section>
  );
}
