"use client";

import { motion } from "framer-motion";
import { useLanguage } from "./LanguageProvider";

export function Comparison() {
  const { t, lang } = useLanguage();
  const c = t.comparison;

  return (
    <section id="comparison" className="bg-background px-4 py-24 sm:px-6 lg:px-8">
      <div className="mx-auto max-w-6xl">
        <div className="mx-auto max-w-2xl text-center">
          <p className="text-xs font-bold uppercase tracking-wider text-body">
            {c.eyebrow}
          </p>
          <h2 className="mt-2 text-3xl font-bold tracking-tight text-foreground sm:text-4xl">
            {c.title}
          </h2>
          <p className="mt-3 text-lg leading-relaxed text-body">{c.subtitle}</p>
        </div>

        <div className="mt-16 grid gap-8 lg:grid-cols-2">
          {c.tables.map((table, i) => (
            <motion.div
              key={`${lang}-${i}`}
              initial={{ opacity: 0, y: 30 }}
              whileInView={{ opacity: 1, y: 0 }}
              viewport={{ once: true, margin: "-80px" }}
              transition={{ duration: 0.6, delay: i * 0.1 }}
              className="group overflow-hidden rounded-2xl border border-hairline bg-surface shadow-lg transition-all duration-300 hover:-translate-y-1 hover:shadow-xl"
            >
              {/* Card header */}
              <div className="flex items-center gap-3 border-b border-hairline bg-surface-soft px-6 py-4">
                <span className="flex h-7 w-7 items-center justify-center rounded-lg bg-accent/10 text-[10px] font-bold text-accent">
                  vs
                </span>
                <h3 className="text-lg font-bold tracking-tight text-foreground">
                  {table.title}
                </h3>
              </div>

              {/* Two-column comparison */}
              <div className="grid grid-cols-1 sm:grid-cols-[1.15fr_1fr]">
                {/* Riffpad column */}
                <div className="bg-accent/10 p-5 sm:p-6">
                  <div className="mb-4 text-[10px] font-bold uppercase tracking-wider text-accent">
                    {table.headers[0]}
                  </div>
                  <ul className="space-y-4">
                    {table.rows.map((row, j) => (
                      <li key={j} className="flex items-start gap-3">
                        <span className="mt-0.5 flex h-5 w-5 flex-shrink-0 items-center justify-center rounded-full bg-accent-green/10 text-accent-green">
                          <CheckIcon className="h-3.5 w-3.5" />
                        </span>
                        <span className="text-sm font-semibold leading-snug text-foreground">
                          {row.riffpad}
                        </span>
                      </li>
                    ))}
                  </ul>
                </div>

                {/* Competitor column */}
                <div className="p-5 sm:p-6">
                  <div className="mb-4 text-[10px] font-bold uppercase tracking-wider text-muted">
                    {table.headers[1]}
                  </div>
                  <ul className="space-y-4">
                    {table.rows.map((row, j) => (
                      <li key={j} className="flex items-start gap-3">
                        <span className="mt-0.5 flex h-5 w-5 flex-shrink-0 items-center justify-center rounded-full bg-surface-soft text-muted">
                          <CrossIcon className="h-3 w-3" />
                        </span>
                        <span className="text-sm leading-snug text-body">
                          {row.other}
                        </span>
                      </li>
                    ))}
                  </ul>
                </div>
              </div>
            </motion.div>
          ))}
        </div>
      </div>
    </section>
  );
}

function CheckIcon({ className }: { className?: string }) {
  return (
    <svg
      className={className}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="3"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <polyline points="20 6 9 17 4 12" />
    </svg>
  );
}

function CrossIcon({ className }: { className?: string }) {
  return (
    <svg
      className={className}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2.5"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <path d="M18 6 6 18" />
      <path d="m6 6 12 12" />
    </svg>
  );
}
