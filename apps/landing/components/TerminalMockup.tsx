"use client";

import { useLanguage } from "./LanguageProvider";

export function TerminalMockup() {
  const { t } = useLanguage();

  return (
    <div className="console-card overflow-hidden">
      <div className="flex flex-wrap items-center justify-between gap-3 border-b border-on-console/10 px-4 py-3 sm:px-5">
        <div className="flex items-center gap-3">
          <div className="flex gap-1.5" aria-hidden="true">
            <span className="h-2.5 w-2.5 rounded-full bg-on-console/15" />
            <span className="h-2.5 w-2.5 rounded-full bg-on-console/25" />
            <span className="h-2.5 w-2.5 rounded-full bg-accent/70" />
          </div>
          <span className="text-xs text-on-console-mute">{t.terminal.title}</span>
        </div>
        <span className="flex items-center gap-1.5 text-xs text-success">
          <span aria-hidden="true">●</span>
          {t.terminal.connection}
        </span>
      </div>

      <div className="px-4 py-5 sm:px-6">
        <div className="flex flex-wrap items-center justify-between gap-2 text-sm">
          <span className="text-on-console">
            {t.terminal.session}
            <span className="text-success"> {t.terminal.running}</span>
          </span>
          <span className="text-xs text-on-console-mute">84ms</span>
        </div>

        <div className="mx-auto mt-5 max-w-[260px] rounded-lg border border-on-console/15 bg-console-elevated p-3">
          <div className="flex items-center justify-between text-xs text-on-console-mute">
            <span>09:41</span>
            <span aria-hidden="true">●</span>
          </div>
          <div className="mt-3 rounded-sm border border-warning/30 bg-warning/10 p-3">
            <div className="text-[11px] text-warning">{t.terminal.approval}</div>
            <div className="mt-1.5 text-sm font-bold text-on-console">
              {t.terminal.approvalSummary}
            </div>
            <div className="mt-3 flex gap-2">
              <span className="inline-flex h-9 flex-1 items-center justify-center rounded-sm border border-success/50 text-xs font-bold text-success">
                {t.terminal.approve}
              </span>
              <span className="inline-flex h-9 flex-1 items-center justify-center rounded-sm border border-danger/50 text-xs font-bold text-danger">
                {t.terminal.reject}
              </span>
            </div>
          </div>
        </div>

        <div className="mt-5 flex flex-wrap gap-x-5 gap-y-1 text-xs text-on-console-mute">
          <span>{t.terminal.statusE2ee}</span>
          <span>{t.terminal.statusRelay}</span>
          <span>{t.terminal.statusLatency}</span>
        </div>
      </div>

      <div className="flex flex-wrap items-center justify-between gap-2 border-t border-on-console/10 px-4 py-3 text-xs text-on-console-mute sm:px-5">
        <span>{t.terminal.hintTab}</span>
        <span>{t.terminal.hintCmd}</span>
        <span>{t.terminal.hintEsc}</span>
      </div>
    </div>
  );
}
