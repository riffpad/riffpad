"use client";

import { useLanguage } from "./LanguageProvider";

export function TerminalMockup() {
  const { t } = useLanguage();

  return (
    <div className="mx-auto max-w-content overflow-hidden border border-hairline bg-surface-dark text-on-dark">
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-on-dark/10 px-4 py-3 text-xs text-on-dark-mute sm:px-5">
        <span>{t.terminal.title}</span>
        <span className="flex items-center gap-2 text-success">
          <span aria-hidden="true">●</span>
          {t.terminal.connection}
        </span>
      </div>

      <div className="px-4 py-6 text-sm leading-[1.8] sm:px-6 sm:py-8">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <span>
            {t.terminal.session}
            <span className="text-success">[{t.terminal.running}]</span>
          </span>
          <span className="text-on-dark-mute">84ms</span>
        </div>

        <div className="mt-4 rounded border border-[rgba(255,159,10,0.35)] bg-[rgba(255,159,10,0.08)] p-4 sm:p-5">
          <div className="text-xs text-warning">[!] {t.terminal.approval}</div>
          <div className="mt-2 text-base font-medium text-on-dark">
            $ {t.terminal.approvalSummary}
          </div>
          <div className="mt-4 flex flex-wrap gap-3">
            <span className="inline-flex h-10 items-center rounded border border-[rgba(48,209,88,0.6)] bg-[rgba(48,209,88,0.12)] px-4 text-success">
              [ {t.terminal.approve} ]
            </span>
            <span className="inline-flex h-10 items-center rounded border border-[rgba(255,59,48,0.6)] bg-[rgba(255,59,48,0.12)] px-4 text-danger">
              [ {t.terminal.reject} ]
            </span>
          </div>
        </div>

        <div className="mt-4 flex flex-wrap gap-x-6 gap-y-1 text-xs text-on-dark-mute">
          <span>+ {t.terminal.statusE2ee}</span>
          <span>+ {t.terminal.statusRelay}</span>
          <span>+ {t.terminal.statusLatency}</span>
        </div>
      </div>

      <div className="flex flex-wrap items-center justify-between gap-2 border-t border-on-dark/10 px-4 py-3 text-xs text-on-dark-mute sm:px-5">
        <span>{t.terminal.hintTab}</span>
        <span>{t.terminal.hintCmd}</span>
        <span>{t.terminal.hintEsc}</span>
      </div>
    </div>
  );
}
