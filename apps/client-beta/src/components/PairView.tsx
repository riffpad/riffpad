import { useEffect, useState } from "react";
import { pairDevice } from "../lib/device";

export default function PairView({ onPaired }: { onPaired: () => void }) {
  const [code, setCode] = useState("");
  const [err, setErr] = useState("");

  useEffect(() => {
    const params = new URLSearchParams(location.search);
    const p = params.get("pair");
    if (p) setCode(p);
  }, []);

  async function submit() {
    setErr("");
    try {
      await pairDevice(code.trim());
      onPaired();
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    }
  }

  return (
    <section id="pair-view" className="card">
      <h2>配对设备</h2>
      <p className="muted">在电脑上运行 <code>riffpad pair</code> 获取配对码，然后输入到这里。</p>
      <div className="row">
        <input placeholder="配对码" value={code} onChange={(e) => setCode(e.target.value)} />
        <button className="primary" onClick={() => void submit()}>配对</button>
      </div>
      {err && <div id="pair-err" className="err">{err}</div>}
    </section>
  );
}
