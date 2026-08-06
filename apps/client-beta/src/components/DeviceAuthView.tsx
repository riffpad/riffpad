import { useState } from "react";

export default function DeviceAuthView() {
  const code = new URLSearchParams(window.location.search).get("code") || "";
  const [err, setErr] = useState("");

  return (
    <section id="device-auth-view" className="card">
      <h2>授权 CLI 登录</h2>
      <p className="muted">这是终端里 <code>riffpad relay login --github</code> 发起的登录请求。</p>
      <p>授权码：<b>{code || "—"}</b></p>
      <button
        className="primary"
        style={{ width: "100%" }}
        disabled={!code}
        onClick={() => {
          if (!code) return;
          setErr("");
          window.location.href = "/api/auth/github/login?device=" + encodeURIComponent(code);
        }}
      >
        使用 GitHub 登录
      </button>
      {!code && <div className="err">链接缺少授权码，请从终端重新发起登录。</div>}
      {err && <div className="err">{err}</div>}
    </section>
  );
}
