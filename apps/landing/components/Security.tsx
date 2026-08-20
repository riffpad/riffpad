"use client";

import { useLanguage } from "./LanguageProvider";
import { ScrollReveal } from "./ScrollReveal";
import type { Messages } from "@/lib/i18n";

// Deterministic PRNG so server and client render the same ciphertext.
function cipherText(seed: number, len = 168): string {
  const chars =
    "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/=";
  let s = "";
  let x = seed;
  for (let i = 0; i < len; i++) {
    x = (x * 1664525 + 1013904223) >>> 0;
    s += chars[x % chars.length];
  }
  return s;
}

const CIPHER_LINES = [
  cipherText(7),
  cipherText(131),
  cipherText(901),
  cipherText(4093),
  cipherText(6151),
];

function LockIcon({ locked }: { locked: boolean }) {
  return (
    <svg
      viewBox="0 0 24 24"
      className="h-3.5 w-3.5"
      fill="none"
      stroke="currentColor"
      strokeWidth={2}
      aria-hidden="true"
    >
      <rect x="5" y="11" width="14" height="9" />
      {locked ? (
        <path d="M8 11V7a4 4 0 0 1 8 0v4" />
      ) : (
        <path d="M8 11V7a4 4 0 0 1 7.5-2" />
      )}
    </svg>
  );
}

function FlowGate({ label, locked }: { label: string; locked: boolean }) {
  return (
    <div className="flex items-center gap-2.5 py-1">
      <svg width="2" height="40" aria-hidden="true">
        <line
          x1="1"
          y1="0"
          x2="1"
          y2="40"
          stroke="var(--accent)"
          strokeWidth={2}
          strokeDasharray="6 6"
          className="flow-dash-line"
        />
      </svg>
      <span className="text-accent">
        <LockIcon locked={locked} />
      </span>
      <span className="text-[10px] font-bold uppercase tracking-[0.08em] text-mute">
        {label}
      </span>
    </div>
  );
}

function PlaintextCard({ label, lines }: { label: string; lines: string[] }) {
  return (
    <div>
      <div className="text-[10px] font-bold uppercase tracking-[0.08em] text-mute">
        {label}
      </div>
      <div className="mt-1.5 border border-hairline bg-surface-muted px-3 py-2.5 text-xs leading-[1.9] text-body">
        {lines.map((line, i) => (
          <div key={line}>
            <span
              className={`mr-2 ${i === 0 ? "text-info" : "text-success"}`}
              aria-hidden="true"
            >
              {i === 0 ? "▸" : "✓"}
            </span>
            {line}
          </div>
        ))}
      </div>
    </div>
  );
}

function CipherFlow({ t }: { t: Messages }) {
  const f = t.security.flow;
  return (
    <div className="min-w-0 py-2">
      <PlaintextCard label={f.daemonLabel} lines={f.plainLines} />

      <FlowGate label={f.encryptLabel} locked />

      {/* relay: nothing but ciphertext, endlessly streaming */}
      <div className="border border-dashed border-hairline-strong px-3 py-2.5">
        <div className="flex items-center justify-between gap-2 text-[10px] font-bold uppercase tracking-[0.08em] text-mute">
          <span>{f.relayLabel}</span>
          <span aria-hidden="true">{f.relayNote}</span>
        </div>
        <div className="mt-2 space-y-1.5 overflow-hidden text-[11px] leading-5 text-mute">
          {CIPHER_LINES.map((line, i) => (
            <div key={i} className="whitespace-nowrap">
              <div
                className="cipher-stream inline-flex w-max"
                style={{
                  animationDuration: `${18 + i * 5}s`,
                  animationDirection: i % 2 ? "reverse" : "normal",
                }}
              >
                <span>{line}</span>
                <span aria-hidden="true">{line}</span>
              </div>
            </div>
          ))}
        </div>
      </div>

      <FlowGate label={f.decryptLabel} locked={false} />

      <PlaintextCard label={f.clientLabel} lines={f.plainLines} />
    </div>
  );
}

export function Security() {
  const { t } = useLanguage();

  return (
    <section
      id="security"
      className="mx-auto max-w-frame scroll-mt-20 px-4 py-12 sm:px-6 sm:py-16 lg:py-32"
    >
      <div className="mx-auto max-w-content">
        <div className="grid items-start gap-10 lg:grid-cols-2">
          <ScrollReveal className="min-w-0">
            <span className="label">{`// ${t.security.label}`}</span>
            <h2 className="mt-6 text-balance text-2xl font-bold leading-[1.25] tracking-[-0.01em] sm:text-3xl">
              {t.security.title}
            </h2>
            <p className="mt-3 max-w-[520px] text-base text-body">
              {t.security.subtitle}
            </p>

            <div className="mt-8 border-t border-hairline">
              {t.security.items.map((item, index) => (
                <div key={index} className="border-b border-hairline py-4">
                  <h3 className="text-base font-bold text-ink">
                    {item.title}
                  </h3>
                  <p className="mt-1.5 max-w-[520px] text-base leading-[1.7] text-body">
                    {item.description}
                  </p>
                </div>
              ))}
            </div>
          </ScrollReveal>

          <ScrollReveal delay={150}>
            <CipherFlow t={t} />
          </ScrollReveal>
        </div>
      </div>
    </section>
  );
}
