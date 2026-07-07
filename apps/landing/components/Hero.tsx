"use client";

import { motion } from "framer-motion";
import { useLanguage } from "./LanguageProvider";
import { MeshGradient } from "./MeshGradient";

export function Hero() {
  const { t } = useLanguage();

  return (
    <section className="relative overflow-hidden bg-background px-4 pb-24 pt-32 text-center sm:px-6 sm:pt-40 lg:px-8">
      <MeshGradient className="pointer-events-none absolute inset-0 -z-10 h-full w-full" />
      <div className="pointer-events-none absolute inset-0 -z-10 bg-gradient-to-b from-transparent via-background/60 to-background" />

      <motion.div
        initial={{ opacity: 0, y: 24 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.7, ease: "easeOut" }}
        className="relative mx-auto max-w-4xl"
      >
        <div className="mb-6 inline-flex items-center gap-2 rounded-full border border-white/10 bg-surface/80 px-4 py-1.5 text-sm font-medium text-foreground backdrop-blur-sm">
          <span className="font-mono text-xs text-accent">{"//"}</span>
          {t.hero.badge}
        </div>

        <h1 className="font-display text-5xl font-semibold tracking-tight text-foreground sm:text-7xl lg:text-8xl">
          {t.hero.h1Before}
          <br />
          <span className="bg-gradient-to-r from-cyan-400 via-blue-500 to-pink-500 bg-clip-text text-transparent">
            {t.hero.h1After}
          </span>
        </h1>

        <p className="mx-auto mt-6 max-w-2xl text-lg text-body sm:text-xl">
          {t.hero.description}
        </p>

        <div className="mt-10 flex flex-col items-center justify-center gap-4 sm:flex-row">
          <a
            href="https://app.riffpad.ai"
            className="rounded-full bg-foreground px-8 py-4 text-base font-medium text-background shadow-[0_0_40px_-12px_rgba(255,255,255,0.25)] transition hover:bg-white"
          >
            {t.hero.primaryCta}
          </a>
          <a
            href="#features"
            className="rounded-full border border-white/10 bg-surface px-8 py-4 text-base font-medium text-foreground transition hover:border-white/20 hover:bg-surface-elevated"
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
        className="relative mx-auto mt-20 max-w-5xl"
      >
        <div className="overflow-hidden rounded-xl border border-white/10 bg-surface shadow-2xl">
          <div className="flex items-center gap-2 border-b border-white/5 px-4 py-3">
            <span className="h-3 w-3 rounded-full bg-red-500/80" />
            <span className="h-3 w-3 rounded-full bg-yellow-500/80" />
            <span className="h-3 w-3 rounded-full bg-green-500/80" />
            <span className="ml-3 font-mono text-xs text-muted">
              amber-spark-9 — {t.hero.terminal.title}
            </span>
          </div>
          <div className="grid gap-px bg-white/5 sm:grid-cols-[240px_1fr]">
            <div className="hidden bg-background p-4 text-left sm:block">
              <p className="mb-2 font-mono text-xs font-medium uppercase tracking-wider text-muted">
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
            <div className="min-h-[260px] bg-black p-4 text-left font-mono text-sm text-foreground">
              <span className="text-accent">$ </span>
              {t.hero.terminal.prompt}
              <br />
              <span className="text-muted">{t.hero.terminal.status}</span>
              <br />
              <span className="text-cyan-400">{t.hero.terminal.ready}</span>{" "}
              <span className="underline decoration-white/20">{t.hero.terminal.url}</span>
            </div>
          </div>
        </div>
      </motion.div>
    </section>
  );
}
