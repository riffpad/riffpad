"use client";

import { useLanguage } from "./LanguageProvider";
import { DeviceMockup } from "./DeviceMockup";
import { InstallCommand } from "./InstallCommand";

export function Hero() {
  const { t } = useLanguage();

  return (
    <section
      id="top"
      className="relative flex min-h-[calc(100svh-3.5rem)] scroll-mt-14 items-center px-4 sm:px-6 lg:px-8"
    >
      <div aria-hidden="true" className="hero-glow" />
      <div className="relative z-10 mx-auto grid w-full max-w-frame items-center gap-8 py-10 sm:gap-10 sm:py-12 lg:grid-cols-[minmax(0,0.65fr)_minmax(0,1.35fr)] lg:gap-16">
        <div className="text-left">
          <h1 className="text-balance text-[28px] font-bold leading-[1.1] tracking-[-0.02em] sm:text-[40px] lg:text-[52px]">
            {t.hero.title1}
            <br />
            {t.hero.title2}
          </h1>
          <p className="mt-5 max-w-[540px] text-base leading-[1.7] text-body sm:mt-6 sm:text-lg">
            {t.hero.description}
          </p>
          <div className="mt-6 flex flex-col items-start gap-3 sm:mt-8 sm:flex-row">
            <a
              href="https://app.riffpad.ai"
              target="_blank"
              rel="noreferrer"
              className="btn btn-primary w-full sm:w-auto"
            >
              {t.hero.ctaPrimary}
            </a>
          </div>
          <InstallCommand className="mt-8 sm:mt-10" />
        </div>

        <div className="relative flex min-w-0 items-center justify-center">
          <div className="origin-center scale-[0.68] sm:scale-[0.8] lg:scale-[0.92]">
            <DeviceMockup />
          </div>
        </div>
      </div>
    </section>
  );
}
