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
    <section className="bg-background px-4 py-20 sm:px-6 lg:px-8">
      <div className="mx-auto max-w-6xl">
        <div className="mx-auto max-w-2xl text-center">
          <p className="text-xs font-bold uppercase tracking-wider text-body">
            {t.testimonials.eyebrow}
          </p>
          <h2 className="mt-2 text-3xl font-bold text-foreground sm:text-4xl">
            {t.testimonials.title}
          </h2>
        </div>

        <motion.div
          initial="hidden"
          whileInView="visible"
          viewport={{ once: true, margin: "-80px" }}
          variants={{
            hidden: { opacity: 0 },
            visible: {
              opacity: 1,
              transition: { staggerChildren: 0.1 },
            },
          }}
          className="mt-12 grid gap-4 md:grid-cols-3"
        >
          {testimonials.map((item) => (
            <motion.div
              key={item.name}
              variants={{
                hidden: { opacity: 0, y: 20 },
                visible: { opacity: 1, y: 0 },
              }}
              className="rounded-md border border-hairline bg-surface p-6"
            >
              <p className="text-base leading-relaxed text-foreground">“{item.quote}”</p>
              <div className="mt-5 flex items-center gap-3">
                <span className="flex h-9 w-9 items-center justify-center rounded-md bg-foreground text-sm font-bold text-background">
                  {item.initials}
                </span>
                <div>
                  <p className="text-sm font-bold text-foreground">{item.name}</p>
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
