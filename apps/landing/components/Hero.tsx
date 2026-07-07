"use client";

import { motion } from "framer-motion";

export function Hero() {
  return (
    <section className="relative overflow-hidden bg-white px-4 pb-20 pt-16 text-center sm:px-6 sm:pt-24 lg:px-8">
      <div className="pointer-events-none absolute inset-0 -z-10">
        <div className="absolute left-1/2 top-0 h-[500px] w-[500px] -translate-x-1/2 rounded-full bg-blue-100/50 blur-3xl" />
      </div>

      <motion.div
        initial={{ opacity: 0, y: 24 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.6, ease: "easeOut" }}
        className="mx-auto max-w-4xl"
      >
        <div className="mb-6 inline-flex items-center gap-2 rounded-full border border-blue-100 bg-blue-50 px-4 py-1.5 text-sm font-medium text-blue-700">
          <span className="h-2 w-2 rounded-full bg-blue-600" />
          AI 时代的代码草稿本
        </div>

        <h1 className="text-4xl font-bold tracking-tight text-gray-950 sm:text-6xl lg:text-7xl">
          一闪而过的灵感
          <br />
          <span className="text-blue-600">→ 能跑的原型</span>
        </h1>

        <p className="mx-auto mt-6 max-w-2xl text-lg text-gray-600 sm:text-xl">
          在地铁上、咖啡馆里、散步时，随口说出想法；Riffpad 的 Agent 会在隔离沙箱里自动写代码、装依赖、跑服务，最终打包成 Cursor / Claude Code 直接可用的项目骨架。
        </p>

        <div className="mt-10 flex flex-col items-center justify-center gap-4 sm:flex-row">
          <a
            href="https://app.riffpad.ai"
            className="rounded-xl bg-blue-600 px-8 py-4 text-base font-medium text-white shadow-lg shadow-blue-600/20 transition hover:bg-blue-700"
          >
            免费开始创作 →
          </a>
          <a
            href="#features"
            className="rounded-xl border border-gray-300 bg-white px-8 py-4 text-base font-medium text-gray-700 transition hover:border-gray-400 hover:bg-gray-50"
          >
            看看怎么工作
          </a>
        </div>

        <p className="mt-4 text-sm text-gray-500">
          无需信用卡 · 500ms 启动沙箱 · 支持手机/平板
        </p>
      </motion.div>

      <motion.div
        initial={{ opacity: 0, y: 40 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.7, delay: 0.15, ease: "easeOut" }}
        className="mx-auto mt-16 max-w-5xl"
      >
        <div className="relative overflow-hidden rounded-2xl border border-gray-200 bg-gray-900 shadow-2xl">
          <div className="flex items-center gap-2 border-b border-gray-800 px-4 py-3">
            <span className="h-3 w-3 rounded-full bg-red-500" />
            <span className="h-3 w-3 rounded-full bg-yellow-500" />
            <span className="h-3 w-3 rounded-full bg-green-500" />
            <span className="ml-3 text-xs text-gray-500">amber-spark-9 — workspace preview</span>
          </div>
          <div className="grid gap-px bg-gray-800 sm:grid-cols-[240px_1fr]">
            <div className="hidden bg-gray-900 p-4 text-left sm:block">
              <p className="mb-2 text-xs font-semibold uppercase tracking-wider text-gray-500">Files</p>
              <ul className="space-y-1.5 text-sm text-gray-400">
                <li>📁 workspace/</li>
                <li className="pl-4">index.html</li>
                <li className="pl-4">main.py</li>
                <li className="pl-4">style.css</li>
                <li>📁 .riffpad/</li>
              </ul>
            </div>
            <div className="min-h-[260px] bg-gray-950 p-4 text-left font-mono text-sm text-gray-300">
              <span className="text-green-400">$ </span>
              帮我做一个倒计时网页，语音输入即可
              <br />
              <span className="text-gray-500">Agent 正在创建文件、安装依赖、启动预览...</span>
              <br />
              <span className="text-blue-400">→ 预览已就绪：</span>{" "}
              <span className="underline">https://preview.riffpad.app/p/abc123</span>
            </div>
          </div>
        </div>
      </motion.div>
    </section>
  );
}
