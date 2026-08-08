"use client";

import { useLanguage } from "./LanguageProvider";
import { DeviceMockup } from "./DeviceMockup";
import { InstallCommand } from "./InstallCommand";
import { WaitlistForm } from "./WaitlistForm";
import { DiscordIcon } from "./icons";

export function Hero() {
  const { t } = useLanguage();

  return (
    <section
      id="top"
      className="mx-auto max-w-frame scroll-mt-20 px-4 pb-12 pt-10 sm:px-6 sm:pb-28 sm:pt-20"
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
            href="https://www.riffpad.ai/docs/guide/quickstart"
            className="btn btn-secondary w-full sm:w-auto"
          >
            {t.hero.ctaSecondary}
          </a>
        </div>
        <p className="mt-5 text-sm text-body">
          {t.hero.betaPrefix}{" "}
          <span className="font-semibold text-ink">{t.hero.betaWaitlist}</span>
        </p>
        <WaitlistForm />
        <p className="mt-3 text-sm text-body">
          {t.hero.betaOr}{" "}
          <a
            href="https://discord.gg/CDNFTg2QyM"
            target="_blank"
            rel="noreferrer"
            className="group inline-flex items-center gap-1.5 font-semibold text-ink transition-colors hover:text-accent"
          >
            <DiscordIcon className="h-4 w-4 shrink-0" />
            <span className="underline decoration-hairline-strong underline-offset-4 transition-colors group-hover:decoration-accent">
              {t.hero.betaDiscord}
            </span>
          </a>
        </p>
        <InstallCommand />
      </div>

      <div className="mt-16 sm:mt-20">
        <DeviceMockup />
      </div>
    </section>
  );
}
