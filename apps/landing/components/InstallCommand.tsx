"use client";

import { useState } from "react";
import { useLanguage } from "./LanguageProvider";

const COMMAND = "curl -fsSL https://riffpad.ai/SKILL.md";

export function InstallCommand({ className = "" }: { className?: string }) {
  const { t } = useLanguage();
  const [copied, setCopied] = useState(false);

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(COMMAND);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1500);
    } catch {
      // clipboard unavailable (e.g. non-secure context) — leave the text for manual selection
    }
  };

  // One full-width command line, on the same Mocha surface as the TUI panes.
  return (
    <div className={`w-full max-w-[760px] ${className}`}>
      <div className="flex items-stretch border border-tui-hairline bg-tui-bg text-tui-fg">
        <span
          className="flex flex-none items-center px-3 text-tui-mauve"
          aria-hidden="true"
        >
          ❯
        </span>
        <code className="flex min-w-0 flex-1 items-center overflow-hidden whitespace-nowrap bg-tui-bg py-4 text-[10px] sm:text-[13px]">
          {COMMAND}
        </code>
        <button
          type="button"
          onClick={copy}
          className="flex-none cursor-pointer border-l border-tui-divider px-4 text-[10px] font-bold uppercase tracking-wide text-tui-mute hover:text-tui-mauve sm:text-[11px]"
        >
          {copied ? t.install.copied : t.install.copy}
        </button>
      </div>
    </div>
  );
}
