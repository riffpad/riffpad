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
      <div className="mx-auto max-w-6xl">
        <div className="grid items-center gap-12 lg:grid-cols-2">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.6, ease: "easeOut" }}
          >
            <h1 className="text-4xl font-bold leading-tight text-foreground sm:text-5xl lg:text-[42px]">
              {t.hero.h1}
            </h1>

            <p className="mt-5 text-lg text-body">{t.hero.description}</p>

            <div className="mt-8 max-w-md">
              <WaitlistForm />
              <div className="mt-3 flex flex-wrap items-center gap-3">
                <button
                  type="button"
                  onClick={handleCopyEmail}
                  className="inline-flex items-center gap-2 rounded-md border border-hairline bg-surface px-4 py-2 text-sm font-semibold text-body transition hover:border-ash hover:text-foreground"
                >
                  <Mail className="h-4 w-4" />
                  {copied ? t.hero.contact.copied : t.hero.contact.email}
                </button>
                <a
                  href="https://discord.gg/CDNFTg2QyM"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="inline-flex items-center gap-2 rounded-md border border-hairline bg-surface px-4 py-2 text-sm font-semibold text-body transition hover:border-ash hover:text-foreground"
                >
                  <Discord className="h-4 w-4" />
                  {t.hero.contact.discord}
                </a>
              </div>
              <p className="mt-3 text-sm text-muted">{t.hero.trust}</p>
            </div>
          </motion.div>

          <motion.div
            initial={{ opacity: 0, y: 30 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.7, delay: 0.15 }}
            className="relative"
          >
            <ChatMockup />
          </motion.div>
        </div>
      </div>
    </section>
  );
}

function ChatMockup() {
  const { t } = useLanguage();

  return (
    <div className="overflow-hidden rounded-md border border-hairline bg-surface shadow-sm">
      <div className="flex items-center justify-between border-b border-hairline-soft px-4 py-3">
        <span className="text-sm font-semibold text-foreground">{t.hero.chat.title}</span>
        <span className="h-2 w-2 rounded-full bg-accent-green" />
      </div>
      <div className="space-y-4 p-4">
        <div className="flex justify-end">
          <div className="max-w-[80%] rounded-2xl rounded-br-sm bg-foreground px-4 py-2.5 text-sm text-background">
            {t.hero.chat.userMessage}
          </div>
        </div>
        <div className="flex justify-start">
          <div className="max-w-[90%] rounded-2xl rounded-bl-sm border border-hairline bg-surface-doc px-4 py-3 text-sm text-body">
            <p>{t.hero.chat.assistantIntro}</p>
            <div className="mt-2 rounded-md bg-surface-dark p-2 font-mono text-xs text-on-dark">
              {t.hero.chat.codeSnippet}
            </div>
            <p className="mt-2">{t.hero.chat.assistantOutro}</p>
          </div>
        </div>
      </div>
      <div className="border-t border-hairline-soft px-4 py-3">
        <div className="flex items-center gap-2 rounded-full border border-hairline bg-surface px-3 py-2">
          <span className="text-sm text-muted">{t.hero.chat.inputPlaceholder}</span>
          <span className="ml-auto text-xs text-accent">↵</span>
        </div>
      </div>
    </div>
  );
}
