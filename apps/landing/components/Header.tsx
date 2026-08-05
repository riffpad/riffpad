"use client";

import { useState } from "react";
import { Logo } from "./Logo";
import { useLanguage } from "./LanguageProvider";
import { LanguageSwitch } from "./LanguageSwitch";
import { ThemeToggle } from "./ThemeToggle";

const NAV_LINK_IDS = ["features", "how", "security", "faq"] as const;

export function Header() {
  const { t } = useLanguage();
  const [open, setOpen] = useState(false);

  const navLinks = NAV_LINK_IDS.map((id) => ({
    id,
    label: t.nav[id],
  }));

  return (
    <header className="sticky top-0 z-50 border-b border-hairline bg-canvas">
      <div className="mx-auto flex h-14 max-w-frame items-center justify-between px-4 sm:px-6">
        <a
          href="#top"
          className="flex h-11 cursor-pointer items-center gap-2 text-ink"
        >
          <Logo className="h-6 w-6" />
          <span className="text-base font-bold">riffpad</span>
        </a>

        <nav
          className="hidden items-center gap-6 md:flex"
          aria-label="Main navigation"
        >
          {navLinks.map((link) => (
            <a
              key={link.id}
              href={`#${link.id}`}
              className="text-sm text-body transition-colors hover:text-ink"
            >
              {link.label}
            </a>
          ))}
        </nav>

        <div className="flex items-center gap-1">
          <LanguageSwitch />
          <ThemeToggle />
          <a
            href="mailto:hi@riffpad.ai"
            className="btn btn-primary ml-2 hidden h-10 px-4 text-sm sm:inline-flex"
          >
            {t.nav.cta}
          </a>
          <button
            type="button"
            onClick={() => setOpen((v) => !v)}
            className="flex h-11 cursor-pointer items-center gap-1 px-2 text-sm text-body transition-colors hover:text-ink md:hidden"
            aria-expanded={open}
            aria-controls="mobile-menu"
          >
            [{open ? "-" : "+"} {t.nav.menu}]
          </button>
        </div>
      </div>

      {open && (
        <div
          id="mobile-menu"
          className="border-t border-hairline bg-canvas md:hidden"
        >
          <nav
            className="mx-auto flex max-w-frame flex-col px-4 py-4 sm:px-6"
            aria-label="Mobile navigation"
          >
            {navLinks.map((link) => (
              <a
                key={link.id}
                href={`#${link.id}`}
                onClick={() => setOpen(false)}
                className="flex min-h-11 cursor-pointer items-center border-b border-hairline text-base text-body transition-colors hover:text-ink"
              >
                {link.label}
              </a>
            ))}
            <a
              href="mailto:hi@riffpad.ai"
              className="btn btn-primary mt-4"
            >
              {t.nav.cta}
            </a>
          </nav>
        </div>
      )}
    </header>
  );
}
