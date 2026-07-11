"use client";

import { useState } from "react";
import { motion } from "framer-motion";
import { useLanguage } from "./LanguageProvider";
import { WaitlistForm } from "./WaitlistForm";
import { Mail, Discord } from "./Icons";

export function Hero() {
  const { t } = useLanguage();
  const [copied, setCopied] = useState(false);

  const handleCopyEmail = async () => {
    try {
      await navigator.clipboard.writeText("hi@riffpad.ai");
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      // Fallback is silent; user can still see the address visually
    }
  };

  return (
    <section className="bg-background px-4 pb-20 pt-16 sm:px-6 sm:pt-20 lg:px-8">
      <div className="mx-auto max-w-3xl text-center">
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.6, ease: "easeOut" }}
        >
          <h1 className="text-4xl font-bold leading-tight text-foreground sm:text-5xl lg:text-[56px]">
            {t.hero.h1}
          </h1>

          <p className="mx-auto mt-5 max-w-2xl text-lg text-body sm:text-xl">{t.hero.description}</p>

          <div className="mx-auto mt-8 max-w-md">
            <WaitlistForm />
            <div className="mt-3 flex flex-wrap items-center justify-center gap-3">
              <a
                href="https://discord.gg/CDNFTg2QyM"
                target="_blank"
                rel="noopener noreferrer"
                className="inline-flex items-center gap-2 rounded-md border border-hairline bg-surface px-4 py-2 text-sm font-semibold text-body transition hover:border-ash hover:text-foreground"
              >
                <Discord className="h-4 w-4" />
                {t.hero.contact.discord}
              </a>
              <button
                type="button"
                onClick={handleCopyEmail}
                className="inline-flex items-center gap-2 rounded-md border border-hairline bg-surface px-4 py-2 text-sm font-semibold text-body transition hover:border-ash hover:text-foreground"
              >
                <Mail className="h-4 w-4" />
                {copied ? t.hero.contact.copied : t.hero.contact.email}
              </button>
            </div>
            <p className="mt-3 text-sm text-muted">{t.hero.trust}</p>
          </div>
        </motion.div>
      </div>
    </section>
  );
}
