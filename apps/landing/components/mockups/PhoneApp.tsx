"use client";

import type { Messages } from "@/lib/i18n";
import { ToolRow } from "./MacTerminal";
import type { Approval } from "./types";

export function PhoneApp({
  t,
  approval,
  busy,
  syncing,
  onResolve,
}: {
  t: Messages;
  approval: Approval;
  busy: boolean;
  syncing: boolean;
  onResolve: (verdict: Exclude<Approval, "pending">) => void;
}) {
  return (
    <div className="w-full rounded-[40px] bg-device-frame p-[10px] shadow-device ring-1 ring-black/60">
      <div className="relative flex h-[560px] flex-col overflow-hidden rounded-[30px] bg-surface">
        {/* dynamic island */}
        <div
          className="pointer-events-none absolute left-1/2 top-[8px] z-10 h-[20px] w-[84px] -translate-x-1/2 rounded-full bg-black"
          aria-hidden="true"
        />

        {/* status bar */}
        <div className="flex items-center justify-between px-6 pt-2.5 text-[11px] font-bold text-ink">
          <span>09:41</span>
          <span className="flex items-center gap-1.5" aria-hidden="true">
            <svg width="14" height="10" viewBox="0 0 14 10" fill="currentColor">
              <rect x="0" y="6" width="2.5" height="4" rx="0.5" />
              <rect x="3.8" y="4" width="2.5" height="6" rx="0.5" />
              <rect x="7.6" y="2" width="2.5" height="8" rx="0.5" />
              <rect x="11.4" y="0" width="2.5" height="10" rx="0.5" />
            </svg>
            <span className="text-[10px]">5G</span>
            <svg width="20" height="10" viewBox="0 0 20 10" fill="none">
              <rect
                x="0.5"
                y="0.5"
                width="16"
                height="9"
                rx="2.5"
                stroke="currentColor"
                opacity="0.5"
              />
              <rect x="2" y="2" width="11" height="6" rx="1" fill="currentColor" />
              <path
                d="M18.5 3.5v3a1.5 1.5 0 0 0 0-3z"
                fill="currentColor"
                opacity="0.5"
              />
            </svg>
          </span>
        </div>

        {/* session detail header — mirrors client-beta SessionDetailView */}
        <div className="mt-2 flex items-center justify-between gap-2 border-b border-hairline px-4 pb-2.5">
          <span className="text-mute" aria-hidden="true">
            ←
          </span>
          <span className="truncate text-xs font-bold">
            {t.mockup.phone.session}
          </span>
          <span
            className={`flex flex-none items-center gap-1.5 text-[10px] ${
              syncing ? "text-accent" : "text-success"
            }`}
          >
            <span className="h-1.5 w-1.5 animate-pulse bg-current" aria-hidden="true" />
            {syncing ? t.mockup.sync.syncing : t.mockup.phone.synced}
          </span>
        </div>

        {/* scrollable event feed */}
        <div className="no-scrollbar flex-1 overflow-y-auto px-3 py-2">
          {/* session_start badges */}
          <div className="flex flex-wrap gap-1.5">
            {t.mockup.phone.badges.map((badge) => (
              <span
                key={badge}
                className="border border-hairline px-1.5 py-0.5 text-[10px] text-mute"
              >
                {badge}
              </span>
            ))}
          </div>

          {/* agent message */}
          <p className="mt-2 text-xs leading-[1.6] text-body">
            {t.mockup.phone.agentMsg1}
          </p>

          {/* finished tool calls */}
          <div className="mt-1.5 flex flex-col gap-1">
            {t.mockup.phone.tools.map((tool) => (
              <ToolRow key={tool} text={tool} state="done" />
            ))}
          </div>

          {/* agent message */}
          <p className="mt-2 text-xs leading-[1.6] text-body">
            {t.mockup.phone.agentMsg2}
          </p>

          {/* the tool call that is stuck on approval */}
          <div className="mt-1.5">
            <ToolRow
              text={t.mockup.phone.pendingTool}
              state={approval === "pending" ? "running" : "done"}
            />
          </div>

          {/* approval card — 2px left border like client-beta EventItem */}
          <div
            className={`mt-1.5 border-l-2 py-0.5 pl-3 ${
              approval === "pending"
                ? "border-warning"
                : approval === "approved"
                  ? "border-success"
                  : "border-danger"
            }`}
          >
            <div className="text-[13px] font-bold">{t.mockup.phone.summary}</div>
            <div className="mt-2.5 flex gap-2">
              {approval === "pending" ? (
                <>
                  <button
                    type="button"
                    disabled={busy}
                    onClick={() => onResolve("approved")}
                    className="inline-flex h-8 flex-1 items-center justify-center border border-success/50 text-xs font-bold text-success transition-colors hover:bg-success/10 disabled:opacity-40"
                  >
                    {t.mockup.phone.approve}
                  </button>
                  <button
                    type="button"
                    disabled={busy}
                    onClick={() => onResolve("rejected")}
                    className="inline-flex h-8 flex-1 items-center justify-center border border-danger/50 text-xs font-bold text-danger transition-colors hover:bg-danger/10 disabled:opacity-40"
                  >
                    {t.mockup.phone.reject}
                  </button>
                </>
              ) : (
                <button
                  type="button"
                  disabled
                  className={`inline-flex h-8 flex-1 items-center justify-center border text-xs font-bold opacity-60 ${
                    approval === "approved"
                      ? "border-success/50 text-success"
                      : "border-danger/50 text-danger"
                  }`}
                >
                  {approval === "approved"
                    ? t.mockup.phone.approved
                    : t.mockup.phone.rejected}
                </button>
              )}
            </div>
          </div>

        </div>

        {/* composer */}
        <div className="border-t border-hairline px-3 pb-2 pt-2">
          <div className="relative border border-hairline bg-surface-muted">
            <div className="px-3 py-2 pr-9 text-xs text-mute">{t.mockup.phone.input}</div>
            <span
              className="absolute right-2.5 top-1/2 -translate-y-1/2 text-sm font-bold text-accent"
              aria-hidden="true"
            >
              →
            </span>
          </div>
        </div>

        {/* home indicator */}
        <div className="flex justify-center pb-1.5 pt-1" aria-hidden="true">
          <span className="h-1 w-24 rounded-full bg-ink/25" />
        </div>
      </div>
    </div>
  );
}
