"use client";

import { useLanguage } from "./LanguageProvider";

export function Hero() {
  const { t } = useLanguage();

  return (
    <section
      id="top"
      className="mx-auto w-full max-w-frame scroll-mt-14 px-4 pb-10 sm:px-6 sm:pb-12"
    >
      {/* The copy owns the first screen: nothing else may peek above the fold. */}
      <div className="flex min-h-[calc(100svh-3.5rem)] flex-col justify-center py-10">
        <span className="label">{`// ${t.hero.label}`}</span>
        <h1 className="mt-5 max-w-[720px] text-balance text-[28px] font-bold leading-[1.1] tracking-[-0.02em] sm:text-[40px] lg:text-[52px]">
          {t.hero.title1}
          <br />
          {t.hero.title2}
        </h1>
        <p className="mt-4 max-w-[560px] text-base leading-[1.7] text-body sm:text-lg">
          {t.hero.description}
        </p>
        <div className="mt-8 flex flex-col items-stretch gap-4 sm:flex-row sm:items-center sm:gap-6">
          <a
            href="https://app.riffpad.ai"
            target="_blank"
            rel="noreferrer"
            className="btn btn-primary h-14 w-full px-10 text-lg sm:w-auto"
          >
            {t.hero.ctaPrimary}
          </a>
          <a
            href="https://www.riffpad.ai/docs/guide/quickstart"
            className="text-center text-sm font-bold text-body underline decoration-hairline-strong decoration-1 underline-offset-4 transition-colors hover:text-ink sm:text-left"
          >
            {t.hero.ctaSecondary}
          </a>
        </div>
      </div>
    </section>
  );
}
