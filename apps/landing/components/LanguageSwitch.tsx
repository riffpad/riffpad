"use client";

import { useLanguage } from "./LanguageProvider";

export function LanguageSwitch() {
  const { lang, setLang, t } = useLanguage();

  return (
    <div className="flex items-center rounded-md border border-hairline bg-surface p-1 text-xs font-semibold">
      {(["en", "zh"] as const).map((code) => (
        <button
          key={code}
          onClick={() => setLang(code)}
          className={`rounded-sm px-3 py-1 transition ${
            lang === code
              ? "bg-foreground text-white"
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
