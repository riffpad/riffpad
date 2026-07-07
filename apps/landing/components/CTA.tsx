"use client";

import { motion } from "framer-motion";

export function CTA() {
  return (
    <section className="bg-gray-950 px-4 py-24 text-center sm:px-6 lg:px-8">
      <motion.div
        initial={{ opacity: 0, scale: 0.98 }}
        whileInView={{ opacity: 1, scale: 1 }}
        viewport={{ once: true }}
        transition={{ duration: 0.5 }}
        className="mx-auto max-w-3xl"
      >
        <h2 className="text-3xl font-bold tracking-tight text-white sm:text-4xl">
          今天就不让一个灵感溜走
        </h2>
        <p className="mt-4 text-lg text-gray-400">
          免费开始，500ms 启动你的第一个沙箱。灵感来的时候，Riffpad 已经准备好了。
        </p>
        <div className="mt-10 flex flex-col items-center justify-center gap-4 sm:flex-row">
          <a
            href="https://app.riffpad.ai"
            className="rounded-xl bg-blue-600 px-8 py-4 text-base font-medium text-white transition hover:bg-blue-700"
          >
            免费开始创作 →
          </a>
          <a
            href="/docs"
            className="rounded-xl border border-gray-700 px-8 py-4 text-base font-medium text-white transition hover:border-gray-500"
          >
            阅读文档
          </a>
        </div>
        <p className="mt-4 text-sm text-gray-500">
          无需信用卡 · 随时可降级回 Free
        </p>
      </motion.div>
    </section>
  );
}
