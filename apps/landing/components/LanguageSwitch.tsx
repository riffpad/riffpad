"use client";

import { GlobeIcon } from "./icons";
import { useLanguage } from "./LanguageProvider";
import type { Language } from "@/lib/i18n";

export function LanguageSwitch({ className = "" }: { className?: string }) {
  const { lang, setLang } = useLanguage();
  const next: Language = lang === "en" ? "zh" : "en";
  const fullWidth = /w-full/.test(className);

  return (
    <button
      type="button"
      onClick={() => setLang(next)}
      className={`relative flex h-9 w-9 cursor-pointer items-center justify-center text-mute transition-colors hover:text-ink active:bg-surface-muted active:text-ink ${className}`}
      aria-label={lang === "en" ? "Switch to 中文" : "Switch to English"}
    >
      <GlobeIcon className="h-4 w-4" />
      <span
        className={`text-[8px] font-bold leading-none ${
          fullWidth
            ? "ml-1"
            : "absolute bottom-1.5 right-1"
        }`}
      >
        {lang === "en" ? "EN" : "中"}
      </span>
    </button>
  );
}
