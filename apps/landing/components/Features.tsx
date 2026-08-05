"use client";

import { useLanguage } from "./LanguageProvider";
import { SectionLabel } from "./SectionLabel";

export function Features() {
  const { t } = useLanguage();

  return (
    <section
      id="features"
      className="mx-auto max-w-frame scroll-mt-20 px-4 py-24 sm:px-6 sm:py-32"
    >
      <div className="mx-auto max-w-content">
        <SectionLabel>[+] {t.features.label}</SectionLabel>
        <h2 className="mt-6 text-balance text-2xl font-bold leading-snug sm:text-3xl">
          {t.features.title}
        </h2>
        <p className="mt-3 text-base text-body">{t.features.subtitle}</p>

        <div className="mt-12 border-t border-hairline">
          {t.features.items.map((item, index) => (
            <div key={index} className="border-b border-hairline py-5 sm:py-6">
              <h3 className="text-base font-bold text-ink">
                [+] {item.title}
              </h3>
              <p className="mt-2 max-w-[640px] text-base leading-[1.7] text-body">
                {item.description}
              </p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
