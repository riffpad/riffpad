"use client";

import { motion } from "framer-motion";
import { Sparkles, Zap, Box, Bridge } from "./Icons";
import { useLanguage } from "./LanguageProvider";
import { Hedgehog } from "./Hedgehog";

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
      transition: { staggerChildren: 0.1 },
    },
  };

  const item = {
    hidden: { opacity: 0, y: 20 },
    show: { opacity: 1, y: 0 },
  };

  return (
    <section id="features" className="bg-background px-4 py-20 sm:px-6 lg:px-8">
      <div className="mx-auto max-w-6xl">
        <div className="mx-auto max-w-2xl text-center">
          <p className="text-xs font-bold uppercase tracking-wider text-body">
            {t.features.eyebrow}
          </p>
          <h2 className="mt-2 text-3xl font-bold text-foreground sm:text-4xl">
            {t.features.title}
          </h2>
          <p className="mt-3 text-body">{t.features.subtitle}</p>
        </div>

        <motion.div
          variants={container}
          initial="hidden"
          whileInView="show"
          viewport={{ once: true, margin: "-80px" }}
          className="mt-12 grid gap-4 sm:grid-cols-2 lg:grid-cols-4"
        >
          {features.map((feature, i) => {
            const Icon = feature.icon;
            return (
              <motion.div
                key={feature.title}
                variants={item}
                className="relative rounded-md border border-hairline bg-surface p-5 transition hover:border-hairline-strong"
              >
                {i === 3 && (
                  <div className="absolute -right-3 -top-3">
                    <Hedgehog className="h-10 w-10" />
                  </div>
                )}
                <span className="inline-block rounded-full bg-surface-soft px-2 py-0.5 text-xs font-bold text-body">
                  {feature.badge}
                </span>
                <div className="mt-3 flex h-9 w-9 items-center justify-center rounded-md bg-foreground text-background">
                  <Icon className="h-4 w-4" />
                </div>
                <h3 className="mt-3 text-lg font-bold text-foreground">
                  {feature.title}
                </h3>
                <p className="mt-1 text-sm leading-relaxed text-body">
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
