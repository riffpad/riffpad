"use client";

import { useLanguage } from "./LanguageProvider";

export function Security() {
  const { t } = useLanguage();

  return (
    <section
      id="security"
      className="mx-auto max-w-frame scroll-mt-20 px-4 py-24 sm:px-6 sm:py-32"
    >
      <div className="mx-auto max-w-content">
        <div className="grid items-start gap-10 lg:grid-cols-2">
          <div>
            <span className="label">{`// ${t.security.label}`}</span>
            <h2 className="mt-6 text-balance text-2xl font-bold leading-[1.25] tracking-[-0.01em] sm:text-3xl">
              {t.security.title}
            </h2>
            <p className="mt-3 max-w-[520px] text-base text-body">
              {t.security.subtitle}
            </p>

            <div className="mt-8 border-t border-hairline">
              {t.security.items.map((item, index) => (
                <div key={index} className="border-b border-hairline py-4">
                  <h3 className="text-base font-bold text-ink">
                    {item.title}
                  </h3>
                  <p className="mt-1.5 max-w-[520px] text-base leading-[1.7] text-body">
                    {item.description}
                  </p>
                </div>
              ))}
            </div>
          </div>

          <div className="console-card overflow-hidden">
            <div className="border-b border-hairline px-4 py-3 text-xs text-on-console-mute sm:px-5">
              riffpad status --security
            </div>
            <div className="px-4 py-5 text-sm leading-[2] sm:px-5">
              {t.security.items.map((item, index) => (
                <div key={index} className="flex gap-3">
                  <span className="shrink-0 text-success" aria-hidden="true">
                    ●
                  </span>
                  <span className="text-on-console">{item.title}</span>
                </div>
              ))}
              <div className="flex gap-3">
                <span className="shrink-0 text-warning" aria-hidden="true">
                  ●
                </span>
                <span className="text-on-console-mute">{t.terminal.statusE2ee}</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
