import { createContext, useContext, useEffect, useState, type ReactNode } from "react";

export type Lang = "zh" | "en";

const zh: Record<string, string> = {
  connecting: "连接中…",
  not_paired: "未配对：请刷新页面并重新输入配对码",
  handshake_failed: "握手失败：",
  connect_failed: "连接失败：",
  cli_auth: "CLI 授权",
  theme_light: "切换到浅色模式",
  theme_dark: "切换到深色模式",
  sidebar_open: "打开菜单",
  sidebar_close: "关闭菜单",

  github_login: "使用 GitHub 登录",

  pair_title: "配对设备",
  pair_hero_title: "配对本地 daemon",
  pair_hero_subtitle: "实时控制你的 AI coding 会话",
  scan_qr: "扫码",
  or_manual: "或手动输入配对码",
  copy_command: "复制命令",
  copied: "已复制",
  pairing: "配对中…",
  scan_title: "扫描二维码",
  scan_cancel: "取消",
  scan_reading: "正在识别二维码…",
  scan_unsupported: "当前浏览器不支持扫码，请手动输入配对码。",
  scan_denied: "无法访问摄像头，请检查权限后重试。",
  pair_hint: "然后扫码",
  pair_above: "在电脑上运行下面的命令",
  bash_label: "bash",
  pair_step1: "在电脑终端运行 {cmd}，会打印 6 位配对码和二维码；",
  pair_step2: "把配对码输入到下面（或扫描二维码）；",
  pair_step3: "配对成功后即可查看和遥控电脑上的 AI coding 会话。",
  pair_code_ph: "例如：A1B2C3",
  pair_btn: "配对",

  session_name_ph: "名称（可选）",
  session_prompt_ph: "初始指令（可选）",
  session_cwd_ph: "工作目录（默认 daemon 当前目录）",
  sessions_label: "会话",
  start_session: "启动会话",
  refresh: "刷新",
  logout: "退出登录",
  no_sessions: "暂无会话",
  empty_title: "还没有会话？三步开始",
  empty_step1_relay: "在电脑上安装并启动 daemon：{cmd}",
  empty_step1_local: "daemon 已就绪（服务在线）",
  empty_step2: "在电脑上创建会话，例如：riffpad run --cli claude --prompt \"你的任务\"",
  empty_step3: "会话会出现在上方列表，点开即可在手机上查看、审批、下达新指令。",
  start_failed: "启动失败",

  back: "返回",
  stop: "停止",
  stopping: "停止中…",
  send: "发送",
  prompt_ph: "下达指令…",
  confirm_stop: "确定停止这个会话？agent 进程会被终止。",
  send_failed: "未连接：设备可能已失效或会话已结束，请刷新页面重试",
  waiting_events: "等待事件…",
  session_default: "会话",
  connected_encrypted: "已连接（加密）",
  reconnecting: "重连中…",
  disconnected: "已断开",
  device_revoked: "连接失败：设备可能已被撤销，请刷新页面并重新配对",
  reconnect_in: "连接断开，{s}s 后自动重连…",

  event_session_start: "会话开始",
  event_session_end: "会话结束",
  event_agent_status: "状态",
  event_agent_message: "Agent",
  event_user_message: "你",
  event_tool_call: "工具调用",
  event_file_change: "文件变更",
  event_command: "命令",
  event_approval_request: "审批",
  event_approval_response: "审批回复",
  event_prompt: "指令",
  event_control: "控制",
  event_notify: "通知",

  approve: "同意",
  reject: "拒绝",
  approved: "已同意",
  rejected: "已拒绝",
  sending: "发送中…",
  approval_send_failed: "审批发送失败：连接已断开，请刷新页面重试",
  approval_send_retry: "审批发送失败，请重试",
  content_preview: "内容预览：",
  truncated: "…[截断]",
  session_end_reason: "原因：",

  devices: "设备",
  revoke_all: "撤销全部设备",
  kill_switch: "熔断",
  no_devices: "暂无已配对设备",
  revoke: "撤销",
  device: "设备",
  confirm_kill: "熔断将停止所有会话并撤销所有设备，确定继续？",
  kill_alert_relay: "已撤销全部云端设备。要停止电脑上的 agent，请在电脑端执行 riffpad kill。",

  cli_auth_title: "授权 CLI 登录",
  cli_auth_desc: "这是终端里 {cmd} 发起的登录请求。",
  auth_code: "授权码：",
  cli_auth_missing: "链接缺少授权码，请从终端重新发起登录。",

  lang_toggle: "EN",
};

const en: Record<string, string> = {
  connecting: "Connecting…",
  not_paired: "Not paired: refresh the page and enter a pairing code",
  handshake_failed: "Handshake failed: ",
  connect_failed: "Connection failed: ",
  cli_auth: "CLI auth",
  theme_light: "Switch to light mode",
  theme_dark: "Switch to dark mode",
  sidebar_open: "Open menu",
  sidebar_close: "Close menu",

  github_login: "Continue with GitHub",

  pair_title: "Pair device",
  pair_hero_title: "Pair Local Daemon",
  pair_hero_subtitle: "Control your AI coding sessions in real-time.",
  scan_qr: "Scan",
  or_manual: "OR ENTER CODE MANUALLY",
  copy_command: "Copy command",
  copied: "Copied",
  pairing: "Pairing…",
  scan_title: "Scan QR code",
  scan_cancel: "Cancel",
  scan_reading: "Looking for a QR code…",
  scan_unsupported: "Camera scanning is not supported in this browser; enter the code manually.",
  scan_denied: "Camera access denied. Check permissions and try again.",
  pair_hint: "Then Scan",
  pair_above: "Run command below in your computer",
  bash_label: "bash",
  pair_step1: "Run {cmd} in the terminal on your computer — it prints a 6-char code and QR;",
  pair_step2: "Enter the code below (or scan the QR);",
  pair_step3: "Once paired you can view and remote-control the AI coding sessions.",
  pair_code_ph: "e.g. A1B2C3",
  pair_btn: "Pair",

  session_name_ph: "Name (optional)",
  session_prompt_ph: "Initial prompt (optional)",
  session_cwd_ph: "Working directory (default: daemon cwd)",
  sessions_label: "Sessions",
  start_session: "Start session",
  refresh: "Refresh",
  logout: "Sign out",
  no_sessions: "No sessions yet",
  empty_title: "No sessions? Three steps",
  empty_step1_relay: "Install and start the daemon on your computer: {cmd}",
  empty_step1_local: "Daemon is ready (online)",
  empty_step2: "Start a session on the computer, e.g.: riffpad run --cli claude --prompt \"your task\"",
  empty_step3: "Sessions appear in the list above; open one to view, approve, and send instructions.",
  start_failed: "Failed to start",

  back: "Back",
  stop: "Stop",
  stopping: "Stopping…",
  send: "Send",
  prompt_ph: "Send a message…",
  confirm_stop: "Stop this session? The agent process will be terminated.",
  send_failed: "Not connected: the device may be invalid or the session ended. Refresh and retry.",
  waiting_events: "Waiting for events…",
  session_default: "Session",
  connected_encrypted: "Connected (encrypted)",
  reconnecting: "Reconnecting…",
  disconnected: "Disconnected",
  device_revoked: "Connection failed: device may have been revoked. Refresh and re-pair.",
  reconnect_in: "Disconnected, reconnecting in {s}s…",

  event_session_start: "Session start",
  event_session_end: "Session end",
  event_agent_status: "Status",
  event_agent_message: "Agent",
  event_user_message: "You",
  event_tool_call: "Tool call",
  event_file_change: "File change",
  event_command: "Command",
  event_approval_request: "Approval",
  event_approval_response: "Approval reply",
  event_prompt: "Prompt",
  event_control: "Control",
  event_notify: "Notify",

  approve: "Approve",
  reject: "Reject",
  approved: "Approved",
  rejected: "Rejected",
  sending: "Sending…",
  approval_send_failed: "Failed to send approval: connection lost. Refresh and retry.",
  approval_send_retry: "Failed to send approval, please retry.",
  content_preview: "Content preview:",
  truncated: "…[truncated]",
  session_end_reason: "Reason: ",

  devices: "Devices",
  revoke_all: "Revoke all devices",
  kill_switch: "Kill switch",
  no_devices: "No paired devices",
  revoke: "Revoke",
  device: "device",
  confirm_kill: "Kill switch stops all sessions and revokes all devices. Continue?",
  kill_alert_relay: "All cloud devices revoked. To stop agents on the computer, run `riffpad kill` there.",

  cli_auth_title: "Authorize CLI login",
  cli_auth_desc: "This request came from {cmd} in your terminal.",
  auth_code: "Code:",
  cli_auth_missing: "Missing code in the link. Start login again from the terminal.",

  lang_toggle: "中",
};

export type Vars = Record<string, string | number>;

export function format(template: string, vars?: Vars): string {
  if (!vars) return template;
  let out = template;
  for (const [k, v] of Object.entries(vars)) {
    out = out.replaceAll(`{${k}}`, String(v));
  }
  return out;
}

export function detectLang(): Lang {
  const q = new URLSearchParams(location.search).get("lang");
  if (q === "zh" || q === "en") return q;
  const saved = localStorage.getItem("riffpad.lang");
  if (saved === "zh" || saved === "en") return saved;
  return navigator.language.toLowerCase().startsWith("zh") ? "zh" : "en";
}

// getT returns a translate function for the currently detected language. It
// is used outside React components (e.g. the session socket) where the hook
// is unavailable.
export function getT(): (key: string, vars?: Vars) => string {
  const lang = detectLang();
  const table = lang === "zh" ? zh : en;
  return (key, vars) => format(table[key] ?? en[key] ?? key, vars);
}

interface I18nValue {
  lang: Lang;
  setLang: (l: Lang) => void;
  t: (key: string, vars?: Vars) => string;
}

const I18nContext = createContext<I18nValue>({
  lang: "zh",
  setLang: () => undefined,
  t: (key) => key,
});

export function I18nProvider({ children }: { children: ReactNode }) {
  const [lang, setLangState] = useState<Lang>(detectLang);

  useEffect(() => {
    document.documentElement.lang = lang;
    const onStorage = () => setLangState(detectLang());
    window.addEventListener("storage", onStorage);
    return () => window.removeEventListener("storage", onStorage);
  }, [lang]);

  const setLang = (l: Lang) => {
    localStorage.setItem("riffpad.lang", l);
    setLangState(l);
  };
  const t = (key: string, vars?: Vars) => {
    const table = lang === "zh" ? zh : en;
    return format(table[key] ?? en[key] ?? key, vars);
  };

  return (
    <I18nContext.Provider value={{ lang, setLang, t }}>
      {children}
    </I18nContext.Provider>
  );
}

export function useI18n() {
  return useContext(I18nContext);
}
