"use client";

import { useLanguage } from "./LanguageProvider";
import type { Language } from "@/lib/i18n";

export function LanguageSwitch() {
  const { lang, setLang } = useLanguage();
  const next: Language = lang === "en" ? "zh" : "en";

  return (
    <button
      type="button"
      onClick={() => setLang(next)}
      className="flex h-11 min-w-11 cursor-pointer items-center justify-center px-3 text-sm font-medium text-body transition-colors hover:text-ink"
      aria-label={lang === "en" ? "Switch to 中文" : "Switch to English"}
    >
      [{lang === "en" ? "中" : "EN"}]
    </button>
  );
}
