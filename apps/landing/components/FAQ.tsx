"use client";

import { useState } from "react";
import { useLanguage } from "./LanguageProvider";
import { SectionLabel } from "./SectionLabel";

export function FAQ() {
  const { t } = useLanguage();
  const [open, setOpen] = useState<number | null>(0);

  return (
    <section
      id="faq"
      className="mx-auto max-w-frame scroll-mt-20 px-4 py-24 sm:px-6 sm:py-32"
    >
      <div className="mx-auto max-w-content">
        <SectionLabel>[-] {t.faq.label}</SectionLabel>
        <h2 className="mt-6 text-balance text-2xl font-bold leading-snug sm:text-3xl">
          {t.faq.title}
        </h2>

        <div className="mt-12 border-t border-hairline">
          {t.faq.items.map((item, index) => {
            const isOpen = open === index;
            return (
              <div key={index} className="border-b border-hairline">
                <button
                  type="button"
                  onClick={() => setOpen(isOpen ? null : index)}
                  className="flex w-full cursor-pointer items-start justify-between gap-4 py-5 text-left"
                  aria-expanded={isOpen}
                >
                  <span className="text-base font-bold">{item.q}</span>
                  <span className="text-mute" aria-hidden="true">
                    {isOpen ? "−" : "+"}
                  </span>
                </button>
                {isOpen && (
                  <p className="max-w-[680px] pb-5 text-base leading-[1.7] text-body">
                    {item.a}
                  </p>
                )}
              </div>
            );
          })}
        </div>
      </div>
    </section>
  );
}
