"use client";

import { useState } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { Menu, X, Github } from "./Icons";
import { LanguageSwitch } from "./LanguageSwitch";
import { ThemeToggle } from "./ThemeToggle";
import { useLanguage } from "./LanguageProvider";
import { Hedgehog } from "./Hedgehog";

export function Header() {
  const [open, setOpen] = useState(false);
  const { t } = useLanguage();

  const navLinks = [
    { label: t.nav.docs, href: "/docs" },
    { label: t.nav.blog, href: "/blog" },
    { label: t.nav.pricing, href: "/pricing" },
  ];

  return (
    <header className="sticky top-0 z-50 border-b border-hairline bg-background">
      <div className="mx-auto flex max-w-6xl items-center justify-between px-4 py-3 sm:px-6 lg:px-8">
        <a
          href="/"
          className="flex items-center gap-2 text-lg font-bold text-foreground"
        >
          <Hedgehog className="h-8 w-8" />
          Riffpad
        </a>

        <nav className="hidden flex-1 items-center justify-end gap-6 pr-6 text-sm font-semibold text-body md:flex">
          {navLinks.map((link) => (
            <a
              key={link.href}
              href={link.href}
              className="transition hover:text-foreground"
            >
              {link.label}
            </a>
          ))}
        </nav>

        <div className="hidden items-center gap-3 md:flex">
          <ThemeToggle />
          <LanguageSwitch />
          <a
            href="https://github.com/riffpad/riffpad"
            target="_blank"
            rel="noopener noreferrer"
            className="rounded-md p-2 text-body transition hover:text-foreground"
            aria-label="GitHub"
          >
            <Github className="h-5 w-5" />
          </a>
        </div>

        <button
          className="rounded-md p-2 text-body md:hidden"
          onClick={() => setOpen((v) => !v)}
          aria-label="Toggle menu"
        >
          {open ? <X className="h-6 w-6" /> : <Menu className="h-6 w-6" />}
        </button>
      </div>

      <AnimatePresence>
        {open && (
          <motion.div
            initial={{ opacity: 0, height: 0 }}
            animate={{ opacity: 1, height: "auto" }}
            exit={{ opacity: 0, height: 0 }}
            className="overflow-hidden border-t border-hairline bg-surface md:hidden"
          >
            <div className="flex flex-col gap-4 px-4 py-6 text-base font-medium text-body">
              {navLinks.map((link) => (
                <a
                  key={link.href}
                  href={link.href}
                  onClick={() => setOpen(false)}
                  className="transition hover:text-foreground"
                >
                  {link.label}
                </a>
              ))}
              <div className="flex items-center justify-between gap-3 pt-2">
                <ThemeToggle />
                <LanguageSwitch />
                <a
                  href="https://github.com/riffpad/riffpad"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="rounded-md p-2 text-body transition hover:text-foreground"
                  aria-label="GitHub"
                >
                  <Github className="h-5 w-5" />
                </a>
              </div>
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </header>
  );
}
