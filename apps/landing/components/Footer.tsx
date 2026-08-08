"use client";

import { Logo } from "./Logo";
import { BookIcon, DiscordIcon, GitHubIcon, MailIcon } from "./icons";
import { useLanguage } from "./LanguageProvider";

export function Footer() {
  const { t } = useLanguage();

  return (
    <footer className="border-t border-hairline bg-surface">
      <div className="mx-auto max-w-frame px-4 py-12 sm:px-6">
        <div className="flex flex-col gap-6 md:flex-row md:items-center md:justify-between">
          <div>
            <a
              href="#top"
              className="flex cursor-pointer items-center gap-2 text-sm font-bold text-ink"
            >
              <Logo className="h-6 w-6" />
              riffpad
            </a>
            <p className="mt-2 text-sm text-mute">{t.footer.tagline}</p>
          </div>
          <nav
            className="flex flex-wrap gap-x-8 gap-y-2 text-sm text-body"
            aria-label="Footer"
          >
            <a
              href="https://github.com/riffpad/riffpad"
              target="_blank"
              rel="noreferrer"
              className="flex items-center gap-1.5 transition-colors hover:text-ink"
            >
              <GitHubIcon className="h-4 w-4" />
              {t.footer.github}
            </a>
            <a
              href="https://www.riffpad.ai/docs/guide/quickstart"
              className="flex items-center gap-1.5 transition-colors hover:text-ink"
            >
              <BookIcon className="h-4 w-4" />
              {t.footer.docs}
            </a>
            <a
              href="https://discord.gg/CDNFTg2QyM"
              target="_blank"
              rel="noreferrer"
              className="flex items-center gap-1.5 transition-colors hover:text-ink"
            >
              <DiscordIcon className="h-4 w-4" />
              {t.footer.discord}
            </a>
            <a
              href="mailto:hi@riffpad.ai"
              className="flex items-center gap-1.5 transition-colors hover:text-ink"
            >
              <MailIcon className="h-4 w-4" />
              {t.footer.contact}
            </a>
          </nav>
        </div>

        <div className="mt-10 flex flex-col gap-2 border-t border-hairline pt-6 text-xs text-mute sm:flex-row sm:items-center sm:justify-between">
          <span>{t.footer.copyright}</span>
          <span>{t.footer.rights}</span>
        </div>
      </div>
    </footer>
  );
}
