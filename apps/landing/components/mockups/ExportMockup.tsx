"use client";

import { motion } from "framer-motion";

const CLAUDE_PATH =
  "M21 10.5h3v3h-3v3h-1.5v3H18v-3h-1.5v3H15v-3H9v3H7.5v-3H6v3H4.5v-3H3v-3H0v-3h3v-6h18Zm-15 0h1.5v-3H6Zm10.5 0H18v-3h-1.5z";

const CURSOR_PATH =
  "M11.503.131 1.891 5.678a.84.84 0 0 0-.42.726v11.188c0 .3.162.575.42.724l9.609 5.55a1 1 0 0 0 .998 0l9.61-5.55a.84.84 0 0 0 .42-.724V6.404a.84.84 0 0 0-.42-.726L12.497.131a1.01 1.01 0 0 0-.996 0M2.657 6.338h18.55c.263 0 .43.287.297.515L12.23 22.918c-.062.107-.229.064-.229-.06V12.335a.59.59 0 0 0-.295-.51l-9.11-5.257c-.109-.063-.064-.23.061-.23";

const OPENAI_PATH =
  "M22.282 9.821a5.985 5.985 0 0 0-.516-4.91 6.046 6.046 0 0 0-6.51-2.9A6.065 6.065 0 0 0 4.981 4.18a5.985 5.985 0 0 0-3.998 2.9 6.046 6.046 0 0 0 .743 7.097 5.98 5.98 0 0 0 .51 4.911 6.051 6.051 0 0 0 6.515 2.9A5.985 5.985 0 0 0 13.26 24a6.056 6.056 0 0 0 5.772-4.206 5.99 5.99 0 0 0 3.997-2.9 6.056 6.056 0 0 0-.747-7.073zM13.26 22.43a4.476 4.476 0 0 1-2.876-1.04l.141-.081 4.779-2.758a.795.795 0 0 0 .392-.681v-6.737l2.02 1.168a.071.071 0 0 1 .038.052v5.583a4.504 4.504 0 0 1-4.494 4.494zM3.6 18.304a4.47 4.47 0 0 1-.535-3.014l.142.085 4.783 2.759a.771.771 0 0 0 .78 0l5.843-3.369v2.332a.08.08 0 0 1-.033.062L9.74 19.95a4.5 4.5 0 0 1-6.14-1.646zM2.34 7.896a4.485 4.485 0 0 1 2.366-1.973V11.6a.766.766 0 0 0 .388.676l5.815 3.355-2.02 1.168a.076.076 0 0 1-.071 0l-4.83-2.786A4.504 4.504 0 0 1 2.34 7.896zm16.597 3.855-5.833-3.387L15.119 7.2a.076.076 0 0 1 .071 0l4.83 2.791a4.494 4.494 0 0 1-.676 8.105v-5.678a.79.79 0 0 0-.407-.667zm2.01-3.023-.141-.085-4.774-2.782a.776.776 0 0 0-.785 0L9.409 9.23V6.897a.066.066 0 0 1 .028-.061l4.83-2.787a4.5 4.5 0 0 1 6.68 4.66zm-12.64 4.135-2.02-1.164a.08.08 0 0 1-.038-.057V6.075a4.5 4.5 0 0 1 7.375-3.453l-.142.08L8.704 5.46a.795.795 0 0 0-.393.681zm1.097-2.365 2.602-1.5 2.607 1.5v2.999l-2.597 1.5-2.607-1.5z";

type NodeDef = {
  id: string;
  label: string;
  x: number;
  y: number;
  icon: React.ReactNode;
};

type EdgeDef = {
  id: string;
  d: string;
};

const edges: EdgeDef[] = [
  { id: "riffpad-zip", d: "M 15 50 L 50 50 L 50 25" },
  { id: "riffpad-prompt", d: "M 15 50 L 50 50 L 50 75" },
  { id: "zip-claude", d: "M 50 25 L 85 25 L 85 15" },
  { id: "zip-codex", d: "M 50 25 L 50 50 L 85 50" },
  { id: "prompt-codex", d: "M 50 75 L 50 50 L 85 50" },
  { id: "prompt-cursor", d: "M 50 75 L 85 75 L 85 85" },
];

export function ExportMockup() {
  const nodes: NodeDef[] = [
    {
      id: "riffpad",
      label: "Riffpad",
      x: 15,
      y: 50,
      icon: <RiffpadIcon />,
    },
    {
      id: "zip",
      label: ".zip",
      x: 50,
      y: 25,
      icon: <ZipIcon />,
    },
    {
      id: "prompt",
      label: "Prompt package",
      x: 50,
      y: 75,
      icon: <PackageIcon />,
    },
    {
      id: "claude",
      label: "Claude Code",
      x: 85,
      y: 15,
      icon: (
        <svg width="6" height="6" viewBox="0 0 24 24" role="img" aria-label="Claude Code">
          <path d={CLAUDE_PATH} fill="currentColor" />
        </svg>
      ),
    },
    {
      id: "codex",
      label: "Codex",
      x: 85,
      y: 50,
      icon: <OpenAIIcon />,
    },
    {
      id: "cursor",
      label: "Cursor",
      x: 85,
      y: 85,
      icon: (
        <svg width="6" height="6" viewBox="0 0 24 24" role="img" aria-label="Cursor">
          <path d={CURSOR_PATH} fill="currentColor" />
        </svg>
      ),
    },
  ];

  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      whileInView={{ opacity: 1, y: 0 }}
      viewport={{ once: true }}
      transition={{ duration: 0.6, ease: "easeOut" }}
      className="flex w-full flex-col items-center"
    >
      <div className="relative mx-auto aspect-square w-full max-w-sm sm:max-w-md">
        <svg
          className="absolute inset-0 h-full w-full overflow-visible"
          viewBox="0 0 100 100"
          preserveAspectRatio="xMidYMid meet"
          aria-hidden="true"
        >
          {/* Connection lines */}
          {edges.map((edge) => (
            <path
              key={edge.id}
              d={edge.d}
              fill="none"
              stroke="currentColor"
              strokeWidth="0.7"
              strokeDasharray="3 3"
              strokeLinecap="round"
              strokeLinejoin="round"
              vectorEffect="non-scaling-stroke"
              className="text-muted export-flow-line"
            />
          ))}

          {/* Nodes */}
          {nodes.map((node, i) => (
            <g key={node.id} transform={`translate(${node.x}, ${node.y})`}>
              <motion.g
                initial={{ opacity: 0 }}
                whileInView={{ opacity: 1 }}
                viewport={{ once: true }}
                transition={{
                  duration: 0.4,
                  delay: 0.1 + i * 0.07,
                  ease: "easeOut",
                }}
                className="text-foreground"
              >
                <circle r="4.5" fill="var(--background)" />
                <g transform="translate(-3, -3)">{node.icon}</g>
                <text
                  y="8.5"
                  fontSize="3.2"
                  textAnchor="middle"
                  fill="currentColor"
                  className="font-sans font-semibold"
                >
                  {node.label}
                </text>
              </motion.g>
            </g>
          ))}
        </svg>
      </div>
    </motion.div>
  );
}

function RiffpadIcon() {
  return (
    <svg width="6" height="6" viewBox="0 0 6 6">
      <title>Riffpad</title>
      <image
        href="/rounded_rect_rotated_shapes_only_favicon.png"
        width="6"
        height="6"
        preserveAspectRatio="xMidYMid meet"
        className="riffpad-logo-white"
      />
    </svg>
  );
}

function ZipIcon() {
  return (
    <svg
      width="6"
      height="6"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z" />
      <polyline points="14 2 14 8 20 8" />
      <path d="M8 12v-1" />
      <path d="M8 16v-1" />
      <path d="M8 20v-1" />
    </svg>
  );
}

function PackageIcon() {
  return (
    <svg
      width="6"
      height="6"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <path d="m7.5 4.27 9 5.15" />
      <path d="M21 8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16Z" />
      <path d="m3.3 7 8.7 5 8.7-5" />
      <path d="M12 22V12" />
    </svg>
  );
}

function OpenAIIcon() {
  return (
    <svg width="6" height="6" viewBox="0 0 24 24" role="img" aria-label="OpenAI" fill="currentColor">
      <path d={OPENAI_PATH} />
    </svg>
  );
}
