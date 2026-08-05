"use client";

import { useLanguage } from "./LanguageProvider";

export function CTA() {
  const { t } = useLanguage();

  return (
    <section className="mx-auto max-w-frame px-4 py-24 text-center sm:px-6 sm:py-32">
      <h2 className="mx-auto max-w-content text-balance text-2xl font-bold leading-[1.25] tracking-[-0.01em] sm:text-3xl">
        {t.cta.title}
      </h2>
      <p className="mx-auto mt-4 max-w-[560px] text-base text-body">
        {t.cta.description}
      </p>

      <div className="console-card mx-auto mt-10 max-w-xl overflow-hidden px-5 py-4 text-left text-sm text-on-console">
        <span className="text-accent">$</span> riffpad request early-access{" "}
        <span className="animate-blink text-accent">▍</span>
      </div>

      <a href="mailto:hi@riffpad.ai" className="btn btn-primary mt-8 h-12 px-8">
        {t.cta.button} <span aria-hidden="true">→</span>
      </a>
      <p className="mt-4 text-xs text-mute">{t.cta.note}</p>
    </section>
  );
}
