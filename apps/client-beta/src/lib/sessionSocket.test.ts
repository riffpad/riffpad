import { describe, expect, it } from "vitest";
import { dedupeEvent } from "./sessionSocket";
import type { RiffpadEvent } from "./types";

function ev(id: string): RiffpadEvent {
  return { id, sessionId: "s1", timestamp: 1, type: "agent_message", payload: {} };
}

describe("dedupeEvent", () => {
  it("passes new events and skips replays", () => {
    const seen = new Set<string>();
    const a = ev("e1");
    const a2 = ev("e1");
    const b = ev("e2");
    expect(dedupeEvent(seen, a)).toBe(false);
    expect(dedupeEvent(seen, a2)).toBe(true); // same id replayed after reconnect
    expect(dedupeEvent(seen, b)).toBe(false);
    expect(seen.size).toBe(2);
  });

  it("keeps distinct events from the same replay burst", () => {
    const seen = new Set<string>();
    const burst = [ev("a"), ev("b"), ev("c"), ev("a")];
    const skipped = burst.filter((e) => dedupeEvent(seen, e));
    expect(skipped).toHaveLength(1);
  });
});
