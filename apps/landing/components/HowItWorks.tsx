"use client";

import { useLanguage } from "./LanguageProvider";
import { SectionLabel } from "./SectionLabel";

export function HowItWorks() {
  const { t } = useLanguage();

  return (
    <section
      id="how"
      className="mx-auto max-w-frame scroll-mt-20 px-4 py-24 sm:px-6 sm:py-32"
    >
      <div className="mx-auto max-w-content">
        <SectionLabel>[2] {t.how.label}</SectionLabel>
        <h2 className="mt-6 text-balance text-2xl font-bold leading-snug sm:text-3xl">
          {t.how.title}
        </h2>
        <p className="mt-3 text-base text-body">{t.how.subtitle}</p>

        <div className="mt-12 grid divide-y divide-hairline border border-hairline md:grid-cols-3 md:divide-x md:divide-y-0">
          {t.how.steps.map((step, index) => (
            <div key={index} className="p-6 sm:p-8">
              <div className="text-sm text-mute">[{index + 1}]</div>
              <h3 className="mt-4 text-lg font-bold">{step.title}</h3>
              <p className="mt-2 text-base leading-[1.7] text-body">
                {step.description}
              </p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
