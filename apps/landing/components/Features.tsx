"use client";

import { motion } from "framer-motion";
import { useLanguage } from "./LanguageProvider";
import {
  SkillsMockup,
  MobileVoiceMockup,
  ExportMockup,
} from "./mockups";

export function Features() {
  const { t, lang } = useLanguage();
  const blocks = t.features.blocks;

  const mockups = [
    <SkillsMockup key="m1" />,
    <MobileVoiceMockup key="m2" />,
    <ExportMockup key="m3" />,
  ];

  const order = [0, 2, 1];

  return (
    <section id="features" className="bg-background px-4 py-24 sm:px-6 lg:px-8">
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

        <div className="mt-20 space-y-32">
          {order.map((originalIndex, i) => {
            const block = blocks[originalIndex];
            const isFullWidth = true;
            return (
              <motion.div
                key={`${lang}-${i}`}
                initial={{ opacity: 0, y: 40 }}
                whileInView={{ opacity: 1, y: 0 }}
                viewport={{ once: true, margin: "-100px" }}
                transition={{ duration: 0.7, ease: "easeOut" }}
                className={`grid items-center gap-12 ${
                  isFullWidth ? "" : "lg:grid-cols-2"
                }`}
              >
                <div
                  className={
                    originalIndex === 2
                      ? "ml-auto max-w-2xl text-right"
                      : "max-w-2xl"
                  }
                >
                  <span className="text-xs font-bold uppercase tracking-wider text-accent">
                    0{i + 1}
                  </span>
                  <h3 className="mt-2 text-2xl font-bold text-foreground sm:text-3xl">
                    {block.title}
                  </h3>
                  <p className="mt-4 text-lg leading-relaxed text-body">
                    {block.description}
                  </p>
                </div>
                <div
                  className={
                    isFullWidth ? "" : i % 2 === 1 ? "lg:order-1" : ""
                  }
                >
                  {mockups[originalIndex]}
                </div>
              </motion.div>
            );
          })}
        </div>
      </div>
    </section>
  );
}
