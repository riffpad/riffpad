"use client";

import { useState } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { ChevronDown } from "./Icons";

const faqs = [
  {
    q: "Riffpad 和 Cursor / Windsurf 有什么区别？",
    a: "Cursor 是重型 IDE，适合在电脑上写生产级代码；Riffpad 是最上游的灵感孵化器，专为手机、平板和碎片化场景设计，让你在 5 分钟内把一句话验证成可运行的原型，再桥接到 Cursor / Claude Code 继续开发。",
  },
  {
    q: "手机上的代码体验会不会很难受？",
    a: "你不会在手机上写大量代码。Riffpad 的核心交互是「语音/文字描述意图 → Agent 自动执行」。文件树、预览、批量确认都针对移动端做了重新设计，必要时还能在 iPad 上获得接近桌面的体验。",
  },
  {
    q: "沙箱里的代码安全吗？",
    a: "每个工作区运行在独立的 Firecracker microVM / gVisor 隔离环境中，Agent 默认只能操作 /sandbox/workspace/ 目录，网络、CPU、内存都有配额限制。高危命令会被默认拦截。",
  },
  {
    q: "可以导出到哪些下游工具？",
    a: "目前支持导出为 Zip 压缩包，以及一键推送到临时 GitHub 仓库。导出包包含重构后的代码骨架、README、技术路径说明和 AI 提示词上下文，方便 Cursor、Claude Code、VS Code 等直接接管。",
  },
  {
    q: "免费计划够用吗？",
    a: "Free 计划每月 10 个 Spark，足以验证大部分临时想法。如果你每天都有灵感需要长期驻留、更大算力或团队协作，再升级到 Pro / Team。",
  },
];

export const faqSchema = {
  "@context": "https://schema.org",
  "@type": "FAQPage",
  mainEntity: faqs.map((f) => ({
    "@type": "Question",
    name: f.q,
    acceptedAnswer: {
      "@type": "Answer",
      text: f.a,
    },
  })),
};

export function FAQ() {
  const [openIndex, setOpenIndex] = useState<number | null>(0);

  return (
    <section id="faq" className="bg-gray-50 px-4 py-24 sm:px-6 lg:px-8">
      <div className="mx-auto max-w-3xl">
        <div className="text-center">
          <p className="text-sm font-semibold uppercase tracking-wider text-blue-600">
            FAQ
          </p>
          <h2 className="mt-3 text-3xl font-bold tracking-tight text-gray-950 sm:text-4xl">
            常见问题
          </h2>
        </div>

        <div className="mt-12 space-y-4">
          {faqs.map((faq, i) => {
            const isOpen = openIndex === i;
            return (
              <div
                key={faq.q}
                className="overflow-hidden rounded-2xl border border-gray-200 bg-white"
              >
                <button
                  onClick={() => setOpenIndex(isOpen ? null : i)}
                  className="flex w-full items-center justify-between px-6 py-5 text-left"
                  aria-expanded={isOpen}
                >
                  <span className="text-base font-medium text-gray-950">{faq.q}</span>
                  <motion.span
                    animate={{ rotate: isOpen ? 180 : 0 }}
                    transition={{ duration: 0.2 }}
                  >
                    <ChevronDown className="h-5 w-5 text-gray-500" />
                  </motion.span>
                </button>
                <AnimatePresence initial={false}>
                  {isOpen && (
                    <motion.div
                      initial={{ height: 0, opacity: 0 }}
                      animate={{ height: "auto", opacity: 1 }}
                      exit={{ height: 0, opacity: 0 }}
                      transition={{ duration: 0.2 }}
                    >
                      <p className="px-6 pb-5 text-sm leading-relaxed text-gray-600">
                        {faq.a}
                      </p>
                    </motion.div>
                  )}
                </AnimatePresence>
              </div>
            );
          })}
        </div>
      </div>
    </section>
  );
}
