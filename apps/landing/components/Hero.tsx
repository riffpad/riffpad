"use client";

import { useLanguage } from "./LanguageProvider";
import { DeviceMockup } from "./DeviceMockup";
import { InstallCommand } from "./InstallCommand";

export function Hero() {
  const { t } = useLanguage();

  return (
    <section
      id="top"
      className="mx-auto max-w-frame scroll-mt-20 px-4 pb-20 pt-14 sm:px-6 sm:pb-28 sm:pt-20"
    >
      <div className="mx-auto max-w-content text-center">
        <h1 className="text-balance text-[28px] font-bold leading-[1.15] tracking-[-0.02em] sm:text-[40px]">
          {t.hero.title1}
          <br />
          {t.hero.title2}
        </h1>
        <p className="mx-auto mt-6 max-w-[560px] text-base leading-[1.7] text-body">
          {t.hero.description}
        </p>
        <div className="mt-8 flex flex-col items-center justify-center gap-3 sm:flex-row">
          <a
            href="https://app.riffpad.ai"
            target="_blank"
            rel="noreferrer"
            className="btn btn-primary w-full sm:w-auto"
          >
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
        <InstallCommand />
      </div>

      <div className="mt-16 sm:mt-20">
        <DeviceMockup />
      </div>
    </section>
  );
}
