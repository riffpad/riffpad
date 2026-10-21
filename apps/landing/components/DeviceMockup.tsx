"use client";

import { useEffect, useRef, useState } from "react";
import { useLanguage } from "./LanguageProvider";
import { NvimPane } from "./mockups/NvimPane";
import { PhoneApp } from "./mockups/PhoneApp";
import type { Approval, TermLine } from "./mockups/types";

export function DeviceMockup() {
  const { t, lang } = useLanguage();

  const idRef = useRef(100);
  const seed = () =>
    t.mockup.tui.lines.map((l, i) => ({ id: i + 1, tone: l.tone, text: l.text }));

  const [lines, setLines] = useState<TermLine[]>(seed);
  const [approval, setApproval] = useState<Approval>("pending");

  const timers = useRef<number[]>([]);
  const later = (fn: () => void, ms: number) => {
    timers.current.push(window.setTimeout(fn, ms));
  };
  useEffect(() => () => timers.current.forEach(clearTimeout), []);

  // Re-seed the whole demo when the language changes.
  useEffect(() => {
    timers.current.forEach(clearTimeout);
    timers.current = [];
    setLines(seed());
    setApproval("pending");
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [lang]);

  // Only the phone resolves the approval; the pane just shows the outcome.
  const resolveApproval = (verdict: Exclude<Approval, "pending">) => {
    if (approval !== "pending") return;
    setApproval(verdict);
    later(() => {
      setLines((ls) => [
        ...ls,
        {
          id: idRef.current++,
          tone: verdict === "approved" ? "ok" : "warn",
          text:
            verdict === "approved"
              ? t.mockup.tui.approvedLine
              : t.mockup.tui.rejectedLine,
        },
      ]);
    }, 450);
  };

  return (
    <div className="grid items-start gap-6 lg:grid-cols-[minmax(0,2fr)_minmax(0,1fr)] lg:items-stretch lg:gap-8">
      <NvimPane t={t} lines={lines} approval={approval} />
      <div className="flex justify-center">
        <PhoneApp t={t} approval={approval} onResolve={resolveApproval} />
      </div>
    </div>
  );
}
