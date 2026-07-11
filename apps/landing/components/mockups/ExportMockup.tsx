"use client";

import { motion } from "framer-motion";

export function ExportMockup() {
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
