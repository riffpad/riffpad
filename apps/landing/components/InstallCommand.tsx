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

  return (
    <div className={`w-full max-w-[560px] ${className}`}>
      <p className="mb-3 text-sm font-semibold text-ink">
        {t.install.tagline}
      </p>
      <div className="border border-hairline bg-console text-on-console shadow-card transition-colors duration-200 hover:border-accent">
        <div className="flex items-stretch border-b border-hairline">
          <span className="flex h-10 items-center px-4 text-xs font-bold uppercase tracking-wide text-on-console-mute">
            {t.install.label}
          </span>
          <button
            type="button"
            onClick={copy}
            className="ml-auto h-10 cursor-pointer px-4 text-xs font-bold text-on-console-mute transition-colors hover:text-accent"
          >
            {copied ? t.install.copied : t.install.copy}
          </button>
        </div>
        <button
          type="button"
          onClick={copy}
          className="flex w-full cursor-pointer items-center justify-center px-4 py-4"
        >
          <code className="break-all text-[13px] sm:text-sm">{COMMAND}</code>
        </button>
      </div>
    </div>
  );
}
