"use client";

import { motion } from "framer-motion";
import { useLanguage } from "./LanguageProvider";

export function Hero() {
  const { t } = useLanguage();

  return (
    <section className="relative overflow-hidden bg-background px-4 pb-24 pt-32 text-center sm:px-6 sm:pt-40 lg:px-8">
      <div className="pointer-events-none absolute inset-0 -z-10">
        <div className="absolute left-1/2 top-0 h-[600px] w-[600px] -translate-x-1/2 rounded-full bg-accent/10 blur-[120px]" />
        <div className="absolute bottom-0 right-0 h-[500px] w-[500px] rounded-full bg-blue-600/10 blur-[120px]" />
      </div>

      <motion.div
        initial={{ opacity: 0, y: 24 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.7, ease: "easeOut" }}
        className="mx-auto max-w-4xl"
      >
        <div className="mb-6 inline-flex items-center gap-2 rounded-full border border-white/10 bg-surface px-4 py-1.5 text-sm font-medium text-accent">
          <span className="h-2 w-2 rounded-full bg-accent" />
          {t.hero.badge}
        </div>

        <h1 className="font-display text-5xl font-bold tracking-tight text-foreground sm:text-7xl lg:text-8xl">
          {t.hero.h1Before}
          <br />
          <span className="text-accent">{t.hero.h1After}</span>
        </h1>

        <p className="mx-auto mt-6 max-w-2xl text-lg text-muted sm:text-xl">
          {t.hero.description}
        </p>

        <div className="mt-10 flex flex-col items-center justify-center gap-4 sm:flex-row">
          <a
            href="https://app.riffpad.ai"
            className="rounded-xl bg-accent px-8 py-4 text-base font-semibold text-black shadow-[0_0_40px_-10px_rgba(200,255,0,0.4)] transition hover:bg-accent-hover"
          >
            {t.hero.primaryCta}
          </a>
          <a
            href="#features"
            className="rounded-xl border border-white/10 bg-surface px-8 py-4 text-base font-medium text-foreground transition hover:border-white/20 hover:bg-surface-elevated"
          >
            {t.hero.secondaryCta}
          </a>
        </div>

        <p className="mt-4 text-sm text-muted">{t.hero.trust}</p>
      </motion.div>

      <motion.div
        initial={{ opacity: 0, y: 48 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.8, delay: 0.2, ease: "easeOut" }}
        className="mx-auto mt-20 max-w-5xl"
      >
        <div className="relative overflow-hidden rounded-2xl border border-white/10 bg-surface shadow-2xl">
          <div className="flex items-center gap-2 border-b border-white/5 px-4 py-3">
            <span className="h-3 w-3 rounded-full bg-red-500/80" />
            <span className="h-3 w-3 rounded-full bg-yellow-500/80" />
            <span className="h-3 w-3 rounded-full bg-green-500/80" />
            <span className="ml-3 font-mono text-xs text-muted">
              amber-spark-9 — {t.hero.terminal.title}
            </span>
          </div>
          <div className="grid gap-px bg-white/5 sm:grid-cols-[240px_1fr]">
            <div className="hidden bg-background/50 p-4 text-left sm:block">
              <p className="mb-2 text-xs font-semibold uppercase tracking-wider text-muted">
                Files
              </p>
              <ul className="space-y-1.5 font-mono text-sm text-muted">
                <li>📁 workspace/</li>
                <li className="pl-4">index.html</li>
                <li className="pl-4">main.py</li>
                <li className="pl-4">style.css</li>
                <li>📁 .riffpad/</li>
              </ul>
            </div>
            <div className="min-h-[260px] bg-background p-4 text-left font-mono text-sm text-foreground">
              <span className="text-accent">$ </span>
              {t.hero.terminal.prompt}
              <br />
              <span className="text-muted">{t.hero.terminal.status}</span>
              <br />
              <span className="text-blue-400">{t.hero.terminal.ready}</span>{" "}
              <span className="underline decoration-white/20">{t.hero.terminal.url}</span>
            </div>
          </div>
        </div>
      </motion.div>
    </section>
  );
}
