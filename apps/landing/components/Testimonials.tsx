"use client";

import { motion } from "framer-motion";
import { useLanguage } from "./LanguageProvider";

export function Testimonials() {
  const { t } = useLanguage();

  const testimonials = [
    {
      quote: t.testimonials.people.indie.quote,
      name: t.testimonials.people.indie.name,
      role: t.testimonials.people.indie.role,
      initials: t.testimonials.people.indie.name.slice(0, 2),
    },
    {
      quote: t.testimonials.people.pm.quote,
      name: t.testimonials.people.pm.name,
      role: t.testimonials.people.pm.role,
      initials: t.testimonials.people.pm.name.slice(0, 2),
    },
    {
      quote: t.testimonials.people.ai.quote,
      name: t.testimonials.people.ai.name,
      role: t.testimonials.people.ai.role,
      initials: t.testimonials.people.ai.name.slice(0, 2),
    },
  ];

  return (
    <section className="relative bg-background px-4 py-24 sm:px-6 lg:px-8">
      <div className="mx-auto max-w-6xl">
        <div className="mx-auto max-w-2xl text-center">
          <p className="font-mono text-sm font-medium uppercase tracking-wider text-muted">
            {t.testimonials.eyebrow}
          </p>
          <h2 className="mt-3 font-display text-3xl font-semibold tracking-tight text-foreground sm:text-4xl lg:text-5xl">
            {t.testimonials.title}
          </h2>
        </div>

        <motion.div
          initial="hidden"
          whileInView="visible"
          viewport={{ once: true, margin: "-100px" }}
          variants={{
            hidden: { opacity: 0 },
            visible: {
              opacity: 1,
              transition: { staggerChildren: 0.1 },
            },
          }}
          className="mt-16 grid gap-6 md:grid-cols-3"
        >
          {testimonials.map((item) => (
            <motion.div
              key={item.name}
              variants={{
                hidden: { opacity: 0, y: 24 },
                visible: { opacity: 1, y: 0 },
              }}
              className="relative rounded-xl border border-white/10 bg-surface p-6 transition hover:border-white/20"
            >
              <div className="absolute -right-4 -top-4 h-24 w-24 rounded-full bg-accent/5 blur-2xl" />
              <p className="relative text-base leading-relaxed text-foreground/90">“{item.quote}”</p>
              <div className="relative mt-6 flex items-center gap-3">
                <span className="flex h-10 w-10 items-center justify-center rounded-full bg-foreground text-xs font-bold text-black">
                  {item.initials}
                </span>
                <div>
                  <p className="text-sm font-semibold text-foreground">{item.name}</p>
                  <p className="text-xs text-muted">{item.role}</p>
                </div>
              </div>
            </motion.div>
          ))}
        </motion.div>
      </div>
    </section>
  );
}
