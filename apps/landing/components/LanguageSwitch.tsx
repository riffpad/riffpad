"use client";

import { useLanguage } from "./LanguageProvider";

export function LanguageSwitch() {
  const { lang, setLang, t } = useLanguage();

  return (
    <div className="flex items-center rounded-md border border-white/10 bg-surface p-1 text-xs font-medium">
      {(["en", "zh"] as const).map((code) => (
        <button
          key={code}
          onClick={() => setLang(code)}
          className={`rounded-sm px-3 py-1 transition ${
            lang === code
              ? "bg-foreground text-background"
              : "text-body hover:text-foreground"
          }`}
          aria-pressed={lang === code}
        >
          {t.languages[code]}
        </button>
      ))}
    </div>
  );
}
