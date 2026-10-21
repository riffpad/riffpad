"use client";

import { useEffect, useState } from "react";
import { useLanguage } from "./LanguageProvider";
import { NvimPane } from "./mockups/NvimPane";
import { PhoneApp } from "./mockups/PhoneApp";
import type { Approval, TermLine } from "./mockups/types";

export function DeviceMockup() {
  const { t, lang } = useLanguage();

  const seed = () =>
    t.mockup.tui.lines.map((l, i) => ({ id: i + 1, tone: l.tone, text: l.text }));
  // The script ends on the pending line; that is the row the answer replaces.
  const pendingId = t.mockup.tui.lines.length;

  const [lines, setLines] = useState<TermLine[]>(seed);
  const [approval, setApproval] = useState<Approval>("pending");

  // Re-seed the whole demo when the language changes.
  useEffect(() => {
    setLines(seed());
    setApproval("pending");
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [lang]);

  // Only the phone resolves the approval; the pane swaps its pending line for
  // the outcome, in place — no "Waiting…" row left dangling above the result.
  const resolveApproval = (verdict: Exclude<Approval, "pending">) => {
    if (approval !== "pending") return;
    setApproval(verdict);
    setLines((ls) =>
      ls.map((line) =>
        line.id === pendingId
          ? {
              ...line,
              tone: verdict === "approved" ? "ok" : "warn",
              text:
                verdict === "approved"
                  ? t.mockup.tui.approvedLine
                  : t.mockup.tui.rejectedLine,
            }
          : line,
      ),
    );
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
