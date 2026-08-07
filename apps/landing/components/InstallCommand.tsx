"use client";

import { useState } from "react";
import { useLanguage } from "./LanguageProvider";

const COMMANDS = {
  unix: "curl -fsSL https://riffpad.ai/install.sh | sh",
} as const;

type Platform = keyof typeof COMMANDS;

export function InstallCommand() {
  const { t } = useLanguage();
  const [platform, setPlatform] = useState<Platform>("unix");
  const [copied, setCopied] = useState(false);

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(COMMANDS[platform]);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1500);
    } catch {
      // clipboard unavailable (e.g. non-secure context) — leave the text for manual selection
    }
  };

  const tabs: { id: Platform; label: string }[] = [
    { id: "unix", label: t.install.unix },
  ];

  return (
    <div className="mx-auto mt-10 w-full max-w-[560px]">
      <div className="border border-hairline bg-console text-on-console shadow-card transition-colors duration-200 hover:border-accent">
        <div className="flex items-stretch border-b border-hairline">
          {tabs.map((tab) => (
            <button
              key={tab.id}
              type="button"
              onClick={() => setPlatform(tab.id)}
              aria-pressed={platform === tab.id}
              className={`h-10 flex-1 cursor-pointer border-b-2 text-xs font-bold transition-colors sm:flex-none sm:px-5 ${
                platform === tab.id
                  ? "border-accent text-on-console"
                  : "border-transparent text-on-console-mute hover:text-accent"
              }`}
            >
              {tab.label}
            </button>
          ))}
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
          <code className="break-all text-[13px] sm:text-sm">
            {COMMANDS[platform]}
          </code>
        </button>
      </div>
    </div>
  );
}
