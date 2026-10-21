"use client";

import type { Messages } from "@/lib/i18n";
import type { Approval, TermLine } from "./types";

export function TermLineView({ line }: { line: TermLine }) {
  if (line.tone === "user") {
    return (
      <div className="mt-1 font-bold">
        <span className="mr-2 text-tui-mauve" aria-hidden="true">
          ❯
        </span>
        {line.text}
      </div>
    );
  }
  if (line.tone === "think") {
    return <div className="my-1 text-tui-mute">{line.text}</div>;
  }
  if (line.tone === "tool") {
    return (
      <div className="mt-1.5">
        <span className="mr-2 text-tui-green" aria-hidden="true">
          ●
        </span>
        <span className="font-bold">{line.text}</span>
      </div>
    );
  }
  if (line.tone === "sub") {
    return (
      <div className="pl-5 text-tui-mute">
        <span className="mr-2" aria-hidden="true">
          └
        </span>
        {line.text}
      </div>
    );
  }
  if (line.tone === "warn") {
    return (
      <div className="text-tui-red">
        <span className="mr-2" aria-hidden="true">
          ✗
        </span>
        {line.text}
      </div>
    );
  }
  if (line.tone === "agent") {
    return <div className="mt-1 text-tui-fg/80">{line.text}</div>;
  }
  return (
    <div className={line.tone === "cmd" ? "mt-1 text-tui-mauve" : undefined}>
      {line.tone === "ok" && (
        <span className="mr-2 text-tui-green" aria-hidden="true">
          ✓
        </span>
      )}
      {line.tone === "info" && (
        <span className="mr-2 text-tui-blue" aria-hidden="true">
          ▸
        </span>
      )}
      {line.text}
    </div>
  );
}

export function NvimPane({
  t,
  lines,
  approval,
}: {
  t: Messages;
  lines: TermLine[];
  approval: Approval;
}) {
  const m = t.mockup.tui;
  const cursor = 6;

  return (
    <div className="flex w-full flex-col overflow-hidden border border-tui-hairline bg-tui-bg text-tui-fg lg:h-full">
      {/* tabline: the one piece of chrome a real nvim window has */}
      <div className="flex items-center bg-tui-surface text-[10px] sm:text-[11px]">
        <span className="border-r border-tui-divider px-2 py-1 text-tui-fg">
          {m.file}
        </span>
        <span className="px-2 py-1 text-tui-mute" aria-hidden="true">
          {m.modified}
        </span>
      </div>

      <div className="flex-1 px-2 py-2 text-[11px] leading-[1.8] sm:text-[13px]">
        {lines.map((line, i) => (
          <div key={line.id} className="flex gap-2">
            <span
              className={`w-6 flex-none select-none text-right ${
                i === cursor ? "text-tui-fg" : "text-tui-mute"
              }`}
              aria-hidden="true"
            >
              {i + 1}
            </span>
            <div className="min-w-0 flex-1 truncate">
              <TermLineView line={line} />
            </div>
          </div>
        ))}

        {approval === "pending" && (
          <div className="mx-2 mt-2 border border-tui-blue/60 px-3 py-1.5">
            <div className="font-bold text-tui-blue">{m.approvalTitle}</div>
            <div className="font-bold">{m.approvalCmd}</div>
            <div className="mt-1 bg-tui-surface px-1">
              <span className="mr-1 text-tui-blue" aria-hidden="true">
                ❯
              </span>
              {m.approvalYes}
            </div>
            <div className="px-1 text-tui-fg/70">{m.approvalNo}</div>
          </div>
        )}
      </div>

      {/* statusline: mode block, file, then the right-hand segment */}
      <div className="flex items-center justify-between gap-2 border-t border-tui-divider bg-tui-surface text-[10px] font-bold sm:text-[11px]">
        <div className="flex min-w-0 items-center">
          <span className="flex-none bg-tui-mauve px-2 py-1 text-tui-bg">
            {m.mode}
          </span>
          <span className="truncate px-2 text-tui-fg/80">{m.file}</span>
        </div>
        <span className="flex-none pr-2 text-tui-mute">{m.statusRight}</span>
      </div>
    </div>
  );
}

export function ToolRow({
  text,
  state,
}: {
  text: string;
  state: "done" | "running";
}) {
  return (
    <div className="flex items-center gap-2 border border-tui-divider px-2.5 py-1.5 text-[11px] text-tui-fg/70">
      {state === "done" ? (
        <span className="h-1.5 w-1.5 flex-none bg-tui-green" aria-hidden="true" />
      ) : (
        <span className="h-1.5 w-1.5 flex-none bg-tui-mauve" aria-hidden="true" />
      )}
      <span className="flex-1 truncate">{text}</span>
      <span className="text-tui-mute" aria-hidden="true">
        ▸
      </span>
    </div>
  );
}
