"use client";

import { BookIcon, DiscordIcon, GitHubIcon, MailIcon, XIcon } from "./icons";
import { useLanguage } from "./LanguageProvider";
import { WaitlistForm } from "./WaitlistForm";

export function Footer() {
  const { t } = useLanguage();

  return (
    <footer className="border-t border-hairline bg-surface">
      <div className="mx-auto max-w-frame px-4 py-12 sm:px-6">
        <div className="flex flex-col items-center gap-4 border-b border-hairline pb-12 text-center">
          <p className="text-sm text-body">
            {t.hero.betaPrefix}{" "}
            <span className="font-semibold text-ink">{t.hero.betaWaitlist}</span>
          </p>
          <WaitlistForm />
          <p className="text-sm text-body">
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
        </div>
        <div className="flex flex-col gap-4 pt-8 sm:flex-row sm:items-center sm:justify-between">
          <div className="text-xs text-mute">{t.footer.copyright}</div>
          <nav
            className="flex flex-wrap items-center gap-4 text-body"
            aria-label="Footer"
          >
            <a
              href="https://github.com/riffpad/riffpad"
              target="_blank"
              rel="noreferrer"
              aria-label={t.footer.github}
              className="inline-flex h-8 w-8 items-center justify-center transition-colors hover:text-ink"
            >
              <GitHubIcon className="h-4 w-4" />
            </a>
            <a
              href="https://www.riffpad.ai/docs/guide/quickstart"
              aria-label={t.footer.docs}
              className="inline-flex h-8 w-8 items-center justify-center transition-colors hover:text-ink"
            >
              <BookIcon className="h-4 w-4" />
            </a>
            <a
              href="https://discord.gg/CDNFTg2QyM"
              target="_blank"
              rel="noreferrer"
              aria-label={t.footer.discord}
              className="inline-flex h-8 w-8 items-center justify-center transition-colors hover:text-ink"
            >
              <DiscordIcon className="h-4 w-4" />
            </a>
            <a
              href="https://x.com/riffpad_ai"
              target="_blank"
              rel="noreferrer"
              aria-label={t.footer.x}
              className="inline-flex h-8 w-8 items-center justify-center transition-colors hover:text-ink"
            >
              <XIcon className="h-4 w-4" />
            </a>
            <a
              href="mailto:hi@riffpad.ai"
              aria-label={t.footer.contact}
              className="inline-flex h-8 w-8 items-center justify-center transition-colors hover:text-ink"
            >
              <MailIcon className="h-4 w-4" />
            </a>
          </nav>
        </div>
      </div>
    </footer>
  );
}
