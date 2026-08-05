"use client";

import { useLanguage } from "./LanguageProvider";

export function HowItWorks() {
  const { t } = useLanguage();

  return (
    <section
      id="how"
      className="mx-auto max-w-frame scroll-mt-20 px-4 py-24 sm:px-6 sm:py-32"
    >
      <div className="mx-auto max-w-content">
        <span className="label">{`// ${t.how.label}`}</span>
        <h2 className="mt-6 text-balance text-2xl font-bold leading-[1.25] tracking-[-0.01em] sm:text-3xl">
          {t.how.title}
        </h2>
        <p className="mt-3 text-base text-body">{t.how.subtitle}</p>

        <div className="relative mt-12">
          <div
            className="pointer-events-none absolute inset-x-6 top-1/2 hidden border-t border-dashed border-hairline md:block"
            aria-hidden="true"
          />
          <div className="grid gap-4 md:grid-cols-3">
            {t.how.steps.map((step, index) => (
              <div key={index} className="card relative p-6 sm:p-8">
                <span className="text-3xl font-bold text-accent">
                  {String(index + 1).padStart(2, "0")}
                </span>
                <h3 className="mt-4 text-lg font-bold text-ink">{step.title}</h3>
                <p className="mt-2 text-base leading-[1.7] text-body">
                  {step.description}
                </p>
              </div>
            ))}
          </div>
        </div>
      </div>
    </section>
  );
}
