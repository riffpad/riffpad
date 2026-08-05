"use client";

import { useLanguage } from "./LanguageProvider";
import { TerminalMockup } from "./TerminalMockup";

export function Hero() {
  const { t } = useLanguage();

  return (
    <section
      id="top"
      className="mx-auto max-w-frame scroll-mt-20 px-4 pb-24 pt-16 sm:px-6 sm:pb-32 sm:pt-24"
    >
      <div className="mx-auto max-w-content text-center">
        <span className="inline-flex items-center rounded border border-hairline px-3 py-1 text-xs text-mute">
          {t.hero.badge}
        </span>
        <h1 className="mt-8 text-balance text-[28px] font-bold leading-[1.5] sm:text-[38px]">
          {t.hero.title1}
          <br />
          {t.hero.title2}
        </h1>
        <p className="mx-auto mt-6 max-w-[640px] text-base leading-[1.7] text-body">
          {t.hero.description}
        </p>
        <div className="mt-10 flex flex-col items-center justify-center gap-3 sm:flex-row">
          <a href="mailto:hi@riffpad.ai" className="btn btn-primary w-full sm:w-auto">
            {t.hero.ctaPrimary}
          </a>
          <a
            href="https://github.com/riffpad/riffpad#readme"
            target="_blank"
            rel="noreferrer"
            className="btn btn-secondary w-full sm:w-auto"
          >
            {t.hero.ctaSecondary}
          </a>
        </div>
        <p className="mt-6 text-xs text-mute">{t.hero.note}</p>
      </div>

      <div className="mt-16 sm:mt-20">
        <TerminalMockup />
      </div>
    </section>
  );
}
