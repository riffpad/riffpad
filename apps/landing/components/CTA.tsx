"use client";

import { useState } from "react";
import { motion } from "framer-motion";
import { useLanguage } from "./LanguageProvider";

export function CTA() {
  const { t } = useLanguage();
  const [email, setEmail] = useState("");
  const [submitted, setSubmitted] = useState(false);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (email.trim()) {
      setSubmitted(true);
    }
  };

  return (
    <section className="bg-background px-4 py-20 sm:px-6 lg:px-8">
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        whileInView={{ opacity: 1, y: 0 }}
        viewport={{ once: true }}
        transition={{ duration: 0.5 }}
        className="mx-auto max-w-xl rounded-md border border-hairline bg-surface p-8 text-center sm:p-12"
      >
        <h2 className="text-3xl font-bold text-foreground sm:text-4xl">
          {t.cta.title}
        </h2>
        <p className="mt-3 text-lg text-body">{t.cta.subtitle}</p>

        {submitted ? (
          <p className="mt-6 text-base font-semibold text-accent">{t.cta.success}</p>
        ) : (
          <form onSubmit={handleSubmit} className="mt-6 flex flex-col gap-3 sm:flex-row">
            <input
              type="email"
              required
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder={t.cta.placeholder}
              className="flex-1 rounded-md border border-hairline bg-background px-4 py-3 text-base text-foreground outline-none transition placeholder:text-muted focus:border-accent"
            />
            <button
              type="submit"
              className="rounded-md bg-accent px-6 py-3 text-base font-bold text-on-accent transition hover:bg-accent-pressed"
            >
              {t.cta.primary}
            </button>
          </form>
        )}

        <div className="mt-5">
          <a
            href="mailto:hi@riffpad.ai"
            className="inline-flex items-center gap-2 text-sm font-semibold text-body transition hover:text-foreground"
          >
            {t.cta.emailUs}
          </a>
        </div>
      </motion.div>
    </section>
  );
}
