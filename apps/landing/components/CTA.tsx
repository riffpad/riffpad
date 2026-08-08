"use client";

import { useLanguage } from "./LanguageProvider";

export function CTA() {
  const { t } = useLanguage();

  return (
    <section className="mx-auto max-w-frame px-4 py-12 text-center sm:px-6 sm:py-16 lg:py-32">
      <h2 className="mx-auto max-w-content text-balance text-2xl font-bold leading-[1.25] tracking-[-0.01em] sm:text-3xl">
        {t.cta.title}
      </h2>
      <p className="mx-auto mt-4 max-w-[560px] text-base text-body">
        {t.cta.description}
      </p>

      <a
        href="https://app.riffpad.ai"
        target="_blank"
        rel="noreferrer"
        className="btn btn-primary mt-10 h-12 px-8"
      >
        {t.cta.button} <span aria-hidden="true">→</span>
      </a>
      <p className="mt-6 text-sm text-body">
        {t.cta.betaPrefix}{" "}
        <a
          href="https://formspree.io/f/xjgqddar"
          target="_blank"
          rel="noreferrer"
          className="font-semibold text-ink underline decoration-hairline-strong underline-offset-4 transition-colors hover:decoration-accent"
        >
          {t.cta.betaWaitlist}
        </a>{" "}
        {t.cta.betaOr}{" "}
        <a
          href="https://discord.gg/CDNFTg2QyM"
          target="_blank"
          rel="noreferrer"
          className="font-semibold text-ink underline decoration-hairline-strong underline-offset-4 transition-colors hover:decoration-accent"
        >
          {t.cta.betaDiscord}
        </a>
      </p>
      <p className="mt-4 text-xs text-mute">{t.cta.note}</p>
    </section>
  );
}
