"use client";

import { Logo } from "./Logo";
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
              className="transition-colors hover:text-ink"
            >
              {t.footer.github}
            </a>
            <a
              href="/docs"
              className="transition-colors hover:text-ink"
            >
              {t.footer.docs}
            </a>
            <a
              href="mailto:hi@riffpad.ai"
              className="transition-colors hover:text-ink"
            >
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
