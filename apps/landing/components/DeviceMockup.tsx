"use client";

import { useEffect, useRef, useState } from "react";
import type { RefObject } from "react";
import { useLanguage } from "./LanguageProvider";
import type { Messages } from "@/lib/i18n";
import { MacTerminal } from "./mockups/MacTerminal";
import { PhoneApp } from "./mockups/PhoneApp";
import { SyncConnector } from "./mockups/SyncConnector";

type TermLine = { id: number; tone: string; text: string };
type Approval = "pending" | "approved" | "rejected";

export function DeviceMockup() {
  const { t, lang } = useLanguage();

  const idRef = useRef(100);
  const nid = () => idRef.current++;

  const [termLines, setTermLines] = useState<TermLine[]>(() =>
    t.mockup.mac.lines.map((l, i) => ({ id: i + 1, tone: l.tone, text: l.text })),
  );
  const [approval, setApproval] = useState<Approval>("pending");
  const [busy, setBusy] = useState(false);
  const [syncing, setSyncing] = useState(false);

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
    setApproval("pending");
    setBusy(false);
    setSyncing(false);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [lang]);

  const termScrollRef = useRef<HTMLDivElement>(null);
  useEffect(() => {
    const el = termScrollRef.current;
    if (el) el.scrollTop = el.scrollHeight;
  }, [termLines, approval]);

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
            approval={approval}
            busy={busy}
            syncing={syncing}
            onResolve={resolveApproval}
          />
        </div>
      </div>
      <p className="mt-8 text-center text-xs text-mute">{t.mockup.hint}</p>
    </div>
  );
}
