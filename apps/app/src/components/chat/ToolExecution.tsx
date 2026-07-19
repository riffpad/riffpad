"use client";

import { useState } from "react";
import { ChevronDown, ChevronRight } from "lucide-react";

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

function toolTarget(args: unknown): string | undefined {
  if (!args || typeof args !== "object") return undefined;
  const a = args as Record<string, unknown>;
  if (typeof a.path === "string") return a.path;
  if (typeof a.directory === "string") return a.directory;
  if (typeof a.command === "string") return a.command;
  if (typeof a.query === "string") return a.query;
  return undefined;
}

function toolLabel(toolName: string, isPartial: boolean): string {
  if (!isPartial) return "Done";
  switch (toolName) {
    case "file_read":
      return "Reading";
    case "file_write":
      return "Writing";
    case "file_list":
      return "Listing files";
    case "bash_exec":
      return "Running";
    case "web_search":
      return "Searching";
    default:
      return "Running";
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

  const target = toolTarget(args);
  const label = toolLabel(toolName, !!isPartial);
  const statusText = isError ? "Failed" : target ? `${label} ${target}` : label;

  const dotClass = isError
    ? "bg-accent-red"
    : isPartial
    ? "bg-primary"
    : "bg-accent-green";

  return (
    <div className="ml-7 min-w-0">
      <button
        onClick={() => setExpanded((v) => !v)}
        className="group flex w-full min-w-0 items-center gap-2 text-left"
      >
        <span
          className={`h-2 w-2 shrink-0 rounded-full ${dotClass} ${
            isPartial ? "animate-pulse" : ""
          }`}
          aria-hidden="true"
        />
        <span className="min-w-0 flex-1 truncate text-xs text-mute transition group-hover:text-ink">
          {statusText}
        </span>
        <span className="shrink-0 text-mute opacity-0 transition group-hover:opacity-100">
          {expanded ? (
            <ChevronDown className="h-3.5 w-3.5" />
          ) : (
            <ChevronRight className="h-3.5 w-3.5" />
          )}
        </span>
      </button>

      {expanded && (
        <div className="mt-2 space-y-2 min-w-0">
          <div className="min-w-0">
            <p className="mb-1 text-[10px] font-medium uppercase tracking-wide text-ash">
              Arguments
            </p>
            <pre className="max-w-full overflow-x-auto rounded bg-card/60 p-2 text-[11px] font-mono text-body">
              {safeStringify(args)}
            </pre>
          </div>
          {result !== undefined && (
            <div className="min-w-0">
              <p className="mb-1 text-[10px] font-medium uppercase tracking-wide text-ash">
                Result
              </p>
              <pre className="max-w-full overflow-x-auto rounded bg-card/60 p-2 text-[11px] font-mono text-body">
                {safeStringify(result)}
              </pre>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
