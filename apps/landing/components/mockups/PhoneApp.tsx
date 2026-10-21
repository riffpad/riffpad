"use client";

import type { Messages } from "@/lib/i18n";
import { ToolRow } from "./NvimPane";
import type { Approval } from "./types";

export function PhoneApp({
  t,
  approval,
  onResolve,
}: {
  t: Messages;
  approval: Approval;
  onResolve: (verdict: Exclude<Approval, "pending">) => void;
}) {
  const p = t.mockup.phone;

  return (
    <div className="w-full rounded-[40px] bg-device-frame p-[10px] shadow-device ring-1 ring-black/60">
      <div className="flex h-[520px] flex-col overflow-hidden rounded-[30px] bg-tui-bg text-tui-fg">
        {/* session bar */}
        <div className="flex items-center justify-between gap-2 border-b border-tui-surface px-4 py-3">
          <span className="truncate text-xs font-bold">{p.session}</span>
          <span className="flex flex-none items-center gap-1.5 text-[10px] text-tui-green">
            <span className="h-1.5 w-1.5 bg-current" aria-hidden="true" />
            {p.synced}
          </span>
        </div>

        {/* event feed */}
        <div className="no-scrollbar flex-1 overflow-y-auto px-3 py-2.5">
          <div className="flex flex-wrap gap-1.5">
            {p.badges.map((badge) => (
              <span
                key={badge}
                className="border border-tui-surface px-1.5 py-0.5 text-[10px] text-tui-mute"
              >
                {badge}
              </span>
            ))}
          </div>

          <div className="mt-1.5 flex flex-col gap-1">
            {p.tools.map((tool) => (
              <ToolRow key={tool} text={tool} state="done" />
            ))}
          </div>

          <p className="mt-2 text-[11px] leading-[1.6] text-tui-fg/70">
            {p.agentMsg}
          </p>

          <div className="mt-1.5">
            <ToolRow
              text={p.pendingTool}
              state={approval === "pending" ? "running" : "done"}
            />
          </div>

          {/* approval card — 2px left border, status colour as the only accent */}
          <div
            className={`mt-1.5 border-l-2 py-0.5 pl-3 ${
              approval === "pending"
                ? "border-tui-mauve"
                : approval === "approved"
                  ? "border-tui-green"
                  : "border-tui-red"
            }`}
          >
            <div className="text-[13px] font-bold">{p.summary}</div>
            <div className="mt-2 flex gap-2">
              {approval === "pending" ? (
                <>
                  <button
                    type="button"
                    onClick={() => onResolve("approved")}
                    className="inline-flex h-11 flex-1 items-center justify-center border border-tui-green/50 text-xs font-bold text-tui-green transition-colors hover:bg-tui-green/10"
                  >
                    {p.approve}
                  </button>
                  <button
                    type="button"
                    onClick={() => onResolve("rejected")}
                    className="inline-flex h-11 flex-1 items-center justify-center border border-tui-red/50 text-xs font-bold text-tui-red transition-colors hover:bg-tui-red/10"
                  >
                    {p.reject}
                  </button>
                </>
              ) : (
                <button
                  type="button"
                  disabled
                  className={`inline-flex h-11 flex-1 items-center justify-center border text-xs font-bold opacity-60 ${
                    approval === "approved"
                      ? "border-tui-green/50 text-tui-green"
                      : "border-tui-red/50 text-tui-red"
                  }`}
                >
                  {approval === "approved" ? p.approved : p.rejected}
                </button>
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
