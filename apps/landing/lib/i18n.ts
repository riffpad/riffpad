export type Language = "en" | "zh";

export const DEFAULT_LANGUAGE: Language = "en";

export const messages = {
  en: {
    nav: {
      features: "Features",
      pricing: "Pricing",
      faq: "FAQ",
      docs: "Docs",
      cta: "Start Creating",
    },
    hero: {
      badge: "AI-Native Code Sketchbook",
      h1Before: "A spark of inspiration",
      h1After: "→ a running prototype",
      description:
        "On the subway, in a café, while walking—just describe your idea by voice or text. Riffpad's Agent spins up an isolated sandbox, writes files, installs dependencies, and serves a live preview in under a second.",
      primaryCta: "Start creating free →",
      secondaryCta: "See how it works",
      trust: "No credit card · Sandbox ready in 500ms · Phone & tablet friendly",
      terminal: {
        title: "workspace preview",
        prompt: "Make a countdown webpage, just by describing it",
        status: "Agent is creating files, installing deps, starting preview...",
        ready: "→ Preview ready:",
        url: "https://preview.riffpad.app/p/abc123",
      },
    },
    features: {
      eyebrow: "From idea to delivery in three steps",
      title: "Treat ideas as code, not chat history",
      subtitle:
        "Stop scrolling through LLM threads for code snippets. Stop opening your laptop for a 5-minute validation.",
      items: {
        zeroFriction: {
          badge: "Before → Capture",
          title: "Zero-friction capture",
          description:
            "The input box is the homepage. No naming, no framework picker, no environment setup. We give every unnamed project a beautiful random codename.",
        },
        runnable: {
          badge: "After → Run",
          title: "Runnable thinking space",
          description:
            "The Agent doesn't just chat. It reads and writes files, runs Bash, installs packages, and serves web previews—HTML, React, Python, whatever you need.",
        },
        bridge: {
          badge: "Bridge → Deliver",
          title: "One-click IDE bridge",
          description:
            "When you're done, the Agent restructures the code, writes a high-quality README, and exports a Zip or pushes to GitHub for Cursor / Claude Code.",
        },
        safe: {
          badge: "Engineering",
          title: "Fast and isolated",
          description:
            "A warm sandbox pool keeps New Spark under 800ms P95. Every workspace runs kernel-isolated with CPU, memory, and network quotas, and hibernates after 48h of inactivity.",
        },
      },
    },
    testimonials: {
      eyebrow: "User voices",
      title: "They stopped letting ideas slip away",
      people: {
        indie: {
          quote:
            "I used to get an idea on the subway and forget half of it by the time I got home. Now I describe it to Riffpad, and the prototype is running before I arrive.",
          name: "Alex Lin",
          role: "Indie Developer",
        },
        pm: {
          quote:
            "As a technical PM, I can finally validate an API integration before a requirements review and drop a clickable prototype on the engineering team.",
          name: "Rachel Chen",
          role: "Technical Product Manager",
        },
        ai: {
          quote:
            "My ChatGPT experiments used to get buried in the scroll. Riffpad workspaces let me branch, persist, and export—finally.",
          name: "Daniel Wang",
          role: "AI Application Explorer",
        },
      },
    },
    pricing: {
      eyebrow: "Simple pricing",
      title: "Start free, pay only when it sticks",
      subtitle:
        "The Free plan is enough to validate most ideas. Upgrade when you need more power, long-running projects, or team collaboration.",
      free: {
        name: "Free",
        period: "forever free",
        description: "For indie hackers validating ideas on the go.",
        cta: "Start free",
      },
      pro: {
        name: "Pro",
        period: "/ month",
        description: "For creators who need faster sandboxes and smoother delivery.",
        cta: "Upgrade to Pro",
        popular: "Most popular",
      },
      team: {
        name: "Team",
        period: "/ user / month",
        description: "For small teams sharing workspaces and technical decisions.",
        cta: "Contact sales",
      },
      freeFeatures: [
        "10 Sparks per month",
        "512MB RAM / 1GB disk per sandbox",
        "Node.js + Python base image",
        "48h auto-hibernation",
        "Community support",
      ],
      proFeatures: [
        "Unlimited Sparks",
        "2GB RAM / 8GB disk per sandbox",
        "Priority warm pool, P95 \u003c 500ms",
        "Long-running projects (no 48h hibernation)",
        "One-click Zip / GitHub export",
        "Voice input + Whisper fallback",
        "Email support",
      ],
      teamFeatures: [
        "Everything in Pro",
        "Shared team workspaces",
        "SSO / SAML (coming soon)",
        "Private model endpoints",
        "Audit logs and quota management",
        "Priority technical support",
      ],
    },
    faq: {
      eyebrow: "FAQ",
      title: "Common questions",
      q1: "How is Riffpad different from Cursor or Windsurf?",
      a1: "Cursor is a heavy IDE for production code on your desktop. Riffpad is the upstream incubator built for phones, tablets, and fragmented moments. It turns a sentence into a runnable prototype in seconds, then bridges the result to Cursor / Claude Code.",
      q2: "Is coding on a phone painful?",
      a2: "You don't type much code on your phone. Riffpad is built around voice/text intent → Agent execution. The file tree, previews, and batch confirmations are redesigned for mobile; iPad gives you a near-desktop experience when you need it.",
      q3: "Is code inside the sandbox safe?",
      a3: "Every workspace runs in an isolated Firecracker microVM / gVisor environment. The Agent can only operate inside /sandbox/workspace/, and network, CPU, and memory are quota-limited. Dangerous commands are blocked by default.",
      q4: "Which downstream tools can I export to?",
      a4: "Export as a Zip archive or push to a temporary GitHub repo. The export includes the restructured code skeleton, README, validated technical path, and AI prompt context for Cursor, Claude Code, VS Code, and more.",
      q5: "Is the Free plan enough?",
      a5: "Free gives you 10 Sparks per month—plenty for most throwaway ideas. Upgrade to Pro or Team when you need daily long-running projects, more power, or collaboration.",
    },
    cta: {
      title: "Don't let another idea slip away today",
      subtitle:
        "Start free. Your first sandbox spins up in 500ms. Riffpad is ready when inspiration strikes.",
      primary: "Start creating free →",
      secondary: "Read docs",
      trust: "No credit card · Downgrade anytime",
    },
    footer: {
      description:
        "The AI-native code sketchbook. Capture, run, and deliver your next idea anywhere.",
      product: "Product",
      resources: "Resources",
      company: "Company",
      links: {
        features: "Features",
        pricing: "Pricing",
        changelog: "Changelog",
        docs: "Docs",
        github: "GitHub",
        status: "Status",
        about: "About",
        contact: "Contact",
        privacy: "Privacy",
      },
      deerflow: "Created By Deerflow",
    },
    languages: {
      en: "EN",
      zh: "中文",
    },
  },
  zh: {
    nav: {
      features: "功能",
      pricing: "价格",
      faq: "FAQ",
      docs: "文档",
      cta: "开始创作",
    },
    hero: {
      badge: "AI 时代的代码草稿本",
      h1Before: "一闪而过的灵感",
      h1After: "→ 能跑的原型",
      description:
        "在地铁上、咖啡馆里、走路时，用语音或文字描述你的想法；Riffpad 的 Agent 会在隔离沙箱里自动写代码、装依赖、跑服务，不到一秒就能看到实时预览。",
      primaryCta: "免费开始创作 →",
      secondaryCta: "看看怎么工作",
      trust: "无需信用卡 · 500ms 启动沙箱 · 支持手机/平板",
      terminal: {
        title: "workspace preview",
        prompt: "帮我做一个倒计时网页，语音输入即可",
        status: "Agent 正在创建文件、安装依赖、启动预览...",
        ready: "→ 预览已就绪：",
        url: "https://preview.riffpad.app/p/abc123",
      },
    },
    features: {
      eyebrow: "从想法到交付，只需三步",
      title: "把灵感当成代码，而不是聊天记录",
      subtitle:
        "告别在 LLM 对话里翻找代码片段，也告别为 5 分钟验证开电脑建项目。",
      items: {
        zeroFriction: {
          badge: "Before → 捕捉",
          title: "零摩擦记录",
          description:
            "打开就是输入框，无需起名、无需选框架、无需配环境。系统会自动分配富有美感的随机代号。",
        },
        runnable: {
          badge: "After → 运行",
          title: "可执行的思考空间",
          description:
            "Agent 不只是聊天。它能直接读写文件、运行 Bash、安装依赖、启动 Web 服务，右侧画布实时预览 HTML / React / Python 的运行结果。",
        },
        bridge: {
          badge: "Bridge → 交付",
          title: "一键桥接下游 IDE",
          description:
            "头脑风暴结束后，Agent 会自动重构代码骨架、生成高质量 README，打包成 Zip 或推送到 GitHub，Cursor / Claude Code 打开即可继续。",
        },
        safe: {
          badge: "Engineering",
          title: "极速且安全",
          description:
            "后台维护预热沙箱池，New Spark 响应 P95 \u003c 800ms。每个沙箱处于内核级隔离，CPU / 内存 / 网络严格配额，48 小时不活跃自动休眠。",
        },
      },
    },
    testimonials: {
      eyebrow: "用户原声",
      title: "他们不再让灵感溜走",
      people: {
        indie: {
          quote:
            "以前在地铁上想到一个点子，回家后已经忘了大半。现在用 Riffpad 语音说两句，到公司时原型已经跑通了。",
          name: "林晓峰",
          role: "独立开发者",
        },
        pm: {
          quote:
            "作为技术 PM，我终于能在需求评审前快速验证一个 API 联动方案，直接把可点击原型丢给研发。",
          name: "陈雨桐",
          role: "技术产品经理",
        },
        ai: {
          quote:
            "ChatGPT 里的实验代码终于不再被滚没了。Riffpad 的工作区让我能沉淀、分叉、导出，太舒服了。",
          name: "王浩然",
          role: "AI 应用探索者",
        },
      },
    },
    pricing: {
      eyebrow: "简单定价",
      title: "先用起来，再决定是否付费",
      subtitle:
        "Free 计划足以验证大多数灵感；当你需要更多算力和长驻项目时，Pro 会接住你。",
      free: {
        name: "Free",
        period: "永久免费",
        description: "适合个人开发者随手验证灵感。",
        cta: "免费开始",
      },
      pro: {
        name: "Pro",
        period: "/ 月",
        description: "适合重度创作者，让原型跑得更快、交付更顺。",
        cta: "升级到 Pro",
        popular: "最受欢迎",
      },
      team: {
        name: "Team",
        period: "/ 人 / 月",
        description: "适合小团队共享沙箱、沉淀技术方案。",
        cta: "联系销售",
      },
      freeFeatures: [
        "每月 10 个 Spark",
        "单沙箱 512MB 内存 / 1GB 磁盘",
        "Node.js + Python 基础镜像",
        "48h 自动休眠",
        "社区支持",
      ],
      proFeatures: [
        "无限 Spark",
        "单沙箱 2GB 内存 / 8GB 磁盘",
        "优先预热池，P95 \u003c 500ms",
        "长驻项目（取消 48h 休眠）",
        "一键导出 Zip / GitHub",
        "语音输入 + Whisper 回退",
        "邮件支持",
      ],
      teamFeatures: [
        "Pro 全部功能",
        "团队工作区共享",
        "SSO / SAML（即将上线）",
        "私有模型接入",
        "审计日志与配额管理",
        "优先技术支持",
      ],
    },
    faq: {
      eyebrow: "FAQ",
      title: "常见问题",
      q1: "Riffpad 和 Cursor / Windsurf 有什么区别？",
      a1: "Cursor 是重型 IDE，适合在电脑上写生产级代码；Riffpad 是最上游的灵感孵化器，专为手机、平板和碎片化场景设计，让你在 5 分钟内把一句话验证成可运行的原型，再桥接到 Cursor / Claude Code 继续开发。",
      q2: "手机上的代码体验会不会很难受？",
      a2: "你不会在手机上写大量代码。Riffpad 的核心交互是「语音/文字描述意图 → Agent 自动执行」。文件树、预览、批量确认都针对移动端做了重新设计，必要时还能在 iPad 上获得接近桌面的体验。",
      q3: "沙箱里的代码安全吗？",
      a3: "每个工作区运行在独立的 Firecracker microVM / gVisor 隔离环境中，Agent 默认只能操作 /sandbox/workspace/ 目录，网络、CPU、内存都有配额限制。高危命令会被默认拦截。",
      q4: "可以导出到哪些下游工具？",
      a4: "目前支持导出为 Zip 压缩包，以及一键推送到临时 GitHub 仓库。导出包包含重构后的代码骨架、README、技术路径说明和 AI 提示词上下文，方便 Cursor、Claude Code、VS Code 等直接接管。",
      q5: "免费计划够用吗？",
      a5: "Free 计划每月 10 个 Spark，足以验证大部分临时想法。如果你每天都有灵感需要长期驻留、更大算力或团队协作，再升级到 Pro / Team。",
    },
    cta: {
      title: "今天就不让一个灵感溜走",
      subtitle: "免费开始，500ms 启动你的第一个沙箱。灵感来的时候，Riffpad 已经准备好了。",
      primary: "免费开始创作 →",
      secondary: "阅读文档",
      trust: "无需信用卡 · 随时可降级回 Free",
    },
    footer: {
      description:
        "AI 时代的代码灵感草稿本。随时随地捕捉、运行、交付你的下一个想法。",
      product: "产品",
      resources: "资源",
      company: "公司",
      links: {
        features: "功能",
        pricing: "价格",
        changelog: "更新日志",
        docs: "文档",
        github: "GitHub",
        status: "状态页",
        about: "关于我们",
        contact: "联系我们",
        privacy: "隐私政策",
      },
      deerflow: "Created By Deerflow",
    },
    languages: {
      en: "EN",
      zh: "中文",
    },
  },
} as const;

export type Messages = (typeof messages)[Language];

export function getBrowserLanguage(): Language {
  if (typeof navigator === "undefined") return DEFAULT_LANGUAGE;
  const lang = navigator.language || (navigator as Navigator & { userLanguage?: string }).userLanguage;
  if (lang?.startsWith("zh")) return "zh";
  return DEFAULT_LANGUAGE;
}
