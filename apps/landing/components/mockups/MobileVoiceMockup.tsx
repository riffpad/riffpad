"use client";

import { useState, useEffect } from "react";
import { motion } from "framer-motion";
import { DeviceFrameset } from "react-device-frameset";
import "react-device-frameset/styles/marvel-devices.min.css";
import { useLanguage } from "../LanguageProvider";
import { RefreshCw, Copy, MoreHorizontal } from "../Icons";

export function MobileVoiceMockup() {
  const { t } = useLanguage();
  const s = t.saasWorkspace;
  const m = t.mobileWorkspace;
  const [typedText, setTypedText] = useState("");
  const [waveHeights, setWaveHeights] = useState<number[]>(() =>
    Array(32).fill(3)
  );

  useEffect(() => {
    let rafId: number;
    let lastTime = 0;

    const tick = (time: number) => {
      if (time - lastTime < 16) {
        rafId = requestAnimationFrame(tick);
        return;
      }
      lastTime = time;

      const now = time / 1000;
      const envelope = speechEnvelope(now);

      setWaveHeights(
        Array.from({ length: 32 }, (_, i) => {
          const spatial = i * 0.35;
          const slow = noise(spatial + now * 1.4) * 0.5 + 0.5;
          const fast = Math.abs(noise(spatial * 1.4 + now * 7.5));
          const burst = Math.abs(noise(spatial * 2.1 - now * 9.0));
          const value =
            0.08 + envelope * (0.22 * slow + 0.5 * fast + 0.28 * burst);
          return Math.min(38, 3 + value * 36);
        })
      );

      rafId = requestAnimationFrame(tick);
    };

    rafId = requestAnimationFrame(tick);
    return () => cancelAnimationFrame(rafId);
  }, []);

  useEffect(() => {
    const text = m.voiceText;
    let index = 0;
    let direction = 1;
    let timer: ReturnType<typeof setTimeout>;

    const tick = () => {
      if (direction === 1) {
        if (index < text.length) {
          index += 1;
          setTypedText(text.slice(0, index));
          timer = setTimeout(tick, 80);
        } else {
          timer = setTimeout(() => {
            direction = -1;
            tick();
          }, 1500);
        }
      } else {
        if (index > 0) {
          index -= 1;
          setTypedText(text.slice(0, index));
          timer = setTimeout(tick, 50);
        } else {
          timer = setTimeout(() => {
            direction = 1;
            tick();
          }, 800);
        }
      }
    };

    tick();
    return () => clearTimeout(timer);
  }, [m.voiceText]);

  return (
    <motion.div
      initial={{ opacity: 0, scale: 0.96 }}
      whileInView={{ opacity: 1, scale: 1 }}
      viewport={{ once: true }}
      transition={{ duration: 0.6 }}
      className="flex justify-center"
    >
      <DeviceFrameset device="iPhone X" color="black" zoom={0.85}>
        <div className="flex h-full flex-col overflow-hidden bg-surface">
          {/* Workspace header */}
          <div className="flex items-center justify-between border-b border-hairline bg-surface px-4 py-3">
            <button
              type="button"
              className="flex h-8 w-8 items-center justify-center rounded-md text-muted transition hover:bg-surface-soft hover:text-foreground"
            >
              <MenuIcon className="h-4 w-4" />
            </button>
            <div className="flex flex-col items-center">
              <span className="text-sm font-semibold text-foreground">
                {m.workspaceName}
              </span>
              <span className="text-xs text-muted">{m.workspaceLabel}</span>
            </div>
            <div className="h-8 w-8" />
          </div>

          {/* Chat content */}
          <div className="hide-scrollbar flex-1 space-y-6 overflow-auto p-4">
            {/* Round 1 — from SaaS workspace mockup */}
            <div className="space-y-3">
              <div className="flex justify-end">
                <div className="max-w-[90%] rounded-2xl rounded-br-sm border border-hairline bg-surface-doc px-4 py-2.5 text-sm text-body">
                  {s.userMessage}
                </div>
              </div>

              <div className="flex flex-col items-start gap-2.5">
                <StatusLine label={s.stepThinking} />
                <StatusLine label={s.stepSearch} />
                <StatusLine label={s.stepWrite} />
              </div>

              <div className="text-sm leading-relaxed text-body">
                <span className="text-muted">{s.assistantReply}</span>{" "}
                <span className="font-mono font-semibold text-foreground">
                  {s.filesContent.analysis.name}
                </span>
                <span className="text-muted">{s.assistantReplySuffix}</span>
                <MessageActions />
              </div>
            </div>

            {/* Round 2 — mobile voice follow-up */}
            <div className="space-y-3">
              <div className="flex justify-end">
                <div className="max-w-[90%] rounded-2xl rounded-br-sm border border-hairline bg-surface-doc px-4 py-2.5 text-sm text-body">
                  {m.userMessage}
                </div>
              </div>

              <div className="flex flex-col items-start gap-2.5">
                <StatusLine label={m.stepRead} />
                <StatusLine label={m.stepSearch} />
                <StatusLine label={m.stepWrite} />
              </div>

              <div className="text-sm leading-relaxed text-body">
                <span className="text-muted">{m.assistantReply}</span>{" "}
                <span className="font-mono font-semibold text-foreground">
                  {m.assistantReplyName}
                </span>
                <span className="text-muted">{m.assistantReplySuffix}</span>
                <MessageActions />
              </div>
            </div>
          </div>

          {/* Voice input area */}
          <div className="flex flex-col items-center gap-3 px-4 pb-6 pt-2">
            <div className="min-h-[2.25rem] text-center">
              <span className="text-base font-medium text-foreground">
                {typedText}
              </span>
              <motion.span
                animate={{ opacity: [1, 0, 1] }}
                transition={{ repeat: Infinity, duration: 1 }}
                className="ml-0.5 inline-block h-5 w-0.5 bg-accent"
              />
            </div>

            <span className="text-xs font-medium uppercase tracking-wider text-muted">
              {m.listening}
            </span>

            <div className="flex h-10 w-full items-end justify-between gap-0.5 px-2">
              {waveHeights.map((h, i) => (
                <motion.div
                  key={i}
                  className="flex-1 rounded-[1px] bg-accent"
                  animate={{ height: h }}
                  transition={{ duration: 0.05, ease: "linear" }}
                />
              ))}
            </div>
          </div>
        </div>
      </DeviceFrameset>
    </motion.div>
  );
}

function StatusLine({ label }: { label: string }) {
  return (
    <div className="flex items-center gap-2 text-sm text-muted">
      <span className="h-1.5 w-1.5 rounded-full bg-accent" />
      {label}
    </div>
  );
}

function MenuIcon({ className }: { className?: string }) {
  return (
    <svg
      className={className}
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      viewBox="0 0 24 24"
    >
      <line x1="4" x2="20" y1="6" y2="6" />
      <line x1="4" x2="20" y1="12" y2="12" />
      <line x1="4" x2="20" y1="18" y2="18" />
    </svg>
  );
}

function noise(x: number) {
  return (
    Math.sin(x) * 0.5 +
    Math.sin(x * 2.1 + 1.3) * 0.25 +
    Math.sin(x * 4.7 + 0.7) * 0.125 +
    Math.sin(x * 8.3 + 2.1) * 0.06
  );
}

function speechEnvelope(t: number) {
  const phrase = (Math.sin(t * 0.45) + 1) / 2;
  const word = Math.max(0, Math.sin(t * 2.3 + Math.sin(t * 0.9) * 1.5));
  const syllable = Math.max(0, Math.sin(t * 7.0) * Math.sin(t * 3.2));
  return Math.min(1, phrase * 0.35 + word * 0.55 + syllable * 0.25);
}

function MessageActions() {
  return (
    <div className="mt-2 flex items-center gap-0.5">
      <button
        type="button"
        className="flex h-6 w-6 items-center justify-center rounded-md text-muted transition hover:bg-surface-soft hover:text-foreground"
        aria-label="Regenerate"
      >
        <RefreshCw className="h-3 w-3" />
      </button>
      <button
        type="button"
        className="flex h-6 w-6 items-center justify-center rounded-md text-muted transition hover:bg-surface-soft hover:text-foreground"
        aria-label="Copy"
      >
        <Copy className="h-3 w-3" />
      </button>
      <button
        type="button"
        className="flex h-6 w-6 items-center justify-center rounded-md text-muted transition hover:bg-surface-soft hover:text-foreground"
        aria-label="More"
      >
        <MoreHorizontal className="h-3 w-3" />
      </button>
    </div>
  );
}
