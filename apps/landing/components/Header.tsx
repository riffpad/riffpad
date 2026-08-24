"use client";

import { Logo } from "./Logo";
import { LanguageSwitch } from "./LanguageSwitch";
import { ThemeToggle } from "./ThemeToggle";
import { MobileMenu } from "./MobileMenu";
import { GitHubIcon, BookIcon } from "./icons";
import { useLanguage } from "./LanguageProvider";
import githubStars from "../lib/github-stars.json";

function formatStars(n: number): string {
  return n >= 1000 ? `${(n / 1000).toFixed(1).replace(/\.0$/, "")}k` : String(n);
}

function GitHubLink({
  compact = false,
  className = "",
}: {
  compact?: boolean;
  className?: string;
}) {
  const stars = githubStars.stars;
  return (
    <a
      href="https://github.com/riffpad/riffpad"
      target="_blank"
      rel="noreferrer"
      aria-label="GitHub"
      className={`flex cursor-pointer items-center justify-center gap-1.5 text-mute transition-colors hover:text-ink active:bg-surface-muted active:text-ink ${
        compact ? "h-9 w-9" : "h-9 px-2"
      } ${className}`}
    >
      <GitHubIcon className="h-4 w-4" />
      {!compact && stars !== null && (
        <span className="text-xs font-bold leading-none">{formatStars(stars)}</span>
      )}
    </a>
  );
}

function DocsLink({
  compact = false,
  className = "",
}: {
  compact?: boolean;
  className?: string;
}) {
  const { t } = useLanguage();

  return (
    <a
      href="https://www.riffpad.ai/docs/guide/quickstart"
      target="_blank"
      rel="noreferrer"
      aria-label={t.header.docs}
      className={`flex cursor-pointer items-center justify-center text-mute transition-colors hover:text-ink active:bg-surface-muted active:text-ink h-9 ${
        compact ? "w-9" : "w-9"
      } ${className}`}
    >
      <BookIcon className="h-4 w-4" />
    </a>
  );
}

function ControlGroup() {
  return (
    <div className="hidden items-center gap-1 md:flex">
      <DocsLink />
      <LanguageSwitch />
      <ThemeToggle />
      <GitHubLink />
    </div>
  );
}

export function Header() {
  return (
    <header className="sticky top-0 z-50 border-b border-hairline bg-surface">
      <div className="mx-auto flex h-14 max-w-frame items-center justify-between px-4 sm:px-6">
        <a
          href="#top"
          className="flex h-11 cursor-pointer items-center gap-2 text-ink"
        >
          <Logo className="h-6 w-6" />
          <span className="text-sm font-bold">riffpad</span>
        </a>

        <div className="flex items-center gap-2">
          <ControlGroup />
          <MobileMenu>
            <DocsLink compact />
            <LanguageSwitch />
            <ThemeToggle />
            <GitHubLink compact />
          </MobileMenu>
        </div>
      </div>
    </header>
  );
}
