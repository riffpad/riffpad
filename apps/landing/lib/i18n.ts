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
      h1: "A real cloud AI Agent, ready to use.",
      description:
        "On your phone, tablet, or desktop — describe an idea and Riffpad turns it into a running workspace. Capture ideas, handle files, validate prototypes, then hand off to Cursor / Claude Code with one click.",
      trust: "Join the waitlist · No spam · Early access",
      contact: {
        email: "Copy email",
        copied: "Copied!",
        discord: "Join Discord",
      },
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
    saasWorkspace: {
      files: "Files",
      docs: "docs",
      editor: "Editor",
      preview: "Preview",
      empty: "Select a file",
      chat: "Chat",
      userMessage: "Do a competitive analysis for cloud AI Agent products.",
      stepThinking: "Thinking...",
      stepSearch: "Web search: AI coding tools 2026",
      stepWrite: "Write file: docs/competitive-analysis.md",
      assistantReply: "Written",
      assistantReplySuffix: "with 11 competitor profiles, market data, and strategic recommendations.",
      inputPlaceholder: "Describe your idea...",
      filesContent: {
        prd: {
          name: "prd.md",
          content: `# Riffpad PRD

## Positioning
AI-native code sketchbook. Capture inspiration anywhere and turn a sentence into a running workspace.

## Core Flow
1. Describe an idea in plain language.
2. Agent builds files and starts a sandbox preview.
3. Validate, then bridge to Cursor / Claude Code.
`,
        },
        analysis: {
          name: "competitive-analysis.md",
          content: `# Competitive Analysis

## Market
- 84% of developers use or plan to use AI tools.
- 62% cite context window as the biggest pain point.

## Competitors
1. ChatGPT / Gemini — chat-first, no running workspace.
2. Cursor / Claude Code — heavy, local-first IDE agents.
3. v0 / Bolt / Lovable — prompt-to-app builders with platform lock-in.

## Riffpad's gap
Upstream incubator: lighter than an IDE, more structured than chat, more open than app builders.
`,
        },
      },
    },
    skillsMockup: {
      chat: "Chat",
      you: "You",
      agent: "Agent",
      userMessageShort: "install this agent skill for me",
      showMore: "show more",
      userMessage: `Install this Agent Skill for me.

Skill page: https://skillsmp.com/creators/anthropics/skills/skills-frontend-design
Source URL: https://github.com/anthropics/skills/tree/main/skills/frontend-design
Skill name: frontend-design
Creator: anthropics
Preferred install command: npx skills add https://github.com/anthropics/skills --skill frontend-design

Please open the skill page and source, review SKILL.md plus any companion files, and explain anything risky before installing.

If shell commands are available, prefer the install command above. If you install manually, copy the complete skill directory that contains SKILL.md, including scripts, references, assets, agents, and any other files shown on the skill page. Preserve the relative folder structure. Do not install SKILL.md alone. After installing, verify the target skills folder contains SKILL.md and all companion files needed by this skill.`,
      agentStep: "Running command: npx skills add https://github.com/anthropics/skills --skill frontend-design",
      thinking: "Thinking...",
      runCommandLabel: "Run command",
      runCommand: "npx skills add https://github.com/anthropics/skills --skill frontend-design",
      assistantReply: "Skill",
      assistantReplyName: "frontend-design",
      assistantReplySuffix: "has been installed successfully.",
      inputPlaceholder: "Describe your idea...",
    },
    mobileWorkspace: {
      workspaceName: "amber-spark-9",
      workspaceLabel: "workspace",
      userMessage: "Add a comparison between Cursor and Claude Code.",
      stepRead: "Reading docs/competitive-analysis.md",
      stepSearch: "Web search: Cursor vs Claude Code 2026",
      stepWrite: "Write file: docs/competitive-analysis.md",
      filesTitle: "Files",
      prdFile: "prd.md",
      analysisFile: "competitive-analysis.md",
      assistantReply: "Updated",
      assistantReplyName: "competitive-analysis.md",
      assistantReplySuffix:
        "with a new comparison of Cursor and Claude Code across local-first setup, cloud sync, and engineering scenarios.",
      previewTitle: "Preview",
      previewType: "Markdown",
      previewHeading: "Cursor / Claude Code",
      previewPoints: [
        "Local-first IDE agent",
        "Requires repo & environment setup",
        "Best for heavy engineering",
      ],
      voiceText: "Add a comparison between Cursor and Claude Code",
      listening: "Listening...",
      holdToSpeak: "Hold to speak",
    },
    features: {
      eyebrow: "What Riffpad does",
      title: "A cloud AI Agent in your pocket",
      subtitle:
        "No installs, no setup, no scattered chat threads. Describe what you want and get a running workspace with files, preview, and an agent.",
      blocks: [
        {
          title: "Skills + MCP, extend on demand",
          description:
            "Install Skills and connect MCP servers to give the agent real capabilities — from generating polished frontends and documents to syncing with GitHub, Linear, and Notion.",
        },
        {
          title: "Use Riffpad anywhere",
          description:
            "Built for fragmented time. Capture ideas on your phone with voice input while walking, commuting, or between meetings.",
        },
        {
          title: "Hand off to Cursor / Claude Code",
          description:
            "When the prototype is validated, export the full context as a prompt package or a clean .zip skeleton — ready for serious engineering.",
        },
      ],
    },
    comparison: {
      eyebrow: "How Riffpad compares",
      title: "The upstream workspace for AI-native ideas",
      subtitle:
        "Riffpad sits between chat and production IDE — lighter than a coding agent, more structured than a chatbot.",
      tables: [
        {
          title: "vs. ChatGPT / Gemini",
          headers: ["Riffpad", "ChatGPT / Gemini"],
          rows: [
            {
              riffpad: "Agent loop that keeps running and using tools until the problem is solved",
              other: "Single or few LLM calls; user manually pushes the conversation forward",
            },
            {
              riffpad: "Install Skills to extend capabilities on demand",
              other: "Limited to the platform's built-in ecosystem",
            },
            {
              riffpad: "Real file system; knowledge persists in an exportable workspace",
              other: "Content is scattered across a chat scroll",
            },
          ],
        },
        {
          title: "vs. Codex / Claude Code",
          headers: ["Riffpad", "Codex / Claude Code"],
          rows: [
            {
              riffpad: "Start with one sentence; no install, repo, or environment setup",
              other: "Requires local install, directory creation, and environment configuration",
            },
            {
              riffpad: "Runs in the cloud and syncs across phone, tablet, and desktop",
              other: "Primarily runs on your local machine",
            },
            {
              riffpad: "Built for fast, lightweight, creative validation",
              other: "Built for heavy engineering on existing codebases",
            },
          ],
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
    pages: {
      docs: {
        title: "Docs",
        description: "Get started with Riffpad.",
        intro: "Riffpad turns a sentence into a running workspace.",
        quickStart: "Quick start",
        quickStartText:
          "Join the waitlist to get early access. Once invited, describe your idea and Riffpad will spin up a sandbox with files, a live preview, and an agent.",
        help: "Need help?",
        helpText: "Email us at hi@riffpad.ai or join our Discord community.",
      },
      changelog: {
        title: "Changelog",
        description: "Updates and improvements to Riffpad.",
        items: [
          {
            version: "v0.2.0",
            date: "2026-07-08",
            content: "Refreshed landing page, added waitlist, Discord/email contact, and mobile menu redesign.",
          },
          {
            version: "v0.1.0",
            date: "2026-06-01",
            content: "Initial landing page and waitlist signup.",
          },
        ],
      },
      about: {
        title: "About",
        description: "The AI-native workspace in seconds.",
        mission:
          "We believe the best ideas appear in fragmented moments — on the subway, during a walk, between meetings. Riffpad exists to capture those ideas before they fade, and turn them into something real.",
        team: "Team",
        teamText: "We are a small team building Riffpad in public. Want to join us? Check out our jobs page.",
      },
      privacy: {
        title: "Privacy Policy",
        description: "How Riffpad handles your data.",
        lastUpdated: "Last updated: July 8, 2026",
        sections: [
          {
            title: "Information we collect",
            content:
              "We collect your email address when you join the waitlist. We may also collect basic usage data and cookies for theme preferences.",
          },
          {
            title: "How we use information",
            content:
              "We use your email to send waitlist updates and product announcements. We do not sell your personal information.",
          },
          {
            title: "Third-party services",
            content:
              "We use Formspree for waitlist submissions and Supabase for authentication and data storage when the app is live.",
          },
          {
            title: "Contact us",
            content: "If you have questions about this policy, email hi@riffpad.ai.",
          },
        ],
      },
      jobs: {
        title: "Join us",
        description: "Help build the AI-native workspace.",
        noRoles: "No open roles right now.",
        contact:
          "We are always curious about exceptional people. Send a note to hi@riffpad.ai and tell us what you would build.",
      },
    },
  },
  zh: {
    nav: {
      docs: "文档",
      blog: "博客",
      pricing: "价格",
    },
    hero: {
      h1: "真正的云端 AI Agent，开箱即用。",
      description:
        "无论在电脑、手机还是平板，一句话就能启动一个可运行的工作空间——记录灵感、处理文件、验证原型，然后一键交给 Cursor / Claude Code 继续开发。",
      trust: "加入等待列表 · 不滥发邮件 · 抢先体验",
      contact: {
        email: "复制邮箱",
        copied: "已复制",
        discord: "加入 Discord",
      },
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
    saasWorkspace: {
      files: "文件",
      docs: "docs",
      editor: "编辑",
      preview: "预览",
      empty: "选择文件以查看",
      chat: "对话",
      userMessage: "帮我做一个云端 AI Agent 产品的调研分析",
      stepThinking: "思考中...",
      stepSearch: "网页搜索：AI 编程工具 2026",
      stepWrite: "写入文件：docs/competitive-analysis.md",
      assistantReply: "已写入",
      assistantReplySuffix: "，包含 11 个竞品画像、市场数据和策略建议。",
      inputPlaceholder: "描述你的想法...",
      filesContent: {
        prd: {
          name: "prd.md",
          content: `# Riffpad 产品需求文档

## 定位
AI 原生代码速写本。随时随地捕捉灵感，把一句话变成可运行的工作空间。

## 核心流程
1. 用自然语言描述想法。
2. Agent 生成文件并启动沙箱预览。
3. 验证后桥接到 Cursor / Claude Code。
`,
        },
        analysis: {
          name: "competitive-analysis.md",
          content: `# 竞品调研分析

## 市场
- 84% 的开发者正在使用或计划使用 AI 工具。
- 62% 的开发者认为上下文窗口是最大的痛点。

## 竞品
1. ChatGPT / Gemini —— 以聊天为主，没有可运行工作空间。
2. Cursor / Claude Code —— 重型、本地优先的 IDE Agent。
3. v0 / Bolt / Lovable —— 一句话生成应用，但平台锁定严重。

## Riffpad 的差异化
上游孵化器：比 IDE 更轻，比聊天更有结构，比应用生成器更开放。
`,
        },
      },
    },
    skillsMockup: {
      chat: "对话",
      you: "用户",
      agent: "Agent",
      userMessageShort: "帮我安装这个 agent skill",
      showMore: "显示更多",
      userMessage: `帮我安装这个 Agent Skill。\n\nSkill page: https://skillsmp.com/creators/anthropics/skills/skills-frontend-design\nSource URL: https://github.com/anthropics/skills/tree/main/skills/frontend-design\nSkill name: frontend-design\nCreator: anthropics\nPreferred install command: npx skills add https://github.com/anthropics/skills --skill frontend-design\n\nPlease open the skill page and source, review SKILL.md plus any companion files, and explain anything risky before installing.\n\nIf shell commands are available, prefer the install command above. If you install manually, copy the complete skill directory that contains SKILL.md, including scripts, references, assets, agents, and any other files shown on the skill page. Preserve the relative folder structure. Do not install SKILL.md alone. After installing, verify the target skills folder contains SKILL.md and all companion files needed by this skill.`,
      agentStep: "运行命令：npx skills add https://github.com/anthropics/skills --skill frontend-design",
      thinking: "思考中...",
      runCommandLabel: "运行命令",
      runCommand: "npx skills add https://github.com/anthropics/skills --skill frontend-design",
      assistantReply: "Skill",
      assistantReplyName: "frontend-design",
      assistantReplySuffix: "已安装完毕",
      inputPlaceholder: "描述你的想法...",
    },
    mobileWorkspace: {
      workspaceName: "amber-spark-9",
      workspaceLabel: "workspace",
      userMessage: "再补充一下 Cursor 和 Claude Code 的对比。",
      stepRead: "读取 docs/competitive-analysis.md",
      stepSearch: "网页搜索：Cursor vs Claude Code 2026",
      stepWrite: "写入文件：docs/competitive-analysis.md",
      filesTitle: "文件",
      prdFile: "prd.md",
      analysisFile: "competitive-analysis.md",
      assistantReply: "已更新",
      assistantReplyName: "competitive-analysis.md",
      assistantReplySuffix:
        "，新增了 Cursor 与 Claude Code 在本地优先、云端同步和工程场景上的对比。",
      previewTitle: "预览",
      previewType: "Markdown",
      previewHeading: "Cursor / Claude Code",
      previewPoints: [
        "本地优先的 IDE Agent",
        "需要仓库和环境配置",
        "适合重型工程",
      ],
      voiceText: "再补充一下 Cursor 和 Claude Code 的对比",
      listening: "聆听中...",
      holdToSpeak: "按住说话",
    },
    features: {
      eyebrow: "Riffpad 能做什么",
      title: "随身可用的云端 AI Agent",
      subtitle:
        "无需安装，无需配置，没有散落在聊天里的片段。描述你想要什么，就能得到一个带文件、预览和 Agent 的可运行工作空间。",
      blocks: [
        {
          title: "Skill + MCP，按需扩展 Agent",
          description:
            "安装 Skill、接入 MCP 服务器，让 Agent 真正拥有能力——从生成精美前端和文档，到同步 GitHub、Linear、Notion，一切按需启用。",
        },
        {
          title: "在任何地方使用 Riffpad",
          description:
            "为碎片时间而生。走路、通勤、会议间隙，用手机语音输入就能创建原型。",
        },
        {
          title: "一键交给 Cursor / Claude Code",
          description:
            "原型验证完成后，导出完整上下文：Prompt 包或干净的 .zip 项目骨架，直接交给工程师继续开发。",
        },
      ],
    },
    comparison: {
      eyebrow: "Riffpad 有什么不同",
      title: "AI 原生想法的上游工作空间",
      subtitle:
        "Riffpad 位于聊天工具和生产 IDE 之间——比 Coding Agent 更轻，比聊天机器人更有结构。",
      tables: [
        {
          title: "vs. ChatGPT / Gemini",
          headers: ["Riffpad", "ChatGPT / Gemini"],
          rows: [
            {
              riffpad: "Agent loop，持续运行并调用工具直到问题解决",
              other: "单次或少量 LLM 调用，需要用户手动推进",
            },
            {
              riffpad: "可安装 Skill，按需扩展能力",
              other: "能力受限于平台自带生态",
            },
            {
              riffpad: "真实的文件系统，知识沉淀在可导出的工作区",
              other: "内容散落在对话流中",
            },
          ],
        },
        {
          title: "vs. Codex / Claude Code",
          headers: ["Riffpad", "Codex / Claude Code"],
          rows: [
            {
              riffpad: "一句话启动，无需安装、建仓库、配环境",
              other: "需要本地安装、新建目录、配置环境",
            },
            {
              riffpad: "云端运行，手机 / 平板 / 电脑多端同步",
              other: "主要运行在本地机器",
            },
            {
              riffpad: "适合快速、轻度、创意型验证",
              other: "适合已有代码库的重型工程",
            },
          ],
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
    pages: {
      docs: {
        title: "文档",
        description: "开始使用 Riffpad。",
        intro: "Riffpad 把一句话变成一个可运行的工作空间。",
        quickStart: "快速开始",
        quickStartText:
          "加入等待列表以获得抢先体验。获得邀请后，描述你的想法，Riffpad 会为你启动一个包含文件、实时预览和 Agent 的沙箱。",
        help: "需要帮助？",
        helpText: "发送邮件至 hi@riffpad.ai，或加入我们的 Discord 社区。",
      },
      changelog: {
        title: "更新日志",
        description: "Riffpad 的更新与改进。",
        items: [
          {
            version: "v0.2.0",
            date: "2026-07-08",
            content: "刷新 landing page、接入等待列表、Discord/邮箱联系、以及移动端菜单重设计。",
          },
          {
            version: "v0.1.0",
            date: "2026-06-01",
            content: "初始 landing page 和等待列表注册。",
          },
        ],
      },
      about: {
        title: "关于",
        description: "几秒钟内，拥有 AI 原生工作空间。",
        mission:
          "我们相信最好的灵感诞生于碎片化的时刻——在地铁上、散步时、会议间隙。Riffpad 的存在，就是在这些灵感消失之前捕捉它们，并把它们变成真实的东西。",
        team: "团队",
        teamText: "我们是一支公开构建 Riffpad 的小团队。想加入我们？看看招聘页面。",
      },
      privacy: {
        title: "隐私政策",
        description: "Riffpad 如何处理你的数据。",
        lastUpdated: "最后更新：2026 年 7 月 8 日",
        sections: [
          {
            title: "我们收集的信息",
            content:
              "当你加入等待列表时，我们会收集你的电子邮箱地址。我们也可能收集基础使用数据和主题偏好 Cookie。",
          },
          {
            title: "我们如何使用信息",
            content:
              "我们使用你的邮箱发送等待列表更新和产品公告。我们不会出售你的个人信息。",
          },
          {
            title: "第三方服务",
            content:
              "我们使用 Formspree 处理等待列表提交，应用上线后将使用 Supabase 进行认证和数据存储。",
          },
          {
            title: "联系我们",
            content: "如果你对本政策有疑问，请发送邮件至 hi@riffpad.ai。",
          },
        ],
      },
      jobs: {
        title: "加入我们",
        description: "一起构建 AI 原生工作空间。",
        noRoles: "目前没有开放职位。",
        contact:
          "我们始终对优秀的人充满好奇。发送邮件至 hi@riffpad.ai，告诉我们你想构建什么。",
      },
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
