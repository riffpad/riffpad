"use client";

import { useState, useRef, useEffect } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { Menu, X, Github } from "./Icons";
import { LanguageSwitch } from "./LanguageSwitch";
import { ThemeToggle } from "./ThemeToggle";
import { useLanguage } from "./LanguageProvider";
import { Logo } from "./Logo";

export function Header() {
  const [open, setOpen] = useState(false);
  const headerRef = useRef<HTMLElement>(null);
  const { t } = useLanguage();

  const navLinks = [
    { label: t.nav.docs, href: "/docs" },
    { label: t.nav.blog, href: "/blog" },
    { label: t.nav.pricing, href: "/pricing" },
  ];

  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (headerRef.current && !headerRef.current.contains(e.target as Node)) {
        setOpen(false);
      }
    };
    const keyHandler = (e: KeyboardEvent) => {
      if (e.key === "Escape") setOpen(false);
    };
    document.addEventListener("mousedown", handler);
    document.addEventListener("keydown", keyHandler);
    return () => {
      document.removeEventListener("mousedown", handler);
      document.removeEventListener("keydown", keyHandler);
    };
  }, []);

  useEffect(() => {
    if (open) {
      document.body.style.overflow = "hidden";
    } else {
      document.body.style.overflow = "";
    }
    return () => {
      document.body.style.overflow = "";
    };
  }, [open]);

  return (
    <header
      ref={headerRef}
      className="relative sticky top-0 z-50 border-b border-hairline bg-background"
    >
      <div className="mx-auto flex max-w-6xl items-center justify-between px-4 py-3 sm:px-6 lg:px-8">
        <a
          href="/"
          className="flex items-center gap-2 text-lg font-bold text-foreground"
        >
          <Logo className="h-8 w-8" />
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
          className="inline-flex min-h-[44px] min-w-[44px] items-center justify-center rounded-md p-2 text-body transition hover:text-foreground md:hidden"
          onClick={() => setOpen((v) => !v)}
          aria-label="Toggle menu"
          aria-expanded={open}
        >
          {open ? <X className="h-6 w-6" /> : <Menu className="h-6 w-6" />}
        </button>
      </div>

      <AnimatePresence>
        {open && (
          <motion.div
            initial={{ opacity: 0, y: -8, scale: 0.96 }}
            animate={{ opacity: 1, y: 0, scale: 1 }}
            exit={{ opacity: 0, y: -8, scale: 0.96 }}
            transition={{ duration: 0.2, ease: "easeOut" }}
            className="absolute right-4 top-full z-50 mt-2 w-56 max-w-[calc(100vw-2rem)] origin-top-right rounded-lg border border-hairline bg-surface p-2 shadow-xl md:hidden"
          >
            <nav className="flex flex-col gap-1">
              {navLinks.map((link) => (
                <a
                  key={link.href}
                  href={link.href}
                  onClick={() => setOpen(false)}
                  className="flex min-h-[44px] items-center rounded-md px-3 py-2 text-sm font-semibold text-body transition hover:bg-surface-soft hover:text-foreground"
                >
                  {link.label}
                </a>
              ))}
            </nav>

            <div className="my-2 h-px bg-hairline" />

            <div className="flex flex-col gap-1">
              <ThemeToggle className="w-full justify-between" showLabel />
              <LanguageSwitch
                variant="inline"
                className="w-full"
                buttonClassName="w-full justify-between"
              />
              <a
                href="https://github.com/riffpad/riffpad"
                target="_blank"
                rel="noopener noreferrer"
                onClick={() => setOpen(false)}
                className="flex min-h-[44px] items-center justify-between rounded-md px-3 py-2 text-sm font-semibold text-body transition hover:bg-surface-soft hover:text-foreground"
              >
                <span>GitHub</span>
                <Github className="h-4 w-4" />
              </a>
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </header>
  );
}
