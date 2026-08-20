"use client";

import { useLanguage } from "./LanguageProvider";
import { BRAND_ICONS } from "./brand-icons";
import { ScrollReveal } from "./ScrollReveal";

const NODE_W = 160;
const NODE_H = 96;
const NODE_Y = 100;

// Event packets travel daemon → relay → client (path direction = travel).
const EVENT_PATHS = [
  { id: "arch-ev-1", d: "M720 126 L560 126" },
  { id: "arch-ev-2", d: "M400 126 L240 126" },
];
// Command packets travel client → relay → daemon.
const COMMAND_PATHS = [
  { id: "arch-cmd-1", d: "M240 170 L400 170" },
  { id: "arch-cmd-2", d: "M560 170 L720 170" },
];

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
        className="animate-pulse"
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

function Packet({
  pathId,
  color,
  begin,
}: {
  pathId: string;
  color: string;
  begin: string;
}) {
  return (
    <rect
      width={12}
      height={7}
      x={-6}
      y={-3.5}
      fill={color}
      className="arch-packet"
      aria-hidden="true"
    >
      <animateMotion dur="4s" begin={begin} repeatCount="indefinite">
        <mpath href={`#${pathId}`} />
      </animateMotion>
      {/* fade in/out at both ends so the loop feels seamless */}
      <animate
        attributeName="opacity"
        values="0;1;1;0"
        keyTimes="0;0.15;0.85;1"
        dur="4s"
        begin={begin}
        repeatCount="indefinite"
      />
    </rect>
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
      <div className="mx-auto max-w-content">
        <ScrollReveal>
          <span className="label">{`// ${a.label}`}</span>
        </ScrollReveal>
        <ScrollReveal delay={75}>
          <h2 className="mt-6 text-balance text-2xl font-bold leading-[1.25] tracking-[-0.01em] sm:text-3xl">
            {a.title}
          </h2>
        </ScrollReveal>
        <ScrollReveal delay={150}>
          <p className="mt-3 text-base text-body">{a.description}</p>
        </ScrollReveal>

        <ScrollReveal delay={225}>
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
              {EVENT_PATHS.map((p) => (
                <path
                  key={p.id}
                  id={p.id}
                  d={p.d}
                  fill="none"
                  stroke="var(--accent)"
                  strokeOpacity={0.5}
                  strokeWidth={1.5}
                  strokeDasharray="6 6"
                  className="flow-dash-line"
                />
              ))}
              {COMMAND_PATHS.map((p) => (
                <path
                  key={p.id}
                  id={p.id}
                  d={p.d}
                  fill="none"
                  stroke="rgb(var(--info))"
                  strokeOpacity={0.5}
                  strokeWidth={1.5}
                  strokeDasharray="6 6"
                  className="flow-dash-line"
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
                  className="flow-dash-line"
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

              {/* travelling packets */}
              {EVENT_PATHS.map((p) =>
                ["0s", "-2s"].map((begin) => (
                  <Packet
                    key={`${p.id}-${begin}`}
                    pathId={p.id}
                    color="var(--accent)"
                    begin={begin}
                  />
                )),
              )}
              {COMMAND_PATHS.map((p) =>
                ["-1s", "-3s"].map((begin) => (
                  <Packet
                    key={`${p.id}-${begin}`}
                    pathId={p.id}
                    color="rgb(var(--info))"
                    begin={begin}
                  />
                )),
              )}
            </svg>
          </div>
        </div>
        </ScrollReveal>
      </div>
    </section>
  );
}
