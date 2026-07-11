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
          <h2 className="mt-2 text-3xl font-bold text-foreground sm:text-4xl">
            {c.title}
          </h2>
          <p className="mt-3 text-body">{c.subtitle}</p>
        </div>

        <div className="mt-16 space-y-16">
          {c.tables.map((table, i) => (
            <motion.div
              key={`${lang}-${i}`}
              initial={{ opacity: 0, y: 30 }}
              whileInView={{ opacity: 1, y: 0 }}
              viewport={{ once: true, margin: "-80px" }}
              transition={{ duration: 0.6, delay: i * 0.1 }}
              className="overflow-hidden rounded-xl border border-hairline bg-surface shadow-sm"
            >
              <h3 className="border-b border-hairline bg-surface-soft px-5 py-4 text-lg font-bold text-foreground">
                {table.title}
              </h3>

              <div className="overflow-x-auto">
                <table className="w-full min-w-[600px] text-left text-sm">
                  <thead className="border-b border-hairline bg-surface-doc">
                    <tr>
                      <th className="w-1/2 px-5 py-4 font-bold text-foreground">
                        {table.headers[0]}
                      </th>
                      <th className="w-1/2 px-5 py-4 font-bold text-body">
                        {table.headers[1]}
                      </th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-hairline-soft">
                    {table.rows.map((row, j) => (
                      <tr key={j}>
                        <td className="px-5 py-4 align-top text-foreground">{row.riffpad}</td>
                        <td className="px-5 py-4 align-top text-body">{row.other}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </motion.div>
          ))}
        </div>
      </div>
    </section>
  );
}
