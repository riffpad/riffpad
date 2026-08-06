import { useEffect, useState } from "react";
import { pairDevice } from "../lib/device";
import { useI18n } from "../lib/i18n";

export default function PairView({ onPaired }: { onPaired: () => void }) {
  const { t } = useI18n();
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
      <h2><span className="glyph">//</span>{t("pair_title")}</h2>
      <ol className="steps">
        <li>{t("pair_step1", { cmd: "riffpad pair" })}</li>
        <li>{t("pair_step2")}</li>
        <li>{t("pair_step3")}</li>
      </ol>
      <div className="row">
        <input
          className="pair-input"
          placeholder={t("pair_code_ph")}
          value={code}
          onChange={(e) => setCode(e.target.value)}
          autoCapitalize="characters"
          spellCheck={false}
        />
        <button className="primary" onClick={() => void submit()}>{t("pair_btn")}</button>
      </div>
      {err && <div id="pair-err" className="err">{err}</div>}
    </section>
  );
}
