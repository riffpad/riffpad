import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import EventItem from "./EventItem";
import type { RiffpadEvent } from "../lib/types";

(globalThis as unknown as { IS_REACT_ACT_ENVIRONMENT: boolean }).IS_REACT_ACT_ENVIRONMENT = true;

function approvalEvent(): RiffpadEvent {
  return {
    id: "e1",
    sessionId: "s1",
    timestamp: 1,
    type: "approval_request",
    payload: { requestId: "r1", action: "Bash", summary: "rm -rf build", options: ["approve", "reject"] },
  };
}

// Multi-device approval consistency (#171): when the daemon broadcasts
// approval_resolved — live or via history replay — every other tab must grey
// out the same card instead of leaving clickable buttons behind.
describe("EventItem approval card with approval_resolved", () => {
  let container: HTMLDivElement;
  let root: Root;
  const send = vi.fn(async () => ({ status: "sent" as const, id: "x" }));

  beforeEach(async () => {
    container = document.createElement("div");
    document.body.appendChild(container);
    root = createRoot(container);
    send.mockClear();
  });

  afterEach(async () => {
    await act(async () => root.unmount());
    container.remove();
  });

  function buttons() {
    return [...container.querySelectorAll<HTMLButtonElement>(".approval-actions button")];
  }

  it("starts with clickable buttons", async () => {
    await act(async () => root.render(<EventItem ev={approvalEvent()} send={send} />));
    expect(buttons().every((b) => !b.disabled)).toBe(true);
  });

  it("greys out the card when approval_resolved arrives for the same requestId", async () => {
    await act(async () => root.render(<EventItem ev={approvalEvent()} send={send} />));
    expect(buttons().every((b) => !b.disabled)).toBe(true);
    // Another viewer approved: this tab's card locks and shows the decision.
    await act(async () =>
      root.render(<EventItem ev={approvalEvent()} send={send} resolved={{ r1: "approve" }} />),
    );
    expect(buttons().every((b) => b.disabled)).toBe(true);
    expect(container.querySelector("button.approve")!.textContent).toBe("approved");
    expect(container.querySelector(".approval-card")!.className).toContain("done");
  });

  it("keeps buttons locked on replay of a reject resolution", async () => {
    // Simulates a reconnect replay that already contains approval_resolved:
    // the card renders settled from the start and the buttons never revive.
    await act(async () =>
      root.render(<EventItem ev={approvalEvent()} send={send} resolved={{ r1: "reject" }} />),
    );
    expect(buttons().every((b) => b.disabled)).toBe(true);
    expect(container.querySelector("button.reject")!.textContent).toBe("rejected");
  });

  it("lets a daemon resolution override a stale local queued state", async () => {
    await act(async () =>
      root.render(<EventItem ev={approvalEvent()} send={send} resolved={{ r1: "approve" }} />),
    );
    // A late local tap must not resurrect the buttons nor show 已过期.
    expect(buttons().every((b) => b.disabled)).toBe(true);
    expect(container.querySelector(".approval-note.expired")).toBeNull();
    expect(send).not.toHaveBeenCalled();
  });
});
