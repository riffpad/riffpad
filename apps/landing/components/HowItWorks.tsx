"use client";

import { useLanguage } from "./LanguageProvider";
import { BRAND_ICONS } from "./brand-icons";

// Pseudo-QR pairing pattern: 1 = ink cell, 2 = accent finder block, 0 = empty.
const QR_ROWS = [
  "2220100222",
  "2221011222",
  "2220110222",
  "0100101010",
  "1011010101",
  "0101101010",
  "1010010110",
  "2220101001",
  "2221010110",
  "2220011010",
];

function InstallVisual() {
  return (
    <div className="inline-flex items-center gap-4 border border-hairline bg-surface-muted px-4 py-3 text-mute">
      {BRAND_ICONS.map((icon) => (
        <svg
          key={icon.title}
          viewBox="0 0 24 24"
          className="h-5 w-5 fill-current transition-colors duration-200 hover:text-ink"
          role="img"
          aria-label={icon.title}
        >
          <title>{icon.title}</title>
          <path d={icon.path} />
        </svg>
      ))}
    </div>
  );
}

function PairVisual() {
  return (
    <div className="inline-flex flex-col gap-2 border border-hairline bg-surface-muted p-3">
      <div
        className="grid grid-cols-[repeat(10,8px)] gap-[2px]"
        aria-hidden="true"
      >
        {QR_ROWS.join("")
          .split("")
          .map((cell, i) => (
            <span
              key={i}
              className={`h-2 w-2 ${
                cell === "2" ? "bg-accent" : cell === "1" ? "bg-ink" : ""
              }`}
            />
          ))}
      </div>
      <div className="text-[11px] text-mute">pair · X25519 · ephemeral</div>
    </div>
  );
}

function NotifyVisual() {
  const { t } = useLanguage();
  return (
    <div className="border border-hairline p-3 text-xs shadow-card">
      <div className="flex items-center justify-between text-mute">
        <span className="font-bold text-ink">{t.mockup.phone.title}</span>
        <span>now</span>
      </div>
      <div className="mt-1.5 font-bold text-ink">{t.mockup.phone.approval}</div>
      <div className="mt-0.5 text-body">{t.mockup.phone.summary}</div>
    </div>
  );
}

const visuals = [InstallVisual, PairVisual, NotifyVisual];

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
            {t.how.steps.map((step, index) => {
              const Visual = visuals[index % visuals.length];
              return (
                <div
                  key={index}
                  className="card relative flex flex-col p-6 sm:p-8"
                >
                  <span className="text-3xl font-bold text-accent">
                    {String(index + 1).padStart(2, "0")}
                  </span>
                  <h3 className="mt-4 text-lg font-bold text-ink">
                    {step.title}
                  </h3>
                  <p className="mt-2 text-base leading-[1.7] text-body">
                    {step.description}
                  </p>
                  <div className="mt-auto pt-6">
                    <Visual />
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      </div>
    </section>
  );
}
