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
      <ol className="steps">
        <li>
          在电脑终端运行 <code>riffpad pair</code>，会打印 6 位配对码和二维码；
        </li>
        <li>把配对码输入到下面（或扫描二维码）；</li>
        <li>配对成功后即可查看和遥控电脑上的 AI coding 会话。</li>
      </ol>
      <div className="row">
        <input placeholder="例如：A1B2C3" value={code} onChange={(e) => setCode(e.target.value)} />
        <button className="primary" onClick={() => void submit()}>配对</button>
      </div>
      {err && <div id="pair-err" className="err">{err}</div>}
    </section>
  );
}
