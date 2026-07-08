"use client";

import { useState, useEffect, useRef } from "react";
import { ChevronDown } from "./Icons";
import { useLanguage } from "./LanguageProvider";

interface LanguageSwitchProps {
  variant?: "dropdown" | "inline";
  className?: string;
  buttonClassName?: string;
}

export function LanguageSwitch({
  variant = "dropdown",
  className,
  buttonClassName,
}: LanguageSwitchProps) {
  const { lang, setLang, t } = useLanguage();
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);
  const inline = variant === "inline";

  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false);
      }
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, []);

  const options = (["en", "zh"] as const).map((code) => (
    <button
      key={code}
      onClick={() => {
        setLang(code);
        setOpen(false);
      }}
      className={`flex min-h-[44px] w-full items-center justify-between px-3 py-2 text-left text-sm transition ${
        lang === code
          ? "bg-surface-soft font-semibold text-foreground"
          : "text-body hover:bg-surface-soft hover:text-foreground"
      }`}
      role="option"
      aria-selected={lang === code}
    >
      {t.languages[code]}
      {lang === code && (
        <span className="h-1.5 w-1.5 rounded-full bg-accent" />
      )}
    </button>
  ));

  return (
    <div ref={ref} className={className ? `${className} relative` : "relative"}>
      <button
        onClick={() => setOpen((v) => !v)}
        className={`flex min-h-[44px] items-center gap-1 rounded-md border border-hairline bg-surface px-3 py-2 text-sm font-semibold text-body transition hover:text-foreground ${
          buttonClassName || ""
        }`}
        aria-expanded={open}
        aria-haspopup="listbox"
      >
        {t.languages[lang]}
        <ChevronDown className={`h-4 w-4 transition ${open ? "rotate-180" : ""}`} />
      </button>
      {open && inline && (
        <div
          className="mt-1 overflow-hidden rounded-md border border-hairline bg-surface shadow-sm"
          role="listbox"
        >
          {options}
        </div>
      )}
      {open && !inline && (
        <div
          className="absolute right-0 top-full z-50 mt-1 min-w-[120px] overflow-hidden rounded-md border border-hairline bg-surface shadow-sm"
          role="listbox"
        >
          {options}
        </div>
      )}
    </div>
  );
}
