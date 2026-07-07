"use client";

import { motion } from "framer-motion";
import { useLanguage } from "./LanguageProvider";

export function CTA() {
  const { t } = useLanguage();

  return (
    <section className="relative overflow-hidden bg-surface px-4 py-24 text-center sm:px-6 lg:px-8">
      <div className="pointer-events-none absolute inset-0 -z-10">
        <div className="absolute left-1/2 top-1/2 h-[500px] w-[500px] -translate-x-1/2 -translate-y-1/2 rounded-full bg-accent/10 blur-[120px]" />
      </div>

      <motion.div
        initial={{ opacity: 0, scale: 0.98 }}
        whileInView={{ opacity: 1, scale: 1 }}
        viewport={{ once: true }}
        transition={{ duration: 0.5 }}
        className="relative mx-auto max-w-3xl"
      >
        <h2 className="font-display text-3xl font-semibold tracking-tight text-foreground sm:text-4xl lg:text-5xl">
          {t.cta.title}
        </h2>
        <p className="mt-4 text-lg text-body">{t.cta.subtitle}</p>
        <div className="mt-10 flex flex-col items-center justify-center gap-4 sm:flex-row">
          <a
            href="https://app.riffpad.ai"
            className="rounded-full bg-foreground px-8 py-4 text-base font-medium text-background shadow-[0_0_40px_-12px_rgba(255,255,255,0.25)] transition hover:bg-white"
          >
            {t.cta.primary}
          </a>
          <a
            href="/docs"
            className="rounded-full border border-white/10 bg-background px-8 py-4 text-base font-medium text-foreground transition hover:border-white/20"
          >
            {t.cta.secondary}
          </a>
        </div>
        <p className="mt-4 text-sm text-muted">{t.cta.trust}</p>
      </motion.div>
    </section>
  );
}
