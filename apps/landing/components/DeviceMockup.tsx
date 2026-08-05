"use client";

import { useLanguage } from "./LanguageProvider";

export function DeviceMockup() {
  return (
    <div className="flex flex-col items-center justify-center gap-8 lg:flex-row lg:gap-6">
      <div className="w-full max-w-[460px]">
        <MacTerminal />
      </div>
      <SyncConnector />
      <div className="w-full max-w-[300px]">
        <PhoneApp />
      </div>
    </div>
  );
}

function MacTerminal() {
  const { t } = useLanguage();
  const lines = t.mockup.mac.lines;

  return (
    <div className="console-card w-full overflow-hidden">
      <div className="flex items-center gap-2 border-b border-on-console/10 px-4 py-3">
        <span className="h-3 w-3 rounded-full bg-[#ff5f57]" aria-hidden="true" />
        <span className="h-3 w-3 rounded-full bg-[#febc2e]" aria-hidden="true" />
        <span className="h-3 w-3 rounded-full bg-[#28c840]" aria-hidden="true" />
        <span className="ml-3 truncate text-xs text-on-console-mute">
          {t.mockup.mac.title}
        </span>
        <span className="ml-auto flex items-center gap-1.5 text-xs text-success">
          <span aria-hidden="true">●</span>
          {t.mockup.phone.synced}
        </span>
      </div>

      <div className="px-4 py-4 text-sm leading-[1.9] sm:px-5">
        <div className="text-on-console-mute">{t.mockup.mac.prompt}</div>
        {lines.map((line, index) => (
          <div
            key={index}
            className={
              line.tone === "warn" ? "text-warning" : "text-on-console"
            }
          >
            {line.tone === "ok" && (
              <span className="mr-2 text-success" aria-hidden="true">
                ✓
              </span>
            )}
            {line.tone === "info" && (
              <span className="mr-2 text-info" aria-hidden="true">
                ▸
              </span>
            )}
            {line.tone === "warn" && (
              <span className="mr-2" aria-hidden="true">
                !
              </span>
            )}
            {line.text}
          </div>
        ))}
        <div className="mt-2 text-xs text-on-console-mute">
          {t.mockup.mac.status}
        </div>
      </div>
    </div>
  );
}

function SyncConnector() {
  const { t } = useLanguage();

  return (
    <div
      className="flex flex-col items-center gap-3 lg:flex-row lg:gap-4"
      aria-hidden="true"
    >
      <div className="relative h-16 w-px bg-hairline-strong lg:h-px lg:w-20">
        <span className="absolute left-1/2 top-1/2 h-2 w-2 -translate-x-1/2 -translate-y-1/2 animate-pulse rounded-full bg-accent" />
      </div>
      <div className="text-center text-[11px] leading-5 text-mute">
        <div>{t.mockup.sync.label}</div>
        <div className="text-success">{t.mockup.sync.status}</div>
        <div>{t.mockup.sync.latency}</div>
      </div>
    </div>
  );
}

function PhoneApp() {
  const { t } = useLanguage();

  return (
    <div className="w-full border border-hairline bg-surface p-3 shadow-card">
      <div className="flex items-center justify-between px-2 pt-1 text-xs text-mute">
        <span>09:41</span>
        <span aria-hidden="true">●●●</span>
      </div>

      <div className="mt-3 flex items-center justify-between border-b border-hairline px-2 pb-3">
        <span className="text-sm font-bold">{t.mockup.phone.title}</span>
        <span className="flex items-center gap-1.5 text-xs text-success">
          <span aria-hidden="true">●</span>
          {t.mockup.phone.synced}
        </span>
      </div>

      <div className="mt-3 border border-hairline p-3">
        <div className="flex items-center justify-between text-xs text-mute">
          <span>{t.mockup.phone.session}</span>
          <span className="text-success">{t.mockup.phone.running}</span>
        </div>
        <div className="mt-1 text-sm font-bold">
          {t.mockup.phone.cli} · {t.mockup.phone.tools}
        </div>

        <div className="mt-3 border border-warning/30 bg-warning/10 p-3">
          <div className="text-[11px] text-warning">
            {t.mockup.phone.approval}
          </div>
          <div className="mt-1 text-sm font-bold">
            {t.mockup.phone.summary}
          </div>
          <div className="mt-3 flex gap-2">
            <span className="inline-flex h-9 flex-1 items-center justify-center border border-success/50 text-xs font-bold text-success">
              {t.mockup.phone.approve}
            </span>
            <span className="inline-flex h-9 flex-1 items-center justify-center border border-danger/50 text-xs font-bold text-danger">
              {t.mockup.phone.reject}
            </span>
          </div>
        </div>

        <div className="mt-3 border border-hairline px-3 py-2.5 text-xs text-mute">
          {t.mockup.phone.input}
        </div>
      </div>

      <div className="mt-3 flex border-t border-hairline pt-2 text-[11px] text-mute">
        {t.mockup.phone.tabs.map((tab, index) => (
          <span
            key={index}
            className={
              index === 1
                ? "flex-1 text-center font-bold text-ink"
                : "flex-1 text-center"
            }
          >
            {tab}
          </span>
        ))}
      </div>
    </div>
  );
}
