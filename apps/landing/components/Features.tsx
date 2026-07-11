"use client";

import { motion } from "framer-motion";
import { useLanguage } from "./LanguageProvider";

export function Features() {
  const { t, lang } = useLanguage();
  const blocks = t.features.blocks;

  const mockups = [
    <AgentWorkspaceMockup key="m1" />,
    <SkillsMockup key="m2" />,
    <MobileVoiceMockup key="m3" />,
    <ExportMockup key="m4" />,
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

function AgentWorkspaceMockup() {
  return (
    <motion.div
      initial={{ opacity: 0, scale: 0.96 }}
      whileInView={{ opacity: 1, scale: 1 }}
      viewport={{ once: true }}
      transition={{ duration: 0.6 }}
      className="overflow-hidden rounded-xl border border-hairline bg-surface shadow-xl"
    >
      <div className="flex h-64 sm:h-80">
        {/* Chat */}
        <div className="flex w-1/3 flex-col gap-2 border-r border-hairline bg-surface p-3">
          <div className="mb-2 text-[10px] font-bold uppercase tracking-wider text-muted">Chat</div>
          <div className="ml-auto max-w-[90%] rounded-lg bg-foreground px-3 py-2 text-xs text-background">
            Build a countdown page
          </div>
          <div className="max-w-[90%] rounded-lg border border-hairline bg-surface-doc px-3 py-2 text-xs text-body">
            Done. Files created.
          </div>
          <div className="ml-auto max-w-[90%] rounded-lg bg-foreground px-3 py-2 text-xs text-background">
            Show preview
          </div>
        </div>

        {/* Editor */}
        <div className="flex w-1/3 flex-col border-r border-hairline bg-surface-doc p-4">
          <div className="mb-2 text-[10px] font-bold uppercase tracking-wider text-muted">Editor</div>
          <div className="font-mono text-xs leading-relaxed text-body">
            <span className="text-accent">const</span> target =
            <span className="text-accent-green"> new</span> Date(
            <span className="text-accent-blue">&quot;2026-08-01&quot;</span>);
          </div>
          <div className="mt-4 rounded-md bg-surface p-2">
            <div className="h-20 rounded bg-accent/10" />
          </div>
        </div>

        {/* File tree */}
        <div className="w-1/3 bg-surface p-4">
          <div className="mb-3 text-[10px] font-bold uppercase tracking-wider text-muted">Files</div>
          <div className="space-y-2">
            <div className="flex items-center gap-2 text-xs text-body">
              <span className="text-accent-blue">◈</span> index.html
            </div>
            <div className="flex items-center gap-2 text-xs text-body">
              <span className="text-accent-purple">◈</span> style.css
            </div>
            <div className="flex items-center gap-2 text-xs text-body">
              <span className="text-accent">◈</span> main.js
            </div>
          </div>
        </div>
      </div>
    </motion.div>
  );
}

function SkillsMockup() {
  return (
    <motion.div
      initial={{ opacity: 0, scale: 0.96 }}
      whileInView={{ opacity: 1, scale: 1 }}
      viewport={{ once: true }}
      transition={{ duration: 0.6 }}
      className="grid gap-4 sm:grid-cols-5"
    >
      <div className="overflow-hidden rounded-xl border border-hairline bg-surface p-4 shadow-xl sm:col-span-3">
        <div className="mb-3 text-[10px] font-bold uppercase tracking-wider text-muted">Chat</div>
        <div className="ml-auto max-w-[85%] rounded-lg bg-foreground px-3 py-2 text-xs text-background">
          Install the GitHub skill
        </div>
        <div className="mt-2 max-w-[90%] rounded-lg border border-hairline bg-surface-doc px-3 py-2 text-xs text-body">
          Installing...
        </div>
        <div className="mt-2 rounded-md bg-surface-dark p-2 font-mono text-[10px] text-on-dark">
          npx skills add github --source https://...
        </div>
        <div className="mt-2 max-w-[90%] rounded-lg border border-hairline bg-surface-doc px-3 py-2 text-xs text-body">
          GitHub skill is ready.
        </div>
      </div>

      <div className="overflow-hidden rounded-xl border border-hairline bg-surface p-4 shadow-xl sm:col-span-2">
        <div className="mb-3 text-[10px] font-bold uppercase tracking-wider text-muted">Skills</div>
        <div className="space-y-3">
          {["GitHub", "Figma", "Stripe"].map((skill) => (
            <div key={skill} className="flex items-center justify-between">
              <span className="text-sm text-body">{skill}</span>
              <div className="h-5 w-9 rounded-full bg-accent p-0.5">
                <div className="ml-auto h-4 w-4 rounded-full bg-on-accent" />
              </div>
            </div>
          ))}
        </div>
      </div>
    </motion.div>
  );
}

function MobileVoiceMockup() {
  return (
    <motion.div
      initial={{ opacity: 0, scale: 0.96 }}
      whileInView={{ opacity: 1, scale: 1 }}
      viewport={{ once: true }}
      transition={{ duration: 0.6 }}
      className="flex justify-center"
    >
      <div className="relative w-56 rounded-[2.5rem] border-[6px] border-hairline bg-background p-3 shadow-2xl sm:w-64">
        <div className="absolute left-1/2 top-0 h-6 w-24 -translate-x-1/2 rounded-b-xl bg-hairline" />
        <div className="flex h-96 flex-col rounded-2xl bg-surface p-4">
          <div className="text-center text-xs font-semibold text-muted">Riffpad</div>

          <div className="mt-6 flex-1 space-y-3">
            <div className="ml-auto max-w-[85%] rounded-2xl rounded-br-sm bg-foreground px-3 py-2 text-xs text-background">
              Build a habit tracker app
            </div>
            <div className="max-w-[90%] rounded-2xl rounded-bl-sm border border-hairline bg-surface-doc px-3 py-2 text-xs text-body">
              Got it. Creating the workspace now.
            </div>
            <div className="rounded-xl bg-surface-soft p-2">
              <div className="h-24 rounded-lg bg-accent/10" />
            </div>
          </div>

          <div className="flex flex-col items-center gap-2 py-4">
            <div className="flex h-2 w-32 items-center justify-between">
              {[...Array(5)].map((_, i) => (
                <div
                  key={i}
                  className="w-1 rounded-full bg-accent"
                  style={{ height: `${16 + Math.random() * 24}px` }}
                />
              ))}
            </div>
            <div className="flex h-12 w-12 items-center justify-center rounded-full bg-accent shadow-lg">
              <svg className="h-5 w-5 text-on-accent" fill="none" stroke="currentColor" strokeWidth="2" viewBox="0 0 24 24">
                <path d="M12 1a3 3 0 0 0-3 3v8a3 3 0 0 0 6 0V4a3 3 0 0 0-3-3z" />
                <path d="M19 10v2a7 7 0 0 1-14 0v-2" />
                <line x1="12" y1="19" x2="12" y2="23" />
              </svg>
            </div>
          </div>
        </div>
      </div>
    </motion.div>
  );
}

function ExportMockup() {
  return (
    <motion.div
      initial={{ opacity: 0, scale: 0.96 }}
      whileInView={{ opacity: 1, scale: 1 }}
      viewport={{ once: true }}
      transition={{ duration: 0.6 }}
      className="rounded-xl border border-hairline bg-surface p-6 shadow-xl"
    >
      <div className="mb-4 text-[10px] font-bold uppercase tracking-wider text-muted">Export</div>
      <div className="grid gap-4 sm:grid-cols-2">
        <div className="rounded-lg border border-hairline bg-surface-doc p-5 transition hover:border-ash">
          <div className="flex items-center gap-2">
            <div className="flex h-8 w-8 items-center justify-center rounded-md bg-accent text-sm font-bold text-on-accent">P</div>
            <div className="text-base font-bold text-foreground">Prompt package</div>
          </div>
          <p className="mt-2 text-sm text-body">Context + files + README for Cursor / Claude Code.</p>
        </div>

        <div className="rounded-lg border border-hairline bg-surface-doc p-5 transition hover:border-ash">
          <div className="flex items-center gap-2">
            <div className="flex h-8 w-8 items-center justify-center rounded-md bg-accent text-sm font-bold text-on-accent">Z</div>
            <div className="text-base font-bold text-foreground">.zip skeleton</div>
          </div>
          <p className="mt-2 text-sm text-body">Clean project archive ready for any IDE.</p>
        </div>
      </div>
    </motion.div>
  );
}
