"use client";

import { useLanguage } from "./LanguageProvider";
import { DeviceMockup } from "./DeviceMockup";
import { InstallCommand } from "./InstallCommand";
import { ScrollReveal } from "./ScrollReveal";

export function Hero() {
  const { t } = useLanguage();

  return (
    <section
      id="top"
      className="mx-auto max-w-frame scroll-mt-20 px-4 pb-12 pt-10 sm:px-6 sm:pb-28 sm:pt-20"
    >
      <div className="mx-auto max-w-content text-center">
        <ScrollReveal>
          <h1 className="text-balance text-[28px] font-bold leading-[1.15] tracking-[-0.02em] sm:text-[40px]">
            {t.hero.title1}
            <br />
            {t.hero.title2}
          </h1>
        </ScrollReveal>
        <ScrollReveal delay={100}>
          <p className="mx-auto mt-6 max-w-[560px] text-base leading-[1.7] text-body">
            {t.hero.description}
          </p>
        </ScrollReveal>
        <ScrollReveal delay={200}>
          <div className="mt-8 flex flex-col items-center justify-center gap-3 sm:flex-row">
            <a
              href="https://app.riffpad.ai"
              target="_blank"
              rel="noreferrer"
              className="btn btn-primary w-full sm:w-auto"
            >
              {t.hero.ctaPrimary}
            </a>
          </div>
        </ScrollReveal>
        <ScrollReveal delay={300}>
          <InstallCommand />
        </ScrollReveal>
      </div>

      <ScrollReveal delay={400}>
        <div className="mt-16 sm:mt-20">
          <DeviceMockup />
        </div>
      </ScrollReveal>
    </section>
  );
}
