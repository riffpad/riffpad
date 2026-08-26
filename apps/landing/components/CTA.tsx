"use client";

import { useLanguage } from "./LanguageProvider";
import { DiscordIcon } from "./icons";
import { WaitlistForm } from "./WaitlistForm";
import { ScrollReveal } from "./ScrollReveal";

export function CTA() {
  const { t } = useLanguage();

  return (
    <section className="mx-auto max-w-frame px-4 py-12 sm:px-6 sm:py-16 lg:py-32">
      <div className="grid items-start gap-10 lg:grid-cols-2 lg:gap-16">
        <ScrollReveal className="min-w-0 text-left">
          <h2 className="max-w-content text-balance text-2xl font-bold leading-[1.25] tracking-[-0.01em] sm:text-3xl">
            {t.cta.title}
          </h2>
          <p className="mt-4 max-w-[560px] text-base text-body">
            {t.cta.description}
          </p>
          <a
            href="https://app.riffpad.ai"
            target="_blank"
            rel="noreferrer"
            className="btn btn-primary mt-8 h-12 px-8 sm:mt-10"
          >
            {t.cta.button} <span aria-hidden="true">→</span>
          </a>
          <p className="mt-4 text-xs text-mute">{t.cta.note}</p>
        </ScrollReveal>

        <ScrollReveal delay={150} className="min-w-0 text-left">
          <p className="text-sm text-body">
            {t.hero.betaPrefix}{" "}
            <span className="font-semibold text-ink">{t.hero.betaWaitlist}</span>
          </p>
          <WaitlistForm className="items-start" />
          <p className="mt-4 text-sm text-body">
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
        </ScrollReveal>
      </div>
    </section>
  );
}
