"use client";

import { motion } from "framer-motion";
import { Sparkles, Zap, Box, Bridge } from "./Icons";
import { useLanguage } from "./LanguageProvider";

export function Features() {
  const { t } = useLanguage();

  const features = [
    {
      icon: Sparkles,
      title: t.features.items.zeroFriction.title,
      description: t.features.items.zeroFriction.description,
      badge: t.features.items.zeroFriction.badge,
    },
    {
      icon: Box,
      title: t.features.items.runnable.title,
      description: t.features.items.runnable.description,
      badge: t.features.items.runnable.badge,
    },
    {
      icon: Bridge,
      title: t.features.items.bridge.title,
      description: t.features.items.bridge.description,
      badge: t.features.items.bridge.badge,
    },
    {
      icon: Zap,
      title: t.features.items.safe.title,
      description: t.features.items.safe.description,
      badge: t.features.items.safe.badge,
    },
  ];

  const container = {
    hidden: { opacity: 0 },
    show: {
      opacity: 1,
      transition: { staggerChildren: 0.12 },
    },
  };

  const item = {
    hidden: { opacity: 0, y: 24 },
    show: { opacity: 1, y: 0 },
  };

  return (
    <section id="features" className="relative bg-surface px-4 py-24 sm:px-6 lg:px-8">
      <div className="pointer-events-none absolute inset-0 -z-10">
        <div className="absolute left-0 top-1/2 h-[400px] w-[400px] -translate-y-1/2 rounded-full bg-accent/5 blur-[100px]" />
      </div>

      <div className="mx-auto max-w-6xl">
        <div className="mx-auto max-w-2xl text-center">
          <p className="text-sm font-semibold uppercase tracking-widest text-accent">
            {t.features.eyebrow}
          </p>
          <h2 className="mt-3 font-display text-3xl font-bold tracking-tight text-foreground sm:text-4xl lg:text-5xl">
            {t.features.title}
          </h2>
          <p className="mt-4 text-muted">{t.features.subtitle}</p>
        </div>

        <motion.div
          variants={container}
          initial="hidden"
          whileInView="show"
          viewport={{ once: true, margin: "-100px" }}
          className="mt-16 grid gap-6 sm:grid-cols-2 lg:grid-cols-4"
        >
          {features.map((feature) => {
            const Icon = feature.icon;
            return (
              <motion.div
                key={feature.title}
                variants={item}
                className="group relative overflow-hidden rounded-2xl border border-white/10 bg-background p-6 transition hover:border-accent/30"
              >
                <div className="absolute -right-8 -top-8 h-32 w-32 rounded-full bg-accent/5 blur-2xl transition group-hover:bg-accent/10" />
                <span className="relative inline-block rounded-full border border-white/10 bg-surface px-2.5 py-1 text-xs font-medium text-accent">
                  {feature.badge}
                </span>
                <div className="relative mt-4 flex h-10 w-10 items-center justify-center rounded-lg bg-foreground text-black">
                  <Icon className="h-5 w-5" />
                </div>
                <h3 className="relative mt-4 text-lg font-semibold text-foreground">
                  {feature.title}
                </h3>
                <p className="relative mt-2 text-sm leading-relaxed text-muted">
                  {feature.description}
                </p>
              </motion.div>
            );
          })}
        </motion.div>
      </div>
    </section>
  );
}
