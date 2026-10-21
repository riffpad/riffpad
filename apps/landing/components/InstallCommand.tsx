"use client";

import { useState } from "react";
import { useLanguage } from "./LanguageProvider";

// This is a prompt you paste into your own agent, not a shell command.
const PROMPT = "help me set up riffpad: curl -fsSL https://riffpad.ai/SKILL.md";

export function InstallCommand({ className = "" }: { className?: string }) {
  const { t } = useLanguage();
  const [copied, setCopied] = useState(false);

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(PROMPT);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1500);
    } catch {
      // clipboard unavailable (e.g. non-secure context) — leave the text for manual selection
    }
  };

  // Prompt + one copy cell, on the same Mocha surface as the TUI panes. On
  // phones the copy cell owns the top row so the prompt gets the full width
  // on the row below and never clips; from `sm` up it is a single row.
  return (
    <div className={`w-full max-w-[760px] ${className}`}>
      <div className="flex flex-wrap items-stretch border border-tui-hairline bg-tui-bg text-tui-fg">
        <button
          type="button"
          onClick={copy}
          className="order-1 flex w-full flex-none cursor-pointer items-center justify-center whitespace-nowrap border-b border-tui-divider py-2 text-[10px] font-bold uppercase tracking-wide text-tui-mute hover:text-tui-mauve sm:order-2 sm:w-[96px] sm:border-b-0 sm:border-l sm:py-0 sm:text-[11px]"
        >
          {copied ? t.install.copied : t.install.copy}
        </button>
        <code className="order-2 min-w-0 basis-full break-words px-3 pb-3 pt-1 text-[11px] leading-[1.7] sm:order-1 sm:flex-1 sm:basis-0 sm:py-4 sm:text-[13px] sm:leading-normal">
          {PROMPT}
        </code>
      </div>
    </div>
  );
}
