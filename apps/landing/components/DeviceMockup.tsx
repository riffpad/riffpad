"use client";

import { useEffect, useRef, useState } from "react";
import type { RefObject } from "react";
import { useLanguage } from "./LanguageProvider";
import type { Messages } from "@/lib/i18n";

type TermLine = { id: number; tone: string; text: string };
type ChatMsg = { id: number; from: "me" | "agent"; text: string };
type Approval = "pending" | "approved" | "rejected";
type Preset = Messages["mockup"]["phone"]["presets"][number];

export function DeviceMockup() {
  const { t, lang } = useLanguage();

  const idRef = useRef(100);
  const nid = () => idRef.current++;

  const [termLines, setTermLines] = useState<TermLine[]>(() =>
    t.mockup.mac.lines.map((l, i) => ({ id: i + 1, tone: l.tone, text: l.text })),
  );
  const [chat, setChat] = useState<ChatMsg[]>([]);
  const [approval, setApproval] = useState<Approval>("pending");
  const [busy, setBusy] = useState(false);
  const [syncing, setSyncing] = useState(false);
  const [typing, setTyping] = useState(false);

  const timers = useRef<number[]>([]);
  const later = (fn: () => void, ms: number) => {
    timers.current.push(window.setTimeout(fn, ms));
  };
  useEffect(() => () => timers.current.forEach(clearTimeout), []);

  // Re-seed the whole demo when the language changes.
  useEffect(() => {
    timers.current.forEach(clearTimeout);
    timers.current = [];
    setTermLines(
      t.mockup.mac.lines.map((l, i) => ({ id: i + 1, tone: l.tone, text: l.text })),
    );
    setChat([]);
    setApproval("pending");
    setBusy(false);
    setSyncing(false);
    setTyping(false);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [lang]);

  const termScrollRef = useRef<HTMLDivElement>(null);
  useEffect(() => {
    const el = termScrollRef.current;
    if (el) el.scrollTop = el.scrollHeight;
  }, [termLines, typing, approval]);

  const chatScrollRef = useRef<HTMLDivElement>(null);
  useEffect(() => {
    const el = chatScrollRef.current;
    // don't scroll on mount — keep the full feed visible from the top
    if (el && chat.length > 0) el.scrollTop = el.scrollHeight;
  }, [chat]);

  const sendPreset = (preset: Preset) => {
    if (busy) return;
    setBusy(true);
    setSyncing(true);
    setChat((c) => [...c, { id: nid(), from: "me", text: preset.send }]);

    later(() => {
      setTermLines((ls) => [
        ...ls,
        {
          id: nid(),
          tone: "cmd",
          text: `${t.mockup.mac.fromPhone} · ${preset.send}`,
        },
      ]);
      setTyping(true);
    }, 500);

    preset.term.forEach((line, i) => {
      later(() => {
        setTermLines((ls) => [
          ...ls,
          { id: nid(), tone: line.tone, text: line.text },
        ]);
        setTyping(i < preset.term.length - 1);
      }, 1200 + i * 700);
    });

    later(
      () => {
        setTyping(false);
        setChat((c) => [...c, { id: nid(), from: "agent", text: preset.ack }]);
        setSyncing(false);
        setBusy(false);
      },
      1200 + preset.term.length * 700 + 600,
    );
  };

  const resolveApproval = (verdict: Exclude<Approval, "pending">) => {
    if (busy || approval !== "pending") return;
    setBusy(true);
    setApproval(verdict);
    setSyncing(true);
    later(() => {
      setTermLines((ls) => [
        ...ls,
        {
          id: nid(),
          tone: verdict === "approved" ? "ok" : "warn",
          text:
            verdict === "approved"
              ? t.mockup.mac.approvedLine
              : t.mockup.mac.rejectedLine,
        },
      ]);
    }, 450);
    later(() => {
      setSyncing(false);
      setBusy(false);
    }, 1000);
  };

  return (
    <div className="flex flex-col items-center">
      <div className="flex w-full flex-col items-center justify-center gap-8 lg:flex-row lg:gap-6">
        <div className="w-full max-w-[480px]">
          <MacTerminal
            t={t}
            lines={termLines}
            typing={typing}
            approval={approval}
            scrollRef={termScrollRef}
          />
        </div>
        <SyncConnector
          syncing={syncing}
          label={`${t.mockup.sync.label} · ${t.mockup.sync.latency}`}
        />
        <div className="w-full max-w-[300px]">
          <PhoneApp
            t={t}
            chat={chat}
            approval={approval}
            busy={busy}
            syncing={syncing}
            onSend={sendPreset}
            onResolve={resolveApproval}
            scrollRef={chatScrollRef}
          />
        </div>
      </div>
      <p className="mt-8 text-center text-xs text-mute">{t.mockup.hint}</p>
    </div>
  );
}

// The terminal palette follows the page theme (light terminal on the
// light page, dark on dark), like a real terminal matching the OS theme.
function TermLineView({ line }: { line: TermLine }) {
  if (line.tone === "user") {
    // full-width highlighted bar, like the real TUI renders a sent prompt
    return (
      <div className="-mx-4 my-1.5 bg-term-elevate-strong px-4 py-1.5 sm:-mx-5 sm:px-5">
        <span className="mr-2 text-term-mute" aria-hidden="true">
          ❯
        </span>
        {line.text}
      </div>
    );
  }
  if (line.tone === "think") {
    return <div className="my-1 text-term-mute">{line.text}</div>;
  }
  if (line.tone === "tool") {
    return (
      <div className="mt-2">
        <span className="mr-2 text-term-green" aria-hidden="true">
          ●
        </span>
        <span className="font-bold">{line.text}</span>
      </div>
    );
  }
  if (line.tone === "sub") {
    return (
      <div className="ml-6 text-term-mute">
        <span className="mr-2" aria-hidden="true">
          └
        </span>
        {line.text}
      </div>
    );
  }
  if (line.tone === "warn") {
    return (
      <div className="text-term-yellow">
        <span className="mr-2" aria-hidden="true">
          !
        </span>
        {line.text}
      </div>
    );
  }
  if (line.tone === "agent") {
    return <div className="mt-1 text-term-fg/80">{line.text}</div>;
  }
  return (
    <div className={line.tone === "cmd" ? "mt-1 text-term-orange" : undefined}>
      {line.tone === "ok" && (
        <span className="mr-2 text-term-green" aria-hidden="true">
          ✓
        </span>
      )}
      {line.tone === "info" && (
        <span className="mr-2 text-term-blue" aria-hidden="true">
          ▸
        </span>
      )}
      {line.text}
    </div>
  );
}

function TypingDots() {
  return (
    <span className="inline-flex items-center gap-1" aria-hidden="true">
      {[0, 1, 2].map((i) => (
        <span
          key={i}
          className="h-1 w-1 animate-bounce rounded-full bg-current"
          style={{ animationDelay: `${i * 0.15}s` }}
        />
      ))}
    </span>
  );
}

function MacTerminal({
  t,
  lines,
  typing,
  approval,
  scrollRef,
}: {
  t: Messages;
  lines: TermLine[];
  typing: boolean;
  approval: Approval;
  scrollRef: RefObject<HTMLDivElement>;
}) {
  const m = t.mockup.mac;
  return (
    <div className="w-full overflow-hidden rounded-[10px] border border-term-border bg-term-bg text-term-fg shadow-terminal">
      {/* macOS traffic lights, no title */}
      <div className="flex items-center border-b border-term-border bg-term-bar px-4 py-2.5">
        <div className="flex items-center gap-2" aria-hidden="true">
          <span className="h-3 w-3 rounded-full bg-mac-red ring-1 ring-inset ring-black/15" />
          <span className="h-3 w-3 rounded-full bg-mac-yellow ring-1 ring-inset ring-black/15" />
          <span className="h-3 w-3 rounded-full bg-mac-green ring-1 ring-inset ring-black/15" />
        </div>
      </div>

      <div
        ref={scrollRef}
        className="no-scrollbar h-[300px] overflow-y-auto px-4 py-4 text-[13px] leading-[1.8] sm:px-5"
      >
        {lines.map((line) => (
          <TermLineView key={line.id} line={line} />
        ))}
        {typing && (
          <div className="text-term-mute">
            <TypingDots />
          </div>
        )}
        {approval === "pending" && (
          <div className="-mx-4 mt-2 border-t-2 border-term-blue bg-term-elevate px-4 py-3 sm:-mx-5 sm:px-5">
            <div className="font-bold text-term-blue">{m.approvalTitle}</div>
            <div className="mt-2 font-bold">{m.approvalCmd}</div>
            <div className="text-term-mute">{m.approvalDesc}</div>
            <div className="mt-3">{m.approvalQuestion}</div>
            <div className="-mx-4 mt-1 bg-term-elevate-strong px-4 sm:-mx-5 sm:px-5">
              <span className="inline-block w-[2ch] text-term-blue" aria-hidden="true">
                ❯
              </span>
              {m.approvalOpt1}
            </div>
            <div className="text-term-fg/80">
              <span className="inline-block w-[2ch]" aria-hidden="true" />
              {m.approvalOpt2}
            </div>
            <div className="text-term-fg/80">
              <span className="inline-block w-[2ch]" aria-hidden="true" />
              {m.approvalOpt3}
            </div>
            <div className="mt-3 text-[11px] text-term-mute">
              {m.approvalFooter}
            </div>
          </div>
        )}
      </div>

      {/* TUI-style input — same bg as the terminal, no placeholder/hints */}
      <div className="px-4 pb-4 pt-1 sm:px-5">
        <div className="flex items-center gap-2 border border-term-border px-3 py-2 text-[13px]">
          <span className="text-term-green" aria-hidden="true">
            ❯
          </span>
          <span
            className="h-3.5 w-[7px] animate-pulse bg-term-green"
            aria-hidden="true"
          />
        </div>
      </div>
    </div>
  );
}

function SyncConnector({
  syncing,
  label,
}: {
  syncing: boolean;
  label: string;
}) {
  const dash = syncing ? "flow-dash-line flow-fast" : "flow-dash-line";
  return (
    <div className="flex flex-col items-center gap-1.5" aria-hidden="true">
      <svg
        className="h-12 w-6 rotate-90 lg:h-6 lg:w-16 lg:rotate-0"
        viewBox="0 0 64 24"
      >
        <path
          id="sync-ev"
          d="M64 7 L0 7"
          fill="none"
          stroke="var(--accent)"
          strokeOpacity={0.6}
          strokeWidth={2}
          strokeDasharray="6 6"
          className={dash}
        />
        <path
          id="sync-cmd"
          d="M0 17 L64 17"
          fill="none"
          stroke="rgb(var(--info))"
          strokeOpacity={0.6}
          strokeWidth={2}
          strokeDasharray="6 6"
          className={dash}
        />
        <rect
          width={8}
          height={5}
          x={-4}
          y={-2.5}
          fill="var(--accent)"
          className="arch-packet"
        >
          <animateMotion dur="1.6s" repeatCount="indefinite">
            <mpath href="#sync-ev" />
          </animateMotion>
          <animate
            attributeName="opacity"
            values="0;1;1;0"
            keyTimes="0;0.2;0.8;1"
            dur="1.6s"
            repeatCount="indefinite"
          />
        </rect>
        <rect
          width={8}
          height={5}
          x={-4}
          y={-2.5}
          fill="rgb(var(--info))"
          className="arch-packet"
        >
          <animateMotion dur="1.6s" begin="-0.8s" repeatCount="indefinite">
            <mpath href="#sync-cmd" />
          </animateMotion>
          <animate
            attributeName="opacity"
            values="0;1;1;0"
            keyTimes="0;0.2;0.8;1"
            dur="1.6s"
            begin="-0.8s"
            repeatCount="indefinite"
          />
        </rect>
      </svg>
      <span className="hidden text-[10px] text-mute lg:block">{label}</span>
    </div>
  );
}

function ToolRow({
  text,
  state,
}: {
  text: string;
  state: "done" | "running";
}) {
  return (
    <div className="flex items-center gap-2 border border-hairline px-2.5 py-1.5 text-[11px] text-body">
      {state === "done" ? (
        <span className="h-1.5 w-1.5 flex-none bg-success" aria-hidden="true" />
      ) : (
        <span className="flex flex-none items-center gap-[3px]" aria-hidden="true">
          {[0, 1, 2].map((i) => (
            <span
              key={i}
              className="h-1 w-1 animate-bounce bg-warning"
              style={{ animationDelay: `${i * 0.15}s` }}
            />
          ))}
        </span>
      )}
      <span className="flex-1 truncate">{text}</span>
      <span className="text-mute" aria-hidden="true">
        ▸
      </span>
    </div>
  );
}

function PhoneApp({
  t,
  chat,
  approval,
  busy,
  syncing,
  onSend,
  onResolve,
  scrollRef,
}: {
  t: Messages;
  chat: ChatMsg[];
  approval: Approval;
  busy: boolean;
  syncing: boolean;
  onSend: (preset: Preset) => void;
  onResolve: (verdict: Exclude<Approval, "pending">) => void;
  scrollRef: RefObject<HTMLDivElement>;
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
        <div ref={scrollRef} className="no-scrollbar flex-1 overflow-y-auto px-3 py-2">
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

          {/* chat */}
          <div className="mt-3 flex flex-col gap-2">
            {chat.map((msg) => (
              <div
                key={msg.id}
                className={
                  msg.from === "me"
                    ? "ml-8 self-end bg-surface-muted px-3 py-2 text-xs leading-[1.6] text-body"
                    : "mr-8 self-start px-3 py-2 text-xs leading-[1.6] text-body"
                }
              >
                {msg.text}
              </div>
            ))}
          </div>
        </div>

        {/* composer */}
        <div className="border-t border-hairline px-3 pb-2 pt-2">
          <div className="flex flex-wrap gap-1.5">
            {t.mockup.phone.presets.map((preset, i) => (
              <button
                key={i}
                type="button"
                disabled={busy}
                onClick={() => onSend(preset)}
                className="border border-hairline px-2 py-1 text-[11px] text-body transition-colors hover:border-accent hover:text-ink disabled:opacity-40"
              >
                {preset.send}
              </button>
            ))}
          </div>
          <div className="relative mt-2 border border-hairline bg-surface-muted">
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
