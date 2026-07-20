"use client";

import { memo, useState } from "react";
import { ChevronDown, ChevronRight, ExternalLink } from "lucide-react";

interface SearchResult {
  index: number;
  title: string;
  url: string;
  snippet: string;
}

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

function parseSearchResults(result: unknown): SearchResult[] | undefined {
  if (!result) return undefined;
  let raw: unknown = result;
  if (typeof result === "string") {
    // The backend returns a JSON array followed by an instruction delimiter.
    // Only parse the JSON array portion.
    const delimiter = result.indexOf("\n\n---\n");
    const jsonPart = delimiter === -1 ? result : result.slice(0, delimiter);
    try {
      raw = JSON.parse(jsonPart);
    } catch {
      return undefined;
    }
  }
  if (!Array.isArray(raw)) return undefined;
  const results: SearchResult[] = [];
  for (const item of raw) {
    if (
      item &&
      typeof item === "object" &&
      typeof (item as Record<string, unknown>).url === "string" &&
      typeof (item as Record<string, unknown>).title === "string"
    ) {
      results.push({
        index: Number((item as Record<string, unknown>).index) || results.length + 1,
        title: String((item as Record<string, unknown>).title),
        url: String((item as Record<string, unknown>).url),
        snippet: String((item as Record<string, unknown>).snippet ?? ""),
      });
    }
  }
  return results.length > 0 ? results : undefined;
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

function ToolExecutionImpl({
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
    ? "bg-accent-yellow"
    : "bg-accent-green";

  const searchResults = toolName === "web_search" ? parseSearchResults(result) : undefined;

  return (
    <div className="min-w-0">
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
        <span className="min-w-0 flex-1 truncate text-xs text-body transition group-hover:text-ink">
          {statusText}
        </span>
        <span className="shrink-0 text-ash opacity-0 transition group-hover:opacity-100">
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
            <pre className="max-w-full whitespace-pre-wrap break-words p-2 text-[11px] font-mono text-body">
              {safeStringify(args)}
            </pre>
          </div>
          {result !== undefined && (
            <div className="min-w-0">
              <p className="mb-1 text-[10px] font-medium uppercase tracking-wide text-ash">
                Result
              </p>
              {searchResults ? (
                <div className="space-y-2">
                  {searchResults.map((r) => (
                    <a
                      key={r.index}
                      href={r.url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="block rounded-md border border-hairline bg-card-doc/50 p-2.5 transition hover:border-primary hover:bg-card-doc"
                    >
                      <div className="flex items-start justify-between gap-2">
                        <span className="text-xs font-semibold text-ink line-clamp-1">
                          [{r.index}] {r.title}
                        </span>
                        <ExternalLink className="mt-0.5 h-3 w-3 shrink-0 text-ash" />
                      </div>
                      <p className="mt-1 text-[11px] leading-relaxed text-body line-clamp-2">
                        {r.snippet}
                      </p>
                      <p className="mt-1 truncate text-[10px] text-mute">
                        {r.url}
                      </p>
                    </a>
                  ))}
                </div>
              ) : (
                <pre className="max-w-full whitespace-pre-wrap break-words p-2 text-[11px] font-mono text-body">
                  {safeStringify(result)}
                </pre>
              )}
            </div>
          )}
        </div>
      )}
    </div>
  );
}

export const ToolExecution = memo(ToolExecutionImpl);
