"use client";

import { motion } from "framer-motion";
import { Sparkles, Zap, Box, Bridge } from "./Icons";

const features = [
  {
    icon: Sparkles,
    title: "零摩擦记录",
    description:
      "打开就是输入框，无需起名、无需选框架、无需配环境。系统会自动分配富有美感的随机代号，让你的第一句话直接变成可执行沙箱。",
    badge: "Before → 捕捉",
  },
  {
    icon: Box,
    title: "可执行的思考空间",
    description:
      "Agent 不只是聊天。它能直接读写文件、运行 Bash、安装依赖、启动 Web 服务，右侧画布实时预览 HTML / React / Python 的运行结果。",
    badge: "After → 运行",
  },
  {
    icon: Bridge,
    title: "一键桥接下游 IDE",
    description:
      "头脑风暴结束后，Agent 会自动重构代码骨架、生成高质量 README，打包成 Zip 或推送到 GitHub，Cursor / Claude Code 打开即可继续。",
    badge: "Bridge → 交付",
  },
  {
    icon: Zap,
    title: "极速且安全",
    description:
      "后台维护预热沙箱池，New Spark 响应 P95 \u003c 800ms。每个沙箱处于内核级隔离，CPU / 内存 / 网络严格配额，48 小时不活跃自动休眠。",
    badge: "Engineering",
  },
];

const container = {
  hidden: { opacity: 0 },
  show: {
    opacity: 1,
    transition: { staggerChildren: 0.12 },
  },
};

const item = {
  hidden: { opacity: 0, y: 20 },
  show: { opacity: 1, y: 0 },
};

export function Features() {
  return (
    <section id="features" className="bg-gray-50 px-4 py-24 sm:px-6 lg:px-8">
      <div className="mx-auto max-w-6xl">
        <div className="mx-auto max-w-2xl text-center">
          <p className="text-sm font-semibold uppercase tracking-wider text-blue-600">
            从想法到交付，只需三步
          </p>
          <h2 className="mt-3 text-3xl font-bold tracking-tight text-gray-950 sm:text-4xl">
            把灵感当成代码，而不是聊天记录
          </h2>
          <p className="mt-4 text-gray-600">
            告别在 LLM 对话里翻找代码片段，也告别为 5 分钟验证开电脑建项目。
          </p>
        </div>

        <motion.div
          variants={container}
          initial="hidden"
          whileInView="show"
          viewport={{ once: true, margin: "-100px" }}
          className="mt-16 grid gap-6 sm:grid-cols-2 lg:grid-cols-4"
        >
          {features.map((feature) => {
            const Icon = feature.icon;
            return (
              <motion.div
                key={feature.title}
                variants={item}
                className="rounded-2xl border border-gray-200 bg-white p-6 shadow-sm transition hover:shadow-md"
              >
                <span className="inline-block rounded-full bg-blue-50 px-2.5 py-1 text-xs font-medium text-blue-700">
                  {feature.badge}
                </span>
                <div className="mt-4 flex h-10 w-10 items-center justify-center rounded-lg bg-gray-950 text-white">
                  <Icon className="h-5 w-5" />
                </div>
                <h3 className="mt-4 text-lg font-semibold text-gray-950">
                  {feature.title}
                </h3>
                <p className="mt-2 text-sm leading-relaxed text-gray-600">
                  {feature.description}
                </p>
              </motion.div>
            );
          })}
        </motion.div>
      </div>
    </section>
  );
}
