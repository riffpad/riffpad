"use client";

import { useState } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { Menu, X } from "./Icons";

const navLinks = [
  { label: "功能", href: "#features" },
  { label: "价格", href: "#pricing" },
  { label: "FAQ", href: "#faq" },
  { label: "文档", href: "/docs" },
];

export function Header() {
  const [open, setOpen] = useState(false);

  return (
    <header className="sticky top-0 z-50 border-b border-gray-200 bg-white/80 backdrop-blur-md">
      <div className="mx-auto flex max-w-6xl items-center justify-between px-4 py-4 sm:px-6 lg:px-8">
        <a href="/" className="flex items-center gap-2 text-lg font-bold text-gray-950">
          <span className="flex h-8 w-8 items-center justify-center rounded-lg bg-gray-950 text-white">
            R
          </span>
          Riffpad
        </a>

        <nav className="hidden items-center gap-8 text-sm font-medium text-gray-600 md:flex">
          {navLinks.map((link) => (
            <a
              key={link.href}
              href={link.href}
              className="transition hover:text-gray-950"
            >
              {link.label}
            </a>
          ))}
        </nav>

        <div className="hidden items-center gap-3 md:flex">
          <a
            href="https://app.riffpad.ai"
            className="rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white transition hover:bg-blue-700"
          >
            开始使用
          </a>
        </div>

        <button
          className="rounded-md p-2 text-gray-600 md:hidden"
          onClick={() => setOpen((v) => !v)}
          aria-label="切换菜单"
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
            className="overflow-hidden border-t border-gray-100 bg-white md:hidden"
          >
            <div className="flex flex-col gap-4 px-4 py-6 text-base font-medium text-gray-600">
              {navLinks.map((link) => (
                <a
                  key={link.href}
                  href={link.href}
                  onClick={() => setOpen(false)}
                  className="transition hover:text-gray-950"
                >
                  {link.label}
                </a>
              ))}
              <a
                href="https://app.riffpad.ai"
                className="mt-2 rounded-lg bg-blue-600 px-4 py-3 text-center text-white transition hover:bg-blue-700"
              >
                开始使用
              </a>
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </header>
  );
}
