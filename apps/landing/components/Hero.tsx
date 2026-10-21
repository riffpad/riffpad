"use client";

import { useLanguage } from "./LanguageProvider";
import { InstallCommand } from "./InstallCommand";

export function Hero() {
  const { t } = useLanguage();

  return (
    <section
      id="top"
      className="mx-auto w-full max-w-frame scroll-mt-14 px-4 pb-4 sm:px-6 sm:pb-6"
    >
      {/* The copy owns the first screen: nothing else may peek above the fold. */}
      <div className="flex min-h-[calc(100svh-3.5rem)] flex-col justify-center py-10">
        <span className="label">{`// ${t.hero.label}`}</span>
        <h1 className="mt-6 text-[clamp(36px,5.4vw,84px)] font-bold leading-[1.04] tracking-[-0.03em]">
          {t.hero.title1}
          <br />
          {t.hero.title2}
        </h1>
        <p className="mt-6 text-base leading-[1.7] text-body sm:text-lg">
          {t.hero.description}
        </p>
        <div className="mt-8 flex flex-col items-stretch gap-5 sm:flex-row sm:items-center sm:gap-7">
          <a
            href="https://app.riffpad.ai"
            target="_blank"
            rel="noreferrer"
            className="btn btn-primary h-16 w-full px-12 text-lg sm:w-auto"
          >
            {t.hero.ctaPrimary}
          </a>
          <a
            href="https://www.riffpad.ai/docs/guide/quickstart"
            className="text-center text-sm font-bold text-body underline decoration-hairline-strong decoration-1 underline-offset-4 hover:text-ink sm:text-left"
          >
            {t.hero.ctaSecondary}
          </a>
        </div>
        <div className="mt-8 flex flex-wrap items-end justify-between gap-4 sm:mt-10">
          <InstallCommand className="max-w-[760px]" />
          <span className="hidden pb-1 text-xs text-mute xl:block">
            {`// v0.2.6 · self-hosted · e2e`}
          </span>
        </div>
      </div>
    </section>
  );
}
