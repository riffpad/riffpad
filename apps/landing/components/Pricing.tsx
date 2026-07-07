"use client";

import { motion } from "framer-motion";
import { Check } from "./Icons";

const plans = [
  {
    name: "Free",
    price: "¥0",
    period: "永久免费",
    description: "适合个人开发者随手验证灵感。",
    features: [
      "每月 10 个 Spark",
      "单沙箱 512MB 内存 / 1GB 磁盘",
      "Node.js + Python 基础镜像",
      "48h 自动休眠",
      "社区支持",
    ],
    cta: "免费开始",
    href: "https://app.riffpad.ai",
    highlighted: false,
  },
  {
    name: "Pro",
    price: "¥79",
    period: "/ 月",
    description: "适合重度创作者，让原型跑得更快、交付更顺。",
    features: [
      "无限 Spark",
      "单沙箱 2GB 内存 / 8GB 磁盘",
      "优先预热池，P95 \u003c 500ms",
      "长驻项目（取消 48h 休眠）",
      "一键导出 Zip / GitHub",
      "语音输入 + Whisper 回退",
      "邮件支持",
    ],
    cta: "升级到 Pro",
    href: "https://app.riffpad.ai/billing",
    highlighted: true,
  },
  {
    name: "Team",
    price: "¥249",
    period: "/ 人 / 月",
    description: "适合小团队共享沙箱、沉淀技术方案。",
    features: [
      "Pro 全部功能",
      "团队工作区共享",
      "SSO / SAML（即将上线）",
      "私有模型接入",
      "审计日志与配额管理",
      "优先技术支持",
    ],
    cta: "联系销售",
    href: "mailto:hello@riffpad.ai",
    highlighted: false,
  },
];

export function Pricing() {
  return (
    <section id="pricing" className="bg-white px-4 py-24 sm:px-6 lg:px-8">
      <div className="mx-auto max-w-6xl">
        <div className="mx-auto max-w-2xl text-center">
          <p className="text-sm font-semibold uppercase tracking-wider text-blue-600">
            简单定价
          </p>
          <h2 className="mt-3 text-3xl font-bold tracking-tight text-gray-950 sm:text-4xl">
            先用起来，再决定是否付费
          </h2>
          <p className="mt-4 text-gray-600">
            Free 计划足以验证大多数灵感；当你需要更多算力和长驻项目时，Pro 会接住你。
          </p>
        </div>

        <div className="mt-16 grid gap-6 lg:grid-cols-3">
          {plans.map((plan) => (
            <motion.div
              key={plan.name}
              whileHover={{ y: -4 }}
              transition={{ duration: 0.2 }}
              className={`relative rounded-2xl p-6 sm:p-8 ${
                plan.highlighted
                  ? "border-2 border-blue-600 bg-blue-50/50 shadow-xl shadow-blue-900/5"
                  : "border border-gray-200 bg-gray-50"
              }`}
            >
              {plan.highlighted && (
                <span className="absolute -top-3 left-1/2 -translate-x-1/2 rounded-full bg-blue-600 px-3 py-1 text-xs font-medium text-white">
                  最受欢迎
                </span>
              )}
              <h3 className="text-lg font-semibold text-gray-950">{plan.name}</h3>
              <p className="mt-1 text-sm text-gray-600">{plan.description}</p>
              <div className="mt-4 flex items-baseline">
                <span className="text-4xl font-bold tracking-tight text-gray-950">
                  {plan.price}
                </span>
                <span className="ml-1 text-sm text-gray-500">{plan.period}</span>
              </div>

              <a
                href={plan.href}
                className={`mt-6 block rounded-lg px-4 py-3 text-center text-sm font-medium transition ${
                  plan.highlighted
                    ? "bg-blue-600 text-white hover:bg-blue-700"
                    : "bg-gray-900 text-white hover:bg-gray-800"
                }`}
              >
                {plan.cta}
              </a>

              <ul className="mt-6 space-y-3">
                {plan.features.map((feature) => (
                  <li key={feature} className="flex items-start gap-3 text-sm text-gray-600">
                    <Check className="mt-0.5 h-4 w-4 shrink-0 text-blue-600" />
                    {feature}
                  </li>
                ))}
              </ul>
            </motion.div>
          ))}
        </div>
      </div>
    </section>
  );
}
