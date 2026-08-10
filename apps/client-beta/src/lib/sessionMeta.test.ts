import { beforeEach, describe, expect, it } from "vitest";
import { applySessionMeta, updateSessionMeta } from "./sessionMeta";
import type { SessionInfo } from "./types";

const LOCAL_KEY = "riffpad.sessionMeta";

describe("sessionMeta local fallback", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it("applies local displayName/hidden on top of host sessions", () => {
    const sessions: SessionInfo[] = [
      { id: "s1", name: "demo", cli: "claude", status: "running" },
      { id: "s2", name: "other", cli: "codex", status: "running" },
    ];
    localStorage.setItem(LOCAL_KEY, JSON.stringify({ s1: { displayName: "我的会话", hidden: true } }));

    const got = applySessionMeta(sessions);
    expect(got[0].displayName).toBe("我的会话");
    expect(got[0].hidden).toBe(true);
    expect(got[1]).toEqual(sessions[1]);
  });

  it("updates and merges local meta", async () => {
    await updateSessionMeta({ id: "s1" }, { displayName: "x", hidden: true });
    const got = applySessionMeta([{ id: "s1", name: "demo" }]);
    expect(got[0].displayName).toBe("x");
    expect(got[0].hidden).toBe(true);

    // A later rename must not reset the hidden flag.
    await updateSessionMeta({ id: "s1" }, { displayName: "y" });
    const got2 = applySessionMeta([{ id: "s1", name: "demo" }]);
    expect(got2[0].displayName).toBe("y");
    expect(got2[0].hidden).toBe(true);
  });

  it("clearing displayName with an empty string is preserved", async () => {
    await updateSessionMeta({ id: "s1" }, { displayName: "temp" });
    await updateSessionMeta({ id: "s1" }, { displayName: "" });
    const got = applySessionMeta([{ id: "s1", name: "demo" }]);
    expect(got[0].displayName).toBe("");
  });
});
