import { api, isRelay } from "./store";
import type { SessionInfo } from "./types";

export interface SessionMetaPatch {
  displayName?: string;
  hidden?: boolean;
}

interface LocalEntry {
  displayName?: string;
  hidden?: boolean;
}

const LOCAL_KEY = "riffpad.sessionMeta";

function loadLocal(): Record<string, LocalEntry> {
  try {
    const v = JSON.parse(localStorage.getItem(LOCAL_KEY) || "{}") as Record<string, LocalEntry>;
    return v && typeof v === "object" ? v : {};
  } catch {
    return {};
  }
}

function saveLocal(map: Record<string, LocalEntry>) {
  try {
    localStorage.setItem(LOCAL_KEY, JSON.stringify(map));
  } catch {
    // ignore storage failures (private mode etc.)
  }
}

// In relay mode /api/sessions already returns displayName/hidden
// (server-authoritative, synced across devices). Local (8787) mode has no
// relay account, so the same fields live in localStorage as a fallback.
export function applySessionMeta(sessions: SessionInfo[]): SessionInfo[] {
  if (isRelay) return sessions;
  const map = loadLocal();
  return sessions.map((s) => {
    const m = map[s.id];
    if (!m) return s;
    return { ...s, displayName: m.displayName ?? s.displayName, hidden: m.hidden ?? s.hidden };
  });
}

export async function updateSessionMeta(
  session: { id: string; hostId?: string },
  patch: SessionMetaPatch,
): Promise<void> {
  if (isRelay) {
    if (!session.hostId) throw new Error("missing hostId");
    const res = await api(`/api/sessions/${session.id}/meta`, {
      method: "PUT",
      body: JSON.stringify({ hostId: session.hostId, ...patch }),
    });
    if (!res.ok) {
      const data = await res.json().catch(() => ({}));
      throw new Error(String(data.error || `HTTP ${res.status}`));
    }
    return;
  }
  const map = loadLocal();
  map[session.id] = { ...(map[session.id] || {}), ...patch };
  saveLocal(map);
}
