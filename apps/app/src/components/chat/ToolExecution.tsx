"use client";

import { useState } from "react";
import { ChevronDown, ChevronRight, Wrench } from "lucide-react";

interface ToolExecutionProps {
  toolName: string;
  args: unknown;
  result?: unknown;
  isError?: boolean;
  isPartial?: boolean;
}

function safeStringify(value: unknown): string {
  if (value === undefined) return "";
  try {
    return JSON.stringify(value, null, 2);
  } catch {
    return String(value);
  }
}

export function ToolExecution({
  toolName,
  args,
  result,
  isError,
  isPartial,
}: ToolExecutionProps) {
  const [expanded, setExpanded] = useState(false);

  const statusClass = isError
    ? "bg-accent-red-soft border-accent-red/20"
    : isPartial
    ? "bg-accent-blue-soft border-accent-blue/20"
    : "bg-accent-green-soft border-accent-green/20";

  const statusText = isError ? "Error" : isPartial ? "Running" : "Done";

  return (
    <div className={`ml-7 rounded-md border ${statusClass} overflow-hidden`}>
      <button
        onClick={() => setExpanded((v) => !v)}
        className="w-full flex items-center justify-between px-3 py-2 text-left"
      >
        <div className="flex items-center gap-2 min-w-0">
          <Wrench className="h-3.5 w-3.5 text-mute shrink-0" />
          <span className="text-xs font-semibold text-ink truncate">{toolName}</span>
          <span
            className={`text-[10px] font-bold uppercase tracking-wide px-1.5 py-0.5 rounded ${
              isError
                ? "bg-accent-red text-white"
                : isPartial
                ? "bg-accent-blue text-white"
                : "bg-accent-green text-white"
            }`}
          >
            {statusText}
          </span>
        </div>
        {expanded ? (
          <ChevronDown className="h-3.5 w-3.5 text-mute shrink-0" />
        ) : (
          <ChevronRight className="h-3.5 w-3.5 text-mute shrink-0" />
        )}
      </button>
      {expanded && (
        <div className="px-3 pb-3 space-y-2 border-t border-hairline-soft/50">
          <div>
            <p className="text-[10px] font-bold uppercase tracking-wide text-mute mb-1">Arguments</p>
            <pre className="text-[11px] font-mono text-body bg-card/60 rounded p-2 overflow-x-auto">
              {safeStringify(args)}
            </pre>
          </div>
          {result !== undefined && (
            <div>
              <p className="text-[10px] font-bold uppercase tracking-wide text-mute mb-1">Result</p>
              <pre className="text-[11px] font-mono text-body bg-card/60 rounded p-2 overflow-x-auto">
                {safeStringify(result)}
              </pre>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
