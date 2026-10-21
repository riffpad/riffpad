"use client";

import { useLanguage } from "./LanguageProvider";
import { BRAND_ICONS } from "./brand-icons";

const NODE_W = 160;
const NODE_H = 96;
const NODE_Y = 100;

// Event lane: daemon → relay → client. Command lane: client → relay → daemon.
const EVENT_PATHS = ["M720 126 L560 126", "M400 126 L240 126"];
const COMMAND_PATHS = ["M240 170 L400 170", "M560 170 L720 170"];

// CLI chips below the daemon (icon-only squares); each links straight up
// to the daemon bottom.
const CLI_CHIPS = [
  { x: 700, link: "M760 196 L722 236" },
  { x: 768, link: "M800 196 L790 236" },
  { x: 836, link: "M840 196 L858 236" },
];

function Node({
  x,
  title,
  caption,
}: {
  x: number;
  title: string;
  caption: string;
}) {
  return (
    <g>
      <rect
        x={x}
        y={NODE_Y}
        width={NODE_W}
        height={NODE_H}
        fill="var(--surface)"
        stroke="var(--hairline-strong)"
      />
      <rect
        x={x + 16}
        y={NODE_Y + 26}
        width={6}
        height={6}
        fill="var(--accent)"
        aria-hidden="true"
      />
      <text
        x={x + 28}
        y={NODE_Y + 33}
        fontSize={12}
        fontWeight={700}
        fill="var(--ink)"
      >
        {title}
      </text>
      <text x={x + 16} y={NODE_Y + 58} fontSize={9.5} fill="var(--mute)">
        {caption}
      </text>
    </g>
  );
}

function FlowLabel({ x, y, children }: { x: number; y: number; children: string }) {
  return (
    <text
      x={x}
      y={y}
      textAnchor="middle"
      fontSize={9}
      fill="var(--mute)"
      className="uppercase"
      style={{ letterSpacing: "0.08em" }}
    >
      {children}
    </text>
  );
}

export function Architecture() {
  const { t } = useLanguage();
  const a = t.architecture;

  return (
    <section
      id="architecture"
      className="mx-auto max-w-frame scroll-mt-20 px-4 py-12 sm:px-6 sm:py-16 lg:py-32"
    >
      <div className="w-full">
        <span className="label">{`// ${a.label}`}</span>
        <h2 className="mt-6 text-balance text-2xl font-bold leading-[1.25] tracking-[-0.01em] sm:text-3xl">
          {a.title}
        </h2>
        <p className="mt-3 text-base text-body">{a.description}</p>

        <div className="mt-12 overflow-x-auto border border-hairline">
          <div className="relative min-w-[680px]">
            <div className="arch-grid absolute inset-0" aria-hidden="true" />
            <svg
              viewBox="0 80 960 220"
              className="relative h-auto w-full"
              role="img"
              aria-label={a.title}
            >
              <title>{a.title}</title>

              {/* flow lanes */}
              {EVENT_PATHS.map((d) => (
                <path
                  key={d}
                  d={d}
                  fill="none"
                  stroke="var(--accent)"
                  strokeOpacity={0.5}
                  strokeWidth={1.5}
                  strokeDasharray="6 6"
                />
              ))}
              {COMMAND_PATHS.map((d) => (
                <path
                  key={d}
                  d={d}
                  fill="none"
                  stroke="rgb(var(--info))"
                  strokeOpacity={0.5}
                  strokeWidth={1.5}
                  strokeDasharray="6 6"
                />
              ))}

              {/* daemon ↔ CLI links */}
              {CLI_CHIPS.map((c) => (
                <path
                  key={c.link}
                  d={c.link}
                  fill="none"
                  stroke="var(--hairline-strong)"
                  strokeWidth={1}
                  strokeDasharray="6 6"
                />
              ))}

              {/* nodes */}
              <Node x={80} title={a.clientTitle} caption={a.clientCaption} />
              <Node x={400} title={a.relayTitle} caption={a.relayCaption} />
              <Node x={720} title={a.hostTitle} caption={a.hostCaption} />

              {/* CLI chips with brand icons */}
              {CLI_CHIPS.map((c, i) => (
                <g key={c.x}>
                  <title>{BRAND_ICONS[i].title}</title>
                  <rect
                    x={c.x}
                    y={236}
                    width={44}
                    height={44}
                    fill="var(--surface)"
                    stroke="var(--hairline)"
                  />
                  <path
                    d={BRAND_ICONS[i].path}
                    transform={`translate(${c.x + 12} 248) scale(0.8333)`}
                    fill="var(--ink)"
                  />
                </g>
              ))}

              {/* more CLIs */}
              <text x={896} y={263} fontSize={10} fill="var(--mute)">
                {a.moreClis}
              </text>

              {/* flow labels */}
              <FlowLabel x={320} y={114}>
                {a.flowEvents}
              </FlowLabel>
              <FlowLabel x={320} y={188}>
                {a.flowCommands}
              </FlowLabel>
              <FlowLabel x={640} y={114}>
                {a.flowEvents}
              </FlowLabel>
              <FlowLabel x={640} y={188}>
                {a.flowCommands}
              </FlowLabel>

            </svg>
          </div>
        </div>
      </div>
    </section>
  );
}
