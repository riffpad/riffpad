"use client";

import { useLanguage } from "./LanguageProvider";

function SyncVisual() {
  const { t } = useLanguage();
  const preset = t.mockup.phone.presets[1];
  return (
    <div className="flex flex-col gap-3 sm:flex-row">
      {/* agent → phone */}
      <div className="flex-1 border border-hairline bg-surface-muted p-3 text-xs leading-[1.9] text-body">
        <div className="text-mute">agent → phone</div>
        <div>
          <span className="mr-2 text-info" aria-hidden="true">
            ▸
          </span>
          edit_file src/auth/middleware.ts
        </div>
        <div>
          <span className="mr-2 text-success" aria-hidden="true">
            ✓
          </span>
          run_tests · 42 passed
          <span
            className="ml-1.5 inline-block h-3 w-[7px] animate-pulse bg-accent align-middle"
            aria-hidden="true"
          />
        </div>
      </div>
      {/* phone → agent */}
      <div className="flex flex-1 flex-col gap-2 text-xs leading-[1.6]">
        <div className="text-mute">phone → agent</div>
        <div className="ml-6 self-end rounded-[10px] rounded-br-[2px] bg-accent px-3 py-2 text-accent-ink">
          {preset.send}
        </div>
        <div className="mr-6 self-start rounded-[10px] rounded-bl-[2px] bg-surface-muted px-3 py-2 text-body">
          {preset.ack}
        </div>
      </div>
    </div>
  );
}

function ApprovalVisual() {
  const { t } = useLanguage();
  return (
    <div className="border border-warning/20 bg-warning/10 p-3 text-xs">
      <div className="text-warning">{t.mockup.phone.approval}</div>
      <div className="mt-1 font-bold text-ink">{t.mockup.phone.summary}</div>
      <div className="mt-3 flex gap-2">
        <span className="flex-1 border border-success/50 py-1 text-center font-bold text-success">
          {t.mockup.phone.approve}
        </span>
        <span className="flex-1 border border-danger/50 py-1 text-center font-bold text-danger">
          {t.mockup.phone.reject}
        </span>
      </div>
    </div>
  );
}

const visuals = [SyncVisual, ApprovalVisual];

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

        <div className="mt-12 grid gap-4 md:grid-cols-12">
          {t.features.items.map((item, index) => {
            const Visual = visuals[index % visuals.length];
            return (
              <div
                key={index}
                className={`card flex flex-col p-6 transition-colors duration-200 hover:border-hairline-strong sm:p-8 ${
                  index % 2 === 0 ? "md:col-span-7" : "md:col-span-5"
                }`}
              >
                <h3 className="text-lg font-bold text-ink">{item.title}</h3>
                <p className="mt-2 text-base leading-[1.7] text-body">
                  {item.description}
                </p>
                <div className="mt-auto pt-6">
                  <Visual />
                </div>
              </div>
            );
          })}
        </div>
      </div>
    </section>
  );
}
