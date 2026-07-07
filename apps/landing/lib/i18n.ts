export type Language = "en" | "zh";

export const DEFAULT_LANGUAGE: Language = "en";

export const messages = {
  en: {
    nav: {
      docs: "Docs",
      blog: "Blog",
      pricing: "Pricing",
    },
    hero: {
      h1: "The AI-native workspace in seconds.",
      description:
        "Brainstorm, capture ideas, and build light prototypes anywhere—on your phone, tablet, or desktop. Riffpad turns a sentence into a running workspace, no setup required.",
      trust: "Join the waitlist · No spam · Early access",
      chat: {
        title: "Riffpad",
        userMessage: "Build me a countdown page for my product launch.",
        assistantIntro: "Done. I've created the files and started the preview.",
        codeSnippet: "const target = new Date('2026-08-01');",
        assistantOutro: "Tap the card to see it live.",
        inputPlaceholder: "Describe your idea...",
      },
    },
    waitlist: {
      cta: "Join waitlist",
      placeholder: "you@example.com",
      success: "Thanks! You're on the list.",
    },
    features: {
      eyebrow: "From idea to delivery in three steps",
      title: "Treat ideas as code, not chat history",
      subtitle:
        "Stop scrolling through LLM threads for snippets. Stop losing sparks in scattered chats.",
      blocks: [
        {
          title: "Ideas become files, not forgotten threads",
          description:
            "Every idea is saved to a workspace with a file tree, version history, and a live preview. No more digging through chat history to find what the AI generated.",
        },
        {
          title: "Run in seconds, on any device",
          description:
            "No local setup, no dependency hell, no terminal. Cloud-synced across phone, tablet, and desktop. Describe what you want and watch it run.",
        },
        {
          title: "AI power without the engineering overhead",
          description:
            "Get the execution ability of Claude Code or Codex without managing repos, environments, or local machines.",
        },
        {
          title: "Hand off to any downstream agent",
          description:
            "Export your workspace context as an Agent Skill, MCP server, or Zip—ready for Cursor, Claude Code, or your own pipeline.",
        },
      ],
    },
    testimonials: {
      eyebrow: "User voices",
      title: "They stopped letting ideas slip away",
      people: {
        indie: {
          quote:
            "I used to get an idea on the subway and forget half of it by the time I got home. Now I describe it to Riffpad, and the prototype is running before I arrive.",
          name: "Alex Lin",
          role: "Indie Creator",
        },
        pm: {
          quote:
            "As a product manager, I can finally validate a workflow before a review and drop a clickable prototype on the team.",
          name: "Rachel Chen",
          role: "Product Manager",
        },
        ai: {
          quote:
            "My ChatGPT experiments used to get buried in the scroll. Riffpad workspaces let me branch, persist, and export—finally.",
          name: "Daniel Wang",
          role: "AI Power User",
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
        description: "For creators validating ideas on the go.",
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
        description: "For small teams sharing workspaces and decisions.",
        cta: "Contact sales",
      },
      freeFeatures: [
        "10 Sparks per month",
        "512MB RAM / 1GB disk per workspace",
        "Node.js + Python base image",
        "48h auto-hibernation",
        "Community support",
      ],
      proFeatures: [
        "Unlimited Sparks",
        "2GB RAM / 8GB disk per workspace",
        "Priority warm pool, P95 < 500ms",
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
      a1: "Cursor is a heavy IDE for production code on your desktop. Riffpad is the upstream workspace built for phones, tablets, and fragmented moments. It turns a sentence into a running workspace in seconds, then bridges the result to Cursor / Claude Code.",
      q2: "Do I need to know how to code?",
      a2: "Not necessarily. Riffpad is built around voice/text intent → Agent execution. You describe what you want, and the Agent generates files, installs dependencies, and runs previews. Power users can always open the code.",
      q3: "Is my workspace data safe?",
      a3: "Every workspace runs in an isolated Firecracker microVM / gVisor environment. The Agent can only operate inside /sandbox/workspace/, and network, CPU, and memory are quota-limited. Dangerous commands are blocked by default.",
      q4: "Which downstream tools can I export to?",
      a4: "Export as a Zip archive, an Agent Skill, or an MCP server. You can also push to a temporary GitHub repo. The export includes restructured files, README, validated path, and prompt context for Cursor, Claude Code, VS Code, and more.",
      q5: "Is the Free plan enough?",
      a5: "Free gives you 10 Sparks per month—plenty for most throwaway ideas. Upgrade to Pro or Team when you need daily long-running projects, more power, or collaboration.",
    },
    footer: {
      description:
        "The AI-native workspace. Capture, run, and share your next idea anywhere.",
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
        jobs: "Jobs",
      },
    },
    languages: {
      en: "EN",
      zh: "中文",
    },
  },
  zh: {
    nav: {
      docs: "文档",
      blog: "博客",
      pricing: "价格",
    },
    hero: {
      h1: "几秒钟内，拥有 AI 原生工作空间。",
      description:
        "无论在手机、平板还是电脑上，随时随地头脑风暴、记录灵感、搭建轻量原型。只需一句话，Riffpad 就能把它变成一个可运行的工作空间。",
      trust: "加入等待列表 · 不滥发邮件 · 抢先体验",
      chat: {
        title: "Riffpad",
        userMessage: "帮我做一个产品发布倒计时页面。",
        assistantIntro: "已完成。文件已生成，预览已启动。",
        codeSnippet: "const target = new Date('2026-08-01');",
        assistantOutro: "点击下方卡片即可查看实时效果。",
        inputPlaceholder: "描述你的想法...",
      },
    },
    waitlist: {
      cta: "加入等待列表",
      placeholder: "you@example.com",
      success: "已收到！你在列表里了。",
    },
    features: {
      eyebrow: "从想法到交付，只需三步",
      title: "把灵感当成代码，而不是聊天记录",
      subtitle:
        "告别在 LLM 对话里翻找片段，也告别让灵感散落在各种聊天窗口中。",
      blocks: [
        {
          title: "灵感变成文件，而不是被遗忘的对话",
          description:
            "每个想法都会保存到带有文件树、版本历史和实时预览的工作区。再也不用去聊天记录里翻找 AI 生成的内容。",
        },
        {
          title: "几秒运行，任何设备可用",
          description:
            "无需本地配置、无需依赖地狱、无需终端。手机、平板、桌面多端云同步。说出你想要什么，看它直接跑起来。",
        },
        {
          title: "AI 能力，无需工程负担",
          description:
            "拥有 Claude Code 或 Codex 的执行能力，却不需要管理仓库、环境或本地机器。",
        },
        {
          title: "一键交给任意下游 Agent",
          description:
            "将工作区上下文导出为 Agent Skill、MCP 服务器或 Zip，直接交给 Cursor、Claude Code 或你自己的工作流。",
        },
      ],
    },
    testimonials: {
      eyebrow: "用户原声",
      title: "他们不再让灵感溜走",
      people: {
        indie: {
          quote:
            "以前在地铁上想到一个点子，回家后已经忘了大半。现在用 Riffpad 语音说两句，到公司时原型已经跑通了。",
          name: "林晓峰",
          role: "独立创作者",
        },
        pm: {
          quote:
            "作为产品经理，我终于能在评审前快速验证一个工作流，直接把可点击原型丢给团队。",
          name: "陈雨桐",
          role: "产品经理",
        },
        ai: {
          quote:
            "ChatGPT 里的实验终于不再被滚没了。Riffpad 的工作区让我能沉淀、分叉、导出，太舒服了。",
          name: "王浩然",
          role: "AI 重度用户",
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
        description: "适合创作者随手验证灵感。",
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
        description: "适合小团队共享工作区、沉淀方案。",
        cta: "联系销售",
      },
      freeFeatures: [
        "每月 10 个 Spark",
        "单工作区 512MB 内存 / 1GB 磁盘",
        "Node.js + Python 基础镜像",
        "48h 自动休眠",
        "社区支持",
      ],
      proFeatures: [
        "无限 Spark",
        "单工作区 2GB 内存 / 8GB 磁盘",
        "优先预热池，P95 < 500ms",
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
      a1: "Cursor 是重型 IDE，适合在电脑上写生产级代码；Riffpad 是最上游的 AI 工作区，专为手机、平板和碎片化场景设计，让你在几秒钟内把一句话变成可运行的工作区，再桥接到 Cursor / Claude Code 继续开发。",
      q2: "我需要会写代码吗？",
      a2: "不一定。Riffpad 的核心交互是「语音/文字描述意图 → Agent 自动执行」。你描述想要什么，Agent 就会生成文件、安装依赖并运行预览。进阶用户当然也可以直接打开代码。",
      q3: "我的工作区数据安全吗？",
      a3: "每个工作区运行在独立的 Firecracker microVM / gVisor 隔离环境中，Agent 默认只能操作 /sandbox/workspace/ 目录，网络、CPU、内存都有配额限制。高危命令会被默认拦截。",
      q4: "可以导出到哪些下游工具？",
      a4: "目前支持导出为 Zip 压缩包、Agent Skill 或 MCP 服务器，也可以一键推送到临时 GitHub 仓库。导出包包含重构后的文件、README、技术路径说明和提示词上下文，方便 Cursor、Claude Code、VS Code 等直接接管。",
      q5: "免费计划够用吗？",
      a5: "Free 计划每月 10 个 Spark，足以验证大部分临时想法。如果你每天都有灵感需要长期驻留、更大算力或团队协作，再升级到 Pro / Team。",
    },
    footer: {
      description:
        "AI 原生工作空间。随时随地捕捉、运行、分享你的下一个想法。",
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
        jobs: "招聘",
      },
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
