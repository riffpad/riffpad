// Pure event -> tool-row mapping for the session timeline. Kept out of the
// view so it can be reasoned about (and tested) without rendering (#300).

import type { useI18n } from "../lib/i18n";
import type { RiffpadEvent } from "../lib/types";
import type { ToolLine } from "./ToolLog";

export function toolLineFromEvent(ev: RiffpadEvent, t: ReturnType<typeof useI18n>["t"]): ToolLine | null {
  const p = ev.payload || {};
  if (ev.type === "tool_call") {
    const tool = String(p.tool || "");
    const status = String(p.status || "started");
    const args = p.args as Record<string, unknown> | undefined;
    const path = args && typeof args.path === "string" ? String(args.path) : "";
    const key = path ? "path:" + path : "tool:" + tool + ":" + String(p.summary || "");
    const glyph = path ? `${tool} ${path}` : `${tool} ${String(p.summary || "")}`.trim();
    const st: ToolLine["status"] = status === "completed" ? "done" : status === "failed" ? "fail" : "run";
    let detail = "";
    if (args) {
      if (typeof args.content === "string") {
        detail = t("content_preview") + "\n" + String(args.content).slice(0, 800) + (String(args.content).length > 800 ? "\n" + t("truncated") : "");
      } else {
        detail = JSON.stringify(args, null, 2).slice(0, 1200);
      }
    }
    return { key, glyph, status: st, detail };
  }
  if (ev.type === "file_change") {
    const path = String(p.path || "");
    return { key: "path:" + path, glyph: "FileChange " + path, status: "done", detail: String(p.summary || path) };
  }
  if (ev.type === "command") {
    const cmd = String(p.command || "");
    const exit = p.exitCode as number | undefined;
    return {
      key: "cmd:" + cmd,
      glyph: "$ " + cmd,
      status: exit === undefined ? "run" : exit === 0 ? "done" : "fail",
      detail: [String(p.output || ""), exit !== undefined ? "exit code: " + exit : ""].filter(Boolean).join("\n"),
    };
  }
  return null;
}

export function mergeTool(ex: ToolLine, line: ToolLine): ToolLine {
  const status = ex.status === "done" || ex.status === "fail" ? ex.status : line.status;
  return {
    key: ex.key,
    glyph: ex.glyph || line.glyph,
    status,
    detail: line.detail || ex.detail || "",
  };
}
