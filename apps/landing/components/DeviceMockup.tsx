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
  const [chat, setChat] = useState<ChatMsg[]>(() => [
    { id: 1, from: "agent", text: t.mockup.phone.hello },
  ]);
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
    setChat([{ id: 1, from: "agent", text: t.mockup.phone.hello }]);
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
  }, [termLines, typing]);

  const chatScrollRef = useRef<HTMLDivElement>(null);
  useEffect(() => {
    const el = chatScrollRef.current;
    if (el) el.scrollTop = el.scrollHeight;
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
            title={t.mockup.mac.title}
            prompt={t.mockup.mac.prompt}
            status={t.mockup.mac.status}
            lines={termLines}
            typing={typing}
            syncing={syncing}
            syncLabel={syncing ? t.mockup.sync.syncing : t.mockup.phone.synced}
            scrollRef={termScrollRef}
          />
        </div>
        <SyncConnector
          label={t.mockup.sync.label}
          status={syncing ? t.mockup.sync.syncing : t.mockup.sync.status}
          latency={t.mockup.sync.latency}
          syncing={syncing}
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

function TermLineView({ line }: { line: TermLine }) {
  if (line.tone === "warn") {
    return (
      <div className="text-warning">
        <span className="mr-2" aria-hidden="true">
          !
        </span>
        {line.text}
      </div>
    );
  }
  return (
    <div className={line.tone === "cmd" ? "text-accent" : undefined}>
      {line.tone === "ok" && (
        <span className="mr-2 text-success" aria-hidden="true">
          ✓
        </span>
      )}
      {line.tone === "info" && (
        <span className="mr-2 text-info" aria-hidden="true">
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
  title,
  prompt,
  status,
  lines,
  typing,
  syncing,
  syncLabel,
  scrollRef,
}: {
  title: string;
  prompt: string;
  status: string;
  lines: TermLine[];
  typing: boolean;
  syncing: boolean;
  syncLabel: string;
  scrollRef: RefObject<HTMLDivElement>;
}) {
  return (
    <div className="w-full overflow-hidden rounded-[10px] border border-hairline bg-console text-on-console shadow-[0_18px_50px_-12px_rgba(0,0,0,0.28)]">
      {/* macOS title bar */}
      <div className="relative flex items-center bg-console-elevated px-4 py-2.5">
        <div className="flex items-center gap-2" aria-hidden="true">
          <span className="h-3 w-3 rounded-full bg-[#ff5f57] ring-1 ring-inset ring-black/15" />
          <span className="h-3 w-3 rounded-full bg-[#febc2e] ring-1 ring-inset ring-black/15" />
          <span className="h-3 w-3 rounded-full bg-[#28c840] ring-1 ring-inset ring-black/15" />
        </div>
        <span className="absolute left-1/2 max-w-[55%] -translate-x-1/2 truncate text-xs text-on-console-mute">
          {title}
        </span>
        <span
          className={`ml-auto flex items-center gap-1.5 text-[11px] ${
            syncing ? "text-accent" : "text-success"
          }`}
        >
          <span aria-hidden="true">●</span>
          {syncLabel}
        </span>
      </div>

      <div
        ref={scrollRef}
        className="no-scrollbar h-[240px] overflow-y-auto px-4 py-4 text-[13px] leading-[1.9] sm:px-5"
      >
        <div className="text-on-console-mute">{prompt}</div>
        {lines.map((line) => (
          <TermLineView key={line.id} line={line} />
        ))}
        {typing && (
          <div className="text-on-console-mute">
            <TypingDots />
          </div>
        )}
      </div>

      <div className="border-t border-hairline px-4 py-2 text-[11px] text-on-console-mute sm:px-5">
        {status}
      </div>
    </div>
  );
}

function SyncConnector({
  label,
  status,
  latency,
  syncing,
}: {
  label: string;
  status: string;
  latency: string;
  syncing: boolean;
}) {
  return (
    <div
      className="flex flex-col items-center gap-3 lg:flex-row lg:gap-4"
      aria-hidden="true"
    >
      <div className="relative h-16 w-px bg-hairline lg:h-px lg:w-20">
        <span className="absolute left-1/2 top-1/2 h-2 w-2 -translate-x-1/2 -translate-y-1/2 animate-pulse rounded-full bg-accent" />
        {syncing && (
          <span className="absolute left-1/2 top-1/2 h-2 w-2 -translate-x-1/2 -translate-y-1/2 animate-ping rounded-full bg-accent" />
        )}
      </div>
      <div className="text-center text-[11px] leading-5 text-mute">
        <div>{label}</div>
        <div className={syncing ? "text-accent" : "text-success"}>{status}</div>
        <div>{latency}</div>
      </div>
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
    <div className="w-full rounded-[40px] bg-[#1b1b19] p-[10px] shadow-[0_24px_60px_-20px_rgba(0,0,0,0.45)] ring-1 ring-black/60">
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

        {/* app header */}
        <div className="mt-2 flex items-center justify-between border-b border-hairline px-4 pb-2.5">
          <span className="text-sm font-bold">{t.mockup.phone.title}</span>
          <span
            className={`flex items-center gap-1.5 text-[11px] ${
              syncing ? "text-accent" : "text-success"
            }`}
          >
            <span aria-hidden="true">●</span>
            {syncing ? t.mockup.sync.syncing : t.mockup.phone.synced}
          </span>
        </div>

        {/* scrollable body */}
        <div ref={scrollRef} className="no-scrollbar flex-1 overflow-y-auto px-3 py-3">
          <div className="border border-hairline p-3">
            <div className="flex items-center justify-between text-[11px] text-mute">
              <span>{t.mockup.phone.session}</span>
              <span className="text-success">{t.mockup.phone.running}</span>
            </div>
            <div className="mt-1 text-sm font-bold">
              {t.mockup.phone.cli} · {t.mockup.phone.tools}
            </div>
          </div>

          {/* approval card */}
          {approval === "pending" ? (
            <div className="mt-3 border border-warning/20 bg-warning/10 p-3">
              <div className="text-[11px] text-warning">
                {t.mockup.phone.approval}
              </div>
              <div className="mt-1 text-sm font-bold">
                {t.mockup.phone.summary}
              </div>
              <div className="mt-3 flex gap-2">
                <button
                  type="button"
                  disabled={busy}
                  onClick={() => onResolve("approved")}
                  className="inline-flex h-9 flex-1 items-center justify-center border border-success/50 text-xs font-bold text-success transition-colors hover:bg-success/10 disabled:opacity-40"
                >
                  {t.mockup.phone.approve}
                </button>
                <button
                  type="button"
                  disabled={busy}
                  onClick={() => onResolve("rejected")}
                  className="inline-flex h-9 flex-1 items-center justify-center border border-danger/50 text-xs font-bold text-danger transition-colors hover:bg-danger/10 disabled:opacity-40"
                >
                  {t.mockup.phone.reject}
                </button>
              </div>
            </div>
          ) : (
            <div
              className={`mt-3 border p-3 text-xs font-bold ${
                approval === "approved"
                  ? "border-success/40 bg-success/10 text-success"
                  : "border-danger/40 bg-danger/10 text-danger"
              }`}
            >
              {approval === "approved" ? "✓ " : "✕ "}
              {approval === "approved"
                ? t.mockup.phone.approved
                : t.mockup.phone.rejected}
            </div>
          )}

          {/* chat */}
          <div className="mt-3 flex flex-col gap-2">
            {chat.map((msg) => (
              <div
                key={msg.id}
                className={
                  msg.from === "me"
                    ? "ml-8 self-end rounded-[10px] rounded-br-[2px] bg-accent px-3 py-2 text-xs leading-[1.6] text-accent-ink"
                    : "mr-8 self-start rounded-[10px] rounded-bl-[2px] bg-surface-muted px-3 py-2 text-xs leading-[1.6] text-body"
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
          <div className="mt-2 border border-hairline px-3 py-2 text-xs text-mute">
            {t.mockup.phone.input}
          </div>
        </div>

        {/* tab bar + home indicator */}
        <div className="flex border-t border-hairline px-2 pt-1.5 text-[11px] text-mute">
          {t.mockup.phone.tabs.map((tab, index) => (
            <span
              key={index}
              className={
                index === 1
                  ? "flex-1 text-center font-bold text-ink"
                  : "flex-1 text-center"
              }
            >
              {tab}
            </span>
          ))}
        </div>
        <div className="flex justify-center pb-1.5 pt-1" aria-hidden="true">
          <span className="h-1 w-24 rounded-full bg-ink/25" />
        </div>
      </div>
    </div>
  );
}
