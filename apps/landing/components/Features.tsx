"use client";

import { useLanguage } from "./LanguageProvider";

export function Features() {
  const { t } = useLanguage();

  return (
    <section
      id="features"
      className="mx-auto max-w-frame scroll-mt-20 px-4 py-24 sm:px-6 sm:py-32"
    >
      <div className="mx-auto max-w-content">
        <span className="label">{`// ${t.features.label}`}</span>
        <h2 className="mt-6 text-balance text-2xl font-bold leading-[1.25] tracking-[-0.01em] sm:text-3xl">
          {t.features.title}
        </h2>
        <p className="mt-3 text-base text-body">{t.features.subtitle}</p>

        <div className="mt-12 grid gap-4 sm:grid-cols-2">
          {t.features.items.map((item, index) => (
            <div
              key={index}
              className="card p-6 transition-colors duration-200 hover:border-hairline-strong sm:p-8"
            >
              <h3 className="text-lg font-bold text-ink">{item.title}</h3>
              <p className="mt-2 text-base leading-[1.7] text-body">
                {item.description}
              </p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
