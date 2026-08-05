"use client";

import { useLanguage } from "./LanguageProvider";
import { TerminalMockup } from "./TerminalMockup";

export function Hero() {
  const { t } = useLanguage();

  return (
    <section
      id="top"
      className="bg-terminal-grid mx-auto max-w-frame scroll-mt-20 px-4 pb-20 pt-14 sm:px-6 sm:pb-28 sm:pt-20"
    >
      <div className="grid items-center gap-12 lg:grid-cols-2">
        <div>
          <span className="label">{`// ${t.hero.badge}`}</span>
          <h1 className="mt-6 text-balance text-[28px] font-bold leading-[1.15] tracking-[-0.02em] sm:text-[40px]">
            {t.hero.title1}
            <br />
            {t.hero.title2}
            <span className="animate-blink text-accent">▍</span>
          </h1>
          <p className="mt-6 max-w-[520px] text-base leading-[1.7] text-body">
            {t.hero.description}
          </p>
          <div className="mt-8 flex flex-col gap-3 sm:flex-row">
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
          <p className="mt-5 text-xs text-mute">{t.hero.note}</p>
        </div>

        <div className="lg:w-full">
          <TerminalMockup />
        </div>
      </div>
    </section>
  );
}
