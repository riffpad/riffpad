"use client";

import { motion } from "framer-motion";
import { useLanguage } from "./LanguageProvider";

export function CTA() {
  const { t } = useLanguage();

  return (
    <section className="bg-background px-4 py-20 sm:px-6 lg:px-8">
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        whileInView={{ opacity: 1, y: 0 }}
        viewport={{ once: true }}
        transition={{ duration: 0.5 }}
        className="mx-auto max-w-3xl rounded-md border border-hairline bg-surface p-8 text-center sm:p-12"
      >
        <h2 className="text-3xl font-bold text-foreground sm:text-4xl">
          {t.cta.title}
        </h2>
        <p className="mt-3 text-lg text-body">{t.cta.subtitle}</p>
        <div className="mt-6 flex flex-col items-center justify-center gap-3 sm:flex-row">
          <a
            href="https://app.riffpad.ai"
            className="rounded-md bg-accent px-6 py-3 text-base font-bold text-on-accent transition hover:bg-accent-pressed"
          >
            {t.cta.primary}
          </a>
          <a
            href="/docs"
            className="rounded-md bg-surface-soft px-6 py-3 text-base font-semibold text-foreground transition hover:bg-hairline-soft"
          >
            {t.cta.secondary}
          </a>
        </div>
        <p className="mt-3 text-sm text-muted">{t.cta.trust}</p>
      </motion.div>
    </section>
  );
}
