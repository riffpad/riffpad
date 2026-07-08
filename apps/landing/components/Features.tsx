"use client";

import { motion } from "framer-motion";
import { useLanguage } from "./LanguageProvider";
import { Check, X } from "./Icons";

export function Features() {
  const { t, lang } = useLanguage();
  const blocks = t.features.blocks;

  const mockups = [
    <IdeasToFilesMockup key="m1" />,
    <InstantOnMockup key="m2" />,
    <ComparisonMockup key="m3" />,
    <EcosystemMockup key="m4" />,
  ];

  return (
    <section id="features" className="bg-background px-4 py-24 sm:px-6 lg:px-8">
      <div className="mx-auto max-w-6xl">
        <div className="mx-auto max-w-2xl text-center">
          <p className="text-xs font-bold uppercase tracking-wider text-body">
            {t.features.eyebrow}
          </p>
          <h2 className="mt-2 text-3xl font-bold text-foreground sm:text-4xl">
            {t.features.title}
          </h2>
          <p className="mt-3 text-body">{t.features.subtitle}</p>
        </div>

        <div className="mt-20 space-y-32">
          {blocks.map((block, i) => (
            <motion.div
              key={`${lang}-${i}`}
              initial={{ opacity: 0, y: 40 }}
              whileInView={{ opacity: 1, y: 0 }}
              viewport={{ once: true, margin: "-100px" }}
              transition={{ duration: 0.7, ease: "easeOut" }}
              className="grid items-center gap-12 lg:grid-cols-2"
            >
              <div className={i % 2 === 1 ? "lg:order-2" : ""}>
                <span className="text-xs font-bold uppercase tracking-wider text-accent">
                  0{i + 1}
                </span>
                <h3 className="mt-2 text-2xl font-bold text-foreground sm:text-3xl">
                  {block.title}
                </h3>
                <p className="mt-4 text-lg leading-relaxed text-body">{block.description}</p>
              </div>
              <div className={i % 2 === 1 ? "lg:order-1" : ""}>
                {mockups[i]}
              </div>
            </motion.div>
          ))}
        </div>
      </div>
    </section>
  );
}

function IdeasToFilesMockup() {
  return (
    <motion.div
      initial={{ opacity: 0, scale: 0.96 }}
      whileInView={{ opacity: 1, scale: 1 }}
      viewport={{ once: true }}
      transition={{ duration: 0.6 }}
      className="relative overflow-hidden rounded-xl border border-hairline bg-surface p-1 shadow-xl"
    >
      <div className="grid gap-1 sm:grid-cols-2">
        <div className="rounded-lg bg-surface-soft/50 p-5">
          <div className="mb-4 flex items-center gap-2">
            <span className="h-2 w-2 rounded-full bg-accent-red" />
            <span className="h-2 w-2 rounded-full bg-accent" />
            <span className="h-2 w-2 rounded-full bg-accent-green" />
            <span className="ml-2 text-xs font-semibold text-muted">Chat Thread #47</span>
          </div>
          <div className="space-y-3">
            <div className="ml-auto max-w-[85%] rounded-2xl rounded-br-sm bg-hairline px-3 py-2 text-xs text-foreground opacity-60">
              Build me a countdown...
            </div>
            <div className="max-w-[90%] rounded-2xl rounded-bl-sm border border-hairline bg-surface px-3 py-2 text-xs text-body">
              Sure, here is the code...
              <span className="mt-1 block font-mono text-[10px] text-muted">const target = new Date(...)</span>
            </div>
            <div className="ml-auto max-w-[85%] rounded-2xl rounded-br-sm bg-hairline px-3 py-2 text-xs text-foreground opacity-40">
              Can you change it to red?
            </div>
            <div className="max-w-[90%] rounded-2xl rounded-bl-sm border border-hairline bg-surface px-3 py-2 text-xs text-body opacity-50">
              Here is the updated code...
            </div>
            <div className="text-center text-[10px] text-muted">... 37 older messages ...</div>
          </div>
        </div>

        <div className="relative rounded-lg bg-surface p-5">
          <div className="mb-4 flex items-center justify-between">
            <span className="text-xs font-semibold text-muted">Riffpad Workspace</span>
            <span className="rounded-full bg-accent-green/10 px-2 py-0.5 text-[10px] font-bold text-accent-green">Live Preview</span>
          </div>
          <div className="font-mono text-sm text-body">
            <div className="flex items-center gap-1.5">
              <span>📁</span>
              <span className="font-semibold text-foreground">launch-countdown</span>
            </div>
            <div className="mt-1 space-y-1 border-l-2 border-hairline-soft pl-4">
              <div className="flex items-center gap-1.5">
                <span className="text-accent-blue">◈</span> index.html
              </div>
              <div className="flex items-center gap-1.5">
                <span className="text-accent-purple">◈</span> style.css
              </div>
              <div className="flex items-center gap-1.5">
                <span className="text-accent">◈</span> main.js
              </div>
            </div>
          </div>
          <motion.div
            initial={{ opacity: 0, y: 10 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true }}
            transition={{ delay: 0.4 }}
            className="mt-4 overflow-hidden rounded-md bg-surface-dark p-3 font-mono text-xs text-on-dark"
          >
            <span className="text-accent">const</span> target =
            <span className="text-accent-green"> new</span> Date(<span className="text-accent-blue">&quot;2026-08-01&quot;</span>);
          </motion.div>
        </div>
      </div>

      <div className="absolute left-1/2 top-1/2 hidden -translate-x-1/2 -translate-y-1/2 sm:block">
        <div className="flex h-10 w-10 items-center justify-center rounded-full bg-accent shadow-lg">
          <svg className="h-5 w-5 text-on-accent" fill="none" stroke="currentColor" strokeWidth="2" viewBox="0 0 24 24">
            <path d="M5 12h14M12 5l7 7-7 7" />
          </svg>
        </div>
      </div>
    </motion.div>
  );
}

function InstantOnMockup() {
  return (
    <motion.div
      initial={{ opacity: 0, scale: 0.96 }}
      whileInView={{ opacity: 1, scale: 1 }}
      viewport={{ once: true }}
      transition={{ duration: 0.6 }}
      className="relative rounded-xl border border-hairline bg-surface p-8 shadow-xl"
    >
      <div className="flex flex-col items-end justify-center gap-6 sm:flex-row sm:items-center">
        <DeviceFrame label="Phone" delay={0}>
          <div className="space-y-1.5">
            <div className="h-1 w-10 rounded bg-hairline" />
            <div className="h-1 w-14 rounded bg-hairline-soft" />
            <div className="mt-2 h-8 rounded bg-accent/20" />
          </div>
        </DeviceFrame>
        <DeviceFrame label="Tablet" delay={0.15}>
          <div className="grid grid-cols-2 gap-1.5">
            <div className="h-7 rounded bg-hairline-soft" />
            <div className="h-7 rounded bg-accent/20" />
            <div className="col-span-2 h-1.5 rounded bg-hairline" />
          </div>
        </DeviceFrame>
        <DeviceFrame label="Laptop" delay={0.3}>
          <div className="space-y-1.5">
            <div className="h-1 w-full rounded bg-hairline" />
            <div className="h-1 w-4/5 rounded bg-hairline-soft" />
            <div className="mt-1 h-6 rounded bg-accent/20" />
          </div>
        </DeviceFrame>
      </div>

      <motion.div
        initial={{ opacity: 0, y: 10 }}
        whileInView={{ opacity: 1, y: 0 }}
        viewport={{ once: true }}
        transition={{ delay: 0.5 }}
        className="mt-8 flex justify-center"
      >
        <button className="relative rounded-full bg-accent px-8 py-3 text-sm font-bold text-on-accent shadow-lg">
          <span className="absolute inset-0 rounded-full bg-accent opacity-50 blur-md animate-pulse" />
          <span className="relative">One-Click Launch</span>
        </button>
      </motion.div>
    </motion.div>
  );
}

function DeviceFrame({
  label,
  delay,
  children,
}: {
  label: string;
  delay: number;
  children: React.ReactNode;
}) {
  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      whileInView={{ opacity: 1, y: 0 }}
      viewport={{ once: true }}
      transition={{ delay, duration: 0.5 }}
      className="flex w-full flex-col items-center gap-2 sm:w-auto"
    >
      <div className="flex h-20 w-full items-center justify-center rounded-lg border border-hairline bg-surface-doc px-4 shadow-sm sm:w-28">
        {children}
      </div>
      <span className="text-[10px] font-bold uppercase tracking-wider text-muted">{label}</span>
    </motion.div>
  );
}

function ComparisonMockup() {
  const rows: Array<{ feature: string; riffpad: boolean | "partial"; claude: boolean | "partial" }> = [
    { feature: "Local setup", riffpad: true, claude: false },
    { feature: "Cloud sync across devices", riffpad: true, claude: false },
    { feature: "Instant runnable preview", riffpad: true, claude: "partial" },
    { feature: "No dependency management", riffpad: true, claude: false },
    { feature: "One-click downstream export", riffpad: true, claude: false },
  ];

  return (
    <motion.div
      initial={{ opacity: 0, scale: 0.96 }}
      whileInView={{ opacity: 1, scale: 1 }}
      viewport={{ once: true }}
      transition={{ duration: 0.6 }}
      className="overflow-hidden rounded-xl border border-gray-700 bg-surface-dark shadow-xl"
    >
      <table className="w-full text-left text-sm">
        <thead className="border-b border-gray-700 bg-gray-900/50">
          <tr>
            <th className="px-5 py-4 text-xs font-bold uppercase tracking-wider text-muted">Feature</th>
            <th className="px-5 py-4 text-center text-xs font-bold uppercase tracking-wider text-accent">Riffpad</th>
            <th className="px-5 py-4 text-center text-xs font-bold uppercase tracking-wider text-muted">Claude Code / Codex</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-gray-700"
        >
          {rows.map((row, i) => (
            <motion.tr
              key={row.feature}
              initial={{ opacity: 0, x: -10 }}
              whileInView={{ opacity: 1, x: 0 }}
              viewport={{ once: true }}
              transition={{ delay: i * 0.08 }}
            >
              <td className="px-5 py-4 font-medium text-on-dark">{row.feature}</td>
              <td className="px-5 py-4 text-center">
                <Status value={row.riffpad} highlight />
              </td>
              <td className="px-5 py-4 text-center">
                <Status value={row.claude} />
              </td>
            </motion.tr>
          ))}
        </tbody>
      </table>
    </motion.div>
  );
}

function Status({ value, highlight }: { value: boolean | "partial"; highlight?: boolean }) {
  if (value === true) {
    return (
      <span className={`inline-flex h-6 w-6 items-center justify-center rounded-full ${highlight ? "bg-accent text-on-accent" : "bg-gray-700 text-gray-300"}`}>
        <Check className="h-3.5 w-3.5" />
      </span>
    );
  }
  if (value === "partial") {
    return (
      <span className="inline-flex h-6 w-6 items-center justify-center rounded-full bg-gray-700 text-gray-300 text-[10px] font-bold">
        ~
      </span>
    );
  }
  return (
    <span className="inline-flex h-6 w-6 items-center justify-center rounded-full bg-gray-700 text-gray-300">
      <X className="h-3.5 w-3.5" />
    </span>
  );
}

function EcosystemMockup() {
  const downstream = [
    { label: "Cursor", icon: "🖱️" },
    { label: "Claude Code", icon: "🧠" },
    { label: "MCP Server", icon: "🔌" },
    { label: ".zip", icon: "🗜️" },
  ];

  return (
    <motion.div
      initial={{ opacity: 0, scale: 0.96 }}
      whileInView={{ opacity: 1, scale: 1 }}
      viewport={{ once: true }}
      transition={{ duration: 0.6 }}
      className="relative rounded-xl border border-hairline bg-surface p-8 shadow-xl"
    >
      <div className="relative flex flex-col items-center justify-center gap-8 sm:flex-row">
        <motion.div
          initial={{ scale: 0 }}
          whileInView={{ scale: 1 }}
          viewport={{ once: true }}
          transition={{ type: "spring", stiffness: 200, damping: 15 }}
          className="relative z-10 flex h-24 w-24 flex-col items-center justify-center rounded-2xl bg-accent shadow-lg"
        >
          <span className="text-2xl font-black text-on-accent">RP</span>
          <span className="text-[10px] font-bold text-on-accent/80">Riffpad</span>
        </motion.div>

        <div className="grid grid-cols-2 gap-4 sm:gap-6">
          {downstream.map((item, i) => (
            <motion.div
              key={item.label}
              initial={{ opacity: 0, scale: 0.8 }}
              whileInView={{ opacity: 1, scale: 1 }}
              viewport={{ once: true }}
              transition={{ delay: 0.2 + i * 0.1, type: "spring" }}
              className="relative flex flex-col items-center gap-2"
            >
              <div className="flex h-14 w-14 items-center justify-center rounded-xl border border-hairline bg-surface-doc text-xl shadow-sm">
                {item.icon}
              </div>
              <span className="text-xs font-semibold text-body">{item.label}</span>
              <svg
                className="absolute -left-8 top-1/2 hidden h-px w-8 -translate-y-1/2 sm:block"
                style={{ strokeDasharray: 4 }}
              >
                <line x1="0" y1="0" x2="100%" y2="0" stroke="var(--hairline)" strokeWidth="1.5" />
              </svg>
            </motion.div>
          ))}
        </div>
      </div>

      <div className="mt-6 text-center text-xs font-semibold uppercase tracking-wider text-muted">
        Upstream incubator → downstream agents
      </div>
    </motion.div>
  );
}
