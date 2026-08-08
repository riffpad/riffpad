export interface RelaySession {
  token: string;
  username: string;
}

export interface Device {
  deviceId: string | null;
  serverPub: string | null;
  jwk: JsonWebKey;
}

export interface SessionInfo {
  id: string;
  name?: string;
  cli?: string;
  cwd?: string;
  status?: string;
  lastSeenAt?: string;
}

export interface EventPayload {
  [key: string]: unknown;
}

export interface RiffpadEvent {
  id: string;
  sessionId: string;
  timestamp: number;
  type: string;
  // Per-session increasing sequence number assigned by the daemon (#173).
  // Absent/0 on events from older daemons or one-off messages.
  seq?: number;
  payload?: EventPayload;
}

export interface ApprovalPayload extends EventPayload {
  requestId: string;
  action?: string;
  summary?: string;
  options?: string[];
  args?: Record<string, unknown>;
}

export const EVENT_LABELS: Record<string, string> = {
  session_start: "会话开始",
  session_end: "会话结束",
  agent_status: "状态",
  agent_message: "Agent",
  user_message: "你",
  tool_call: "工具调用",
  file_change: "文件变更",
  command: "命令",
  approval_request: "审批",
  approval_response: "审批回复",
  approval_resolved: "审批结果",
  prompt: "指令",
  control: "控制",
  notify: "通知",
};
