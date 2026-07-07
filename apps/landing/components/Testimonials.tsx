"use client";

import { motion } from "framer-motion";

const testimonials = [
  {
    quote:
      "以前在地铁上想到一个点子，回家后已经忘了大半。现在用 Riffpad 语音说两句，到公司时原型已经跑通了。",
    name: "林晓峰",
    role: "独立开发者",
    initials: "LX",
  },
  {
    quote:
      "作为技术 PM，我终于能在需求评审前快速验证一个 API 联动方案，直接把可点击原型丢给研发。",
    name: "陈雨桐",
    role: "技术产品经理",
    initials: "CY",
  },
  {
    quote:
      "ChatGPT 里的实验代码终于不再被滚没了。Riffpad 的工作区让我能沉淀、分叉、导出，太舒服了。",
    name: "王浩然",
    role: "AI 应用探索者",
    initials: "WH",
  },
];

export function Testimonials() {
  return (
    <section className="bg-white px-4 py-24 sm:px-6 lg:px-8">
      <div className="mx-auto max-w-6xl">
        <div className="mx-auto max-w-2xl text-center">
          <p className="text-sm font-semibold uppercase tracking-wider text-blue-600">
            用户原声
          </p>
          <h2 className="mt-3 text-3xl font-bold tracking-tight text-gray-950 sm:text-4xl">
            他们不再让灵感溜走
          </h2>
        </div>

        <motion.div
          initial="hidden"
          whileInView="visible"
          viewport={{ once: true, margin: "-100px" }}
          variants={{
            hidden: { opacity: 0 },
            visible: {
              opacity: 1,
              transition: { staggerChildren: 0.1 },
            },
          }}
          className="mt-16 grid gap-6 md:grid-cols-3"
        >
          {testimonials.map((t) => (
            <motion.div
              key={t.name}
              variants={{
                hidden: { opacity: 0, y: 20 },
                visible: { opacity: 1, y: 0 },
              }}
              className="rounded-2xl border border-gray-200 bg-gray-50 p-6"
            >
              <p className="text-base leading-relaxed text-gray-700">“{t.quote}”</p>
              <div className="mt-6 flex items-center gap-3">
                <span className="flex h-10 w-10 items-center justify-center rounded-full bg-gray-200 text-sm font-semibold text-gray-700">
                  {t.initials}
                </span>
                <div>
                  <p className="text-sm font-semibold text-gray-950">{t.name}</p>
                  <p className="text-xs text-gray-500">{t.role}</p>
                </div>
              </div>
            </motion.div>
          ))}
        </motion.div>
      </div>
    </section>
  );
}
