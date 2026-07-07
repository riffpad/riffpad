"use client";

import { useLanguage } from "./LanguageProvider";

export function LanguageSwitch() {
  const { lang, setLang, t } = useLanguage();

  return (
    <div className="flex items-center rounded-full border border-white/10 bg-white/5 p-1 text-xs font-medium">
      {(["en", "zh"] as const).map((code) => (
        <button
          key={code}
          onClick={() => setLang(code)}
          className={`rounded-full px-3 py-1 transition ${
            lang === code
              ? "bg-accent text-black"
              : "text-muted hover:text-foreground"
          }`}
          aria-pressed={lang === code}
        >
          {t.languages[code]}
        </button>
      ))}
    </div>
  );
}
