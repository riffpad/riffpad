"use client";

import { useState } from "react";
import { useLanguage } from "./LanguageProvider";

export function FAQ() {
  const { t } = useLanguage();
  const [open, setOpen] = useState<number | null>(0);

  return (
    <section
      id="faq"
      className="mx-auto max-w-frame scroll-mt-20 px-4 py-24 sm:px-6 sm:py-32"
    >
      <div className="mx-auto max-w-content">
        <span className="label">{`// ${t.faq.label}`}</span>
        <h2 className="mt-6 text-balance text-2xl font-bold leading-[1.25] tracking-[-0.01em] sm:text-3xl">
          {t.faq.title}
        </h2>

        <div className="card mt-12 divide-y divide-hairline p-2 sm:p-4">
          {t.faq.items.map((item, index) => {
            const isOpen = open === index;
            return (
              <div key={index}>
                <button
                  type="button"
                  onClick={() => setOpen(isOpen ? null : index)}
                  className="flex w-full cursor-pointer items-start justify-between gap-4 px-3 py-5 text-left"
                  aria-expanded={isOpen}
                >
                  <span className="text-base font-bold">{item.q}</span>
                  <span className="text-sm text-mute" aria-hidden="true">
                    {isOpen ? "v" : ">"}
                  </span>
                </button>
                {isOpen && (
                  <p className="max-w-[680px] px-3 pb-5 text-base leading-[1.7] text-body">
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
