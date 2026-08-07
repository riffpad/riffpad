import { useState } from "react";
import DotMatrix from "./DotMatrix";

export interface ToolLine {
  key: string;
  glyph: string;
  status: "run" | "done" | "fail";
  detail: string;
}

export default function ToolLog({ line }: { line: ToolLine }) {
  const [open, setOpen] = useState(false);
  const expandable = line.detail.length > 0;
  return (
    <div className="tool-log-wrap">
      <div
        className={"tool-log " + line.status + (expandable ? " expandable" : "")}
        onClick={() => expandable && setOpen(!open)}
        role={expandable ? "button" : undefined}
        tabIndex={expandable ? 0 : undefined}
        onKeyDown={(e) => {
          if (expandable && (e.key === "Enter" || e.key === " ")) {
            e.preventDefault();
            setOpen(!open);
          }
        }}
      >
        <span className="tool-glyph">
          {line.status === "run" ? (
            <DotMatrix />
          ) : line.status === "done" ? (
            <span className="tool-check" />
          ) : (
            "✗"
          )}
        </span>
        <span className="tool-text truncate">{line.glyph}</span>
        {expandable && <span className="tool-chevron">{open ? "▾" : "▸"}</span>}
      </div>
      {open && <pre className="tool-log-detail">{line.detail}</pre>}
    </div>
  );
}
