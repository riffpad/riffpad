"use client";

import { motion } from "framer-motion";
import { useLanguage } from "./LanguageProvider";
import { Hedgehog } from "./Hedgehog";

export function Hero() {
  const { t } = useLanguage();

  return (
    <section className="bg-background px-4 pb-20 pt-16 sm:px-6 sm:pt-20 lg:px-8">
      <div className="mx-auto max-w-6xl">
        <div className="grid items-center gap-12 lg:grid-cols-2">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.6, ease: "easeOut" }}
          >
            <div className="mb-4 inline-flex items-center gap-2 rounded-md bg-surface px-3 py-1.5 text-sm font-bold text-body shadow-sm">
              🦔 {t.hero.badge}
            </div>

            <h1 className="text-4xl font-bold leading-tight text-foreground sm:text-5xl lg:text-[42px]">
              {t.hero.h1Before}
              <br />
              <span className="text-accent">{t.hero.h1After}</span>
            </h1>

            <p className="mt-5 text-lg text-body">{t.hero.description}</p>

            <div className="mt-8 flex flex-col items-start gap-3 sm:flex-row">
              <a
                href="https://app.riffpad.ai"
                className="rounded-md bg-accent px-6 py-3 text-base font-bold text-on-accent transition hover:bg-accent-pressed"
              >
                {t.hero.primaryCta}
              </a>
              <a
                href="#features"
                className="rounded-md bg-surface-soft px-6 py-3 text-base font-semibold text-foreground transition hover:bg-hairline-soft"
              >
                {t.hero.secondaryCta}
              </a>
            </div>

            <p className="mt-3 text-sm text-muted">{t.hero.trust}</p>
          </motion.div>

          <motion.div
            initial={{ opacity: 0, y: 30 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.7, delay: 0.15 }}
            className="relative"
          >
            <div className="overflow-hidden rounded-md border border-hairline bg-surface shadow-sm">
              <div className="flex items-center gap-2 border-b border-hairline-soft px-4 py-3">
                <span className="h-3 w-3 rounded-full bg-accent-red" />
                <span className="h-3 w-3 rounded-full bg-accent" />
                <span className="h-3 w-3 rounded-full bg-accent-green" />
                <span className="ml-3 font-mono text-xs text-muted">
                  amber-spark-9 — {t.hero.terminal.title}
                </span>
              </div>
              <div className="grid gap-px bg-hairline-soft sm:grid-cols-[220px_1fr]">
                <div className="hidden bg-surface-doc p-4 text-left sm:block">
                  <p className="mb-2 font-mono text-xs font-bold uppercase text-muted">
                    Files
                  </p>
                  <ul className="space-y-1.5 font-mono text-sm text-body">
                    <li>📁 workspace/</li>
                    <li className="pl-4">index.html</li>
                    <li className="pl-4">main.py</li>
                    <li className="pl-4">style.css</li>
                    <li>📁 .riffpad/</li>
                  </ul>
                </div>
                <div className="min-h-[240px] bg-surface-dark p-4 text-left font-mono text-sm text-white">
                  <span className="text-accent">$ </span>
                  {t.hero.terminal.prompt}
                  <br />
                  <span className="text-hairline">{t.hero.terminal.status}</span>
                  <br />
                  <span className="text-accent-blue-soft">{t.hero.terminal.ready}</span>{" "}
                  <span className="underline decoration-hairline">{t.hero.terminal.url}</span>
                </div>
              </div>
            </div>

            <div className="absolute -bottom-4 -right-4 hidden lg:block">
              <Hedgehog className="h-20 w-20" />
            </div>
          </motion.div>
        </div>
      </div>
    </section>
  );
}
