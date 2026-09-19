"use client";

import type { RefObject } from "react";
import type { Messages } from "@/lib/i18n";
import type { Approval, TermLine } from "./types";

export function TermLineView({ line }: { line: TermLine }) {
  if (line.tone === "user") {
    // full-width highlighted bar, like the real TUI renders a sent prompt
    return (
      <div className="-mx-4 my-1.5 bg-term-elevate-strong px-4 py-1.5 sm:-mx-5 sm:px-5">
        <span className="mr-2 text-term-mute" aria-hidden="true">
          ❯
        </span>
        {line.text}
      </div>
    );
  }
  if (line.tone === "think") {
    return <div className="my-1 text-term-mute">{line.text}</div>;
  }
  if (line.tone === "tool") {
    return (
      <div className="mt-2">
        <span className="mr-2 text-term-green" aria-hidden="true">
          ●
        </span>
        <span className="font-bold">{line.text}</span>
      </div>
    );
  }
  if (line.tone === "sub") {
    return (
      <div className="ml-6 text-term-mute">
        <span className="mr-2" aria-hidden="true">
          └
        </span>
        {line.text}
      </div>
    );
  }
  if (line.tone === "warn") {
    return (
      <div className="text-term-yellow">
        <span className="mr-2" aria-hidden="true">
          !
        </span>
        {line.text}
      </div>
    );
  }
  if (line.tone === "agent") {
    return <div className="mt-1 text-term-fg/80">{line.text}</div>;
  }
  return (
    <div className={line.tone === "cmd" ? "mt-1 text-term-orange" : undefined}>
      {line.tone === "ok" && (
        <span className="mr-2 text-term-green" aria-hidden="true">
          ✓
        </span>
      )}
      {line.tone === "info" && (
        <span className="mr-2 text-term-blue" aria-hidden="true">
          ▸
        </span>
      )}
      {line.text}
    </div>
  );
}

export function MacTerminal({
  t,
  lines,
  approval,
  scrollRef,
}: {
  t: Messages;
  lines: TermLine[];
  approval: Approval;
  scrollRef: RefObject<HTMLDivElement>;
}) {
  const m = t.mockup.mac;
  return (
    <div className="w-full overflow-hidden rounded-[10px] border border-term-border bg-term-bg text-term-fg shadow-terminal">
      {/* macOS traffic lights, no title */}
      <div className="flex items-center border-b border-term-border bg-term-bar px-4 py-2.5">
        <div className="flex items-center gap-2" aria-hidden="true">
          <span className="h-3 w-3 rounded-full bg-mac-red ring-1 ring-inset ring-black/15" />
          <span className="h-3 w-3 rounded-full bg-mac-yellow ring-1 ring-inset ring-black/15" />
          <span className="h-3 w-3 rounded-full bg-mac-green ring-1 ring-inset ring-black/15" />
        </div>
      </div>

      <div
        ref={scrollRef}
        className="no-scrollbar h-[300px] overflow-y-auto px-4 py-4 text-[13px] leading-[1.8] sm:px-5"
      >
        {lines.map((line) => (
          <TermLineView key={line.id} line={line} />
        ))}
        {approval === "pending" && (
          <div className="-mx-4 mt-2 border-t-2 border-term-blue bg-term-elevate px-4 py-3 sm:-mx-5 sm:px-5">
            <div className="font-bold text-term-blue">{m.approvalTitle}</div>
            <div className="mt-2 font-bold">{m.approvalCmd}</div>
            <div className="text-term-mute">{m.approvalDesc}</div>
            <div className="mt-3">{m.approvalQuestion}</div>
            <div className="-mx-4 mt-1 bg-term-elevate-strong px-4 sm:-mx-5 sm:px-5">
              <span className="inline-block w-[2ch] text-term-blue" aria-hidden="true">
                ❯
              </span>
              {m.approvalOpt1}
            </div>
            <div className="text-term-fg/80">
              <span className="inline-block w-[2ch]" aria-hidden="true" />
              {m.approvalOpt2}
            </div>
            <div className="text-term-fg/80">
              <span className="inline-block w-[2ch]" aria-hidden="true" />
              {m.approvalOpt3}
            </div>
            <div className="mt-3 text-[11px] text-term-mute">
              {m.approvalFooter}
            </div>
          </div>
        )}
      </div>

      {/* TUI-style input — same bg as the terminal, no placeholder/hints */}
      <div className="px-4 pb-4 pt-1 sm:px-5">
        <div className="flex items-center gap-2 border border-term-border px-3 py-2 text-[13px]">
          <span className="text-term-green" aria-hidden="true">
            ❯
          </span>
          <span
            className="h-3.5 w-[7px] animate-pulse bg-term-green"
            aria-hidden="true"
          />
        </div>
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
    <div className="flex items-center gap-2 border border-hairline px-2.5 py-1.5 text-[11px] text-body">
      {state === "done" ? (
        <span className="h-1.5 w-1.5 flex-none bg-success" aria-hidden="true" />
      ) : (
        <span className="flex flex-none items-center gap-[3px]" aria-hidden="true">
          {[0, 1, 2].map((i) => (
            <span
              key={i}
              className="h-1 w-1 animate-bounce bg-warning"
              style={{ animationDelay: `${i * 0.15}s` }}
            />
          ))}
        </span>
      )}
      <span className="flex-1 truncate">{text}</span>
      <span className="text-mute" aria-hidden="true">
        ▸
      </span>
    </div>
  );
}

