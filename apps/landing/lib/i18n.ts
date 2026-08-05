export type Language = "en" | "zh";

const en = {
  hero: {
    title1: "Your AI coding agents,",
    title2: "in your pocket.",
    description:
      "Riffpad bridges Claude Code, Codex and other CLI agents to your phone — watch progress, approve actions and steer sessions without staying chained to the desk.",
    ctaPrimary: "Get early access",
    ctaSecondary: "Read the docs",
    note: "local daemon · end-to-end encrypted · zero-knowledge relay",
  },
  install: {
    unix: "macOS / Linux",
    windows: "Windows",
    copy: "copy",
    copied: "copied ✓",
  },
  mockup: {
    hint: "Live demo — send a message from the phone and watch the Mac react.",
    mac: {
      title: "riffpad — zsh — 80×24",
      prompt: "~/projects/api % codex exec --json",
      lines: [
        { text: "reading task: refactor auth middleware", tone: "ok" },
        { text: "edit_file  src/auth/middleware.ts", tone: "info" },
        { text: "run_tests  go test ./...", tone: "info" },
        { text: "waiting for approval · delete src/old.ts", tone: "warn" },
      ],
      status: "riffpad daemon · e2ee ●",
      fromPhone: "⏎ phone",
      approvedLine: "approved from phone · resuming agent",
      rejectedLine: "rejected from phone · agent paused",
    },
    phone: {
      title: "riffpad",
      synced: "synced",
      session: "s_9f2a",
      cli: "claude",
      running: "running",
      tools: "2 tool calls",
      approval: "approval request",
      summary: "delete src/old.ts",
      approve: "Approve",
      reject: "Reject",
      approved: "approved · agent resumed",
      rejected: "rejected · agent paused",
      hello: "Session attached — I'll ping you when I need a decision.",
      input: "message agent…",
      tabs: ["Sessions", "Session", "Settings"],
      presets: [
        {
          send: "Approve the deletion",
          ack: "Approved — continuing the refactor.",
          term: [
            { text: "approval granted · delete src/old.ts", tone: "ok" },
            { text: "refactor auth middleware · 3 files changed", tone: "info" },
          ],
        },
        {
          send: "Run the tests first",
          ack: "On it — tests run before any further edits.",
          term: [
            { text: "run_tests  go test ./...", tone: "info" },
            { text: "42 passed · 0 failed · 1.8s", tone: "ok" },
          ],
        },
        {
          send: "Use CSV output instead",
          ack: "Got it — switching output to CSV.",
          term: [
            { text: "new instruction · output format → csv", tone: "info" },
            { text: "edit_file  src/report/export.ts", tone: "info" },
          ],
        },
      ],
    },
    sync: {
      label: "e2ee",
      status: "synced",
      syncing: "syncing…",
      latency: "84ms",
    },
  },
  terminal: {
    title: "riffpad daemon — attached",
    connection: "relay api.riffpad.ai · encrypted",
    session: "session s_9f2a · claude · ",
    running: "running",
    approval: "approval_request",
    approvalSummary: "delete src/old.ts",
    approve: "approve",
    reject: "reject",
    statusE2ee: "e2ee aes-256-gcm",
    statusRelay: "relay zero-knowledge",
    statusLatency: "latency 84ms",
    hintTab: "tab switch agent",
    hintCmd: "ctrl-p commands",
    hintEsc: "esc approve",
  },
  features: {
    label: "features",
    title: "Built for the moments your agent needs you",
    subtitle: "No more camping next to the laptop while a long task runs.",
    items: [
      {
        title: "Real-time supervision",
        description:
          "Structured events for status, tool calls, file changes and commands — with a terminal fallback when a CLI doesn't expose them.",
      },
      {
        title: "Approve from anywhere",
        description:
          "Approval requests become push notifications. One tap to allow, deny, or edit the condition before saying yes.",
      },
      {
        title: "Steer mid-flight",
        description:
          "Send a new instruction to a running session from your phone. Your agent changes direction while you stay away from the desk.",
      },
      {
        title: "Privacy by construction",
        description:
          "Agents run on your machine with your keys and your repos. The relay is zero-knowledge: it routes ciphertext and stores nothing.",
      },
    ],
  },
  how: {
    label: "how-it-works",
    title: "Three hops. Zero cloud custody.",
    subtitle: "The loop is short, and every hop is encrypted.",
    steps: [
      {
        title: "Install the daemon",
        description:
          "One binary on macOS or Linux. It wraps or attaches to your CLI sessions and owns the keys locally.",
      },
      {
        title: "Pair your phone",
        description:
          "Scan a QR code to exchange device keys (X25519). Each session derives an ephemeral key of its own.",
      },
      {
        title: "Supervise anywhere",
        description:
          "Watch encrypted events, approve, reject and steer — over cellular, in the subway, from bed.",
      },
    ],
  },
  security: {
    label: "security",
    title: "Zero-knowledge by design",
    subtitle: "The relay cannot read your sessions. The product is built so it never has to.",
    items: [
      {
        title: "Local-first",
        description:
          "Code, repos and API keys never leave your computer. The daemon only bridges the CLI to your phone.",
      },
      {
        title: "End-to-end encrypted",
        description:
          "X25519 key exchange plus AES-256-GCM. Keys live on your daemon and your phone.",
      },
      {
        title: "Relay routes, never reads",
        description:
          "api.riffpad.ai forwards ciphertext and keeps no message history. No logs, no content store.",
      },
      {
        title: "Read-only by default",
        description:
          "Viewing is passive. Approve, reject, pause and prompt are explicit actions — and the daemon has a one-key kill switch.",
      },
    ],
  },
  faq: {
    label: "faq",
    title: "Common questions",
    items: [
      {
        q: "Which coding CLIs are supported?",
        a: "Today: Claude Code and Codex, via structured output plus hooks. DeepSeek CLI, Kimi and GLM are next, with tmux/PTY as the universal fallback.",
      },
      {
        q: "Where do my code and API keys live?",
        a: "On your computer. Riffpad stores no API key material; the daemon only forwards the session to your phone.",
      },
      {
        q: "Can the relay read my session?",
        a: "No. Keys are exchanged directly between your daemon and phone, so the relay only ever sees ciphertext — and keeps none of it.",
      },
      {
        q: "Is the phone client read-only?",
        a: "By default, yes. Every remote action — approve, reject, edit condition, send a prompt — is an explicit tap you make.",
      },
      {
        q: "What's the current status?",
        a: "The M0 closed loop is verified: attach a Claude session, get an approval card on the phone, approve, and the agent continues. MVP (M1) is in progress.",
      },
    ],
  },
  cta: {
    title: "Stop babysitting the terminal.",
    description: "Get early access and approve from the subway instead of the desk chair.",
    button: "Email hi@riffpad.ai",
    note: "No spam. We build in public.",
  },
  footer: {
    tagline: "The pocket remote for your AI coding agents.",
    github: "GitHub",
    docs: "Docs",
    contact: "Contact",
    copyright: "© 2026 Riffpad",
    rights: "local-first · e2ee · zero-knowledge",
  },
  languages: {
    en: "EN",
    zh: "中",
  },
};

const zh: typeof en = {
  hero: {
    title1: "你的 AI 编程 Agent，",
    title2: "装进你的口袋。",
    description:
      "Riffpad 把 Claude Code、Codex 等 CLI agent 桥接到手机：随时看进度、批准操作、远程转向，人不必守在电脑前。",
    ctaPrimary: "抢先体验",
    ctaSecondary: "阅读文档",
    note: "本地 daemon · 端到端加密 · 零知识中继",
  },
  install: {
    unix: "macOS / Linux",
    windows: "Windows",
    copy: "复制",
    copied: "已复制 ✓",
  },
  mockup: {
    hint: "可交互演示：在手机上发一条消息，看 Mac 终端实时响应。",
    mac: {
      title: "riffpad — zsh — 80×24",
      prompt: "~/projects/api % codex exec --json",
      lines: [
        { text: "读取任务：重构 auth 中间件", tone: "ok" },
        { text: "edit_file  src/auth/middleware.ts", tone: "info" },
        { text: "run_tests  go test ./...", tone: "info" },
        { text: "等待审批 · 删除 src/old.ts", tone: "warn" },
      ],
      status: "riffpad daemon · 端到端加密 ●",
      fromPhone: "⏎ 手机",
      approvedLine: "手机端已批准 · 继续执行",
      rejectedLine: "手机端已拒绝 · agent 暂停",
    },
    phone: {
      title: "riffpad",
      synced: "已同步",
      session: "s_9f2a",
      cli: "claude",
      running: "运行中",
      tools: "2 次工具调用",
      approval: "审批请求",
      summary: "删除 src/old.ts",
      approve: "同意",
      reject: "拒绝",
      approved: "已同意 · agent 继续运行",
      rejected: "已拒绝 · agent 已暂停",
      hello: "会话已附着——需要你决策时我会推送给你。",
      input: "给 agent 发消息…",
      tabs: ["会话", "会话", "设置"],
      presets: [
        {
          send: "同意删除",
          ack: "已批准——继续重构。",
          term: [
            { text: "批准通过 · 删除 src/old.ts", tone: "ok" },
            { text: "重构 auth 中间件 · 改动 3 个文件", tone: "info" },
          ],
        },
        {
          send: "先跑一遍测试",
          ack: "收到——先跑测试，再继续改动。",
          term: [
            { text: "run_tests  go test ./...", tone: "info" },
            { text: "42 通过 · 0 失败 · 1.8s", tone: "ok" },
          ],
        },
        {
          send: "改用 CSV 输出",
          ack: "明白——输出切换为 CSV。",
          term: [
            { text: "新指令 · 输出格式 → csv", tone: "info" },
            { text: "edit_file  src/report/export.ts", tone: "info" },
          ],
        },
      ],
    },
    sync: {
      label: "端到端加密",
      status: "已同步",
      syncing: "同步中…",
      latency: "84ms",
    },
  },
  terminal: {
    title: "riffpad daemon — 已附着",
    connection: "relay api.riffpad.ai · 已加密",
    session: "会话 s_9f2a · claude · ",
    running: "运行中",
    approval: "审批请求",
    approvalSummary: "删除 src/old.ts",
    approve: "同意",
    reject: "拒绝",
    statusE2ee: "端到端加密 aes-256-gcm",
    statusRelay: "中继零知识",
    statusLatency: "延迟 84ms",
    hintTab: "tab 切换会话",
    hintCmd: "ctrl-p 命令",
    hintEsc: "esc 批准",
  },
  features: {
    label: "features",
    title: "为 agent 需要你的时刻而生",
    subtitle: "长任务跑着的时候，不用再守着电脑。",
    items: [
      {
        title: "实时监督",
        description:
          "状态、工具调用、文件变更与命令都有结构化事件；CLI 不提供结构化输出时，自动降级到终端视图。",
      },
      {
        title: "随时随地审批",
        description:
          "审批请求变成推送通知。手机上同意、拒绝，或先修改条件再放行。",
      },
      {
        title: "远程转向",
        description:
          "给正在运行的会话发一条新指令，agent 在不碰电脑的情况下改变方向。",
      },
      {
        title: "安全是默认项",
        description:
          "Agent 跑在你自己的电脑上，用自己的 key 和仓库；中继只转发密文，不落盘。",
      },
    ],
  },
  how: {
    label: "how-it-works",
    title: "三跳，零云端托管",
    subtitle: "链路很短，每一跳都加密。",
    steps: [
      {
        title: "安装 daemon",
        description:
          "macOS / Linux 单二进制，包装或附着你的 CLI 会话，密钥只留在本机。",
      },
      {
        title: "手机扫码配对",
        description:
          "X25519 交换设备密钥；每个会话再派生独立的临时密钥。",
      },
      {
        title: "随时监督",
        description:
          "地铁、床上、会议间隙：看加密事件、批准、拒绝、发指令。",
      },
    ],
  },
  security: {
    label: "security",
    title: "默认零知识",
    subtitle: "中继读不到你的会话。产品从一开始就不需要它读。",
    items: [
      {
        title: "本地优先",
        description:
          "代码、仓库与 API key 不出你的电脑；daemon 只把会话桥接到手机。",
      },
      {
        title: "端到端加密",
        description:
          "X25519 密钥交换 + AES-256-GCM，密钥只存在于 daemon 和手机两端。",
      },
      {
        title: "中继只路由，不读取",
        description:
          "api.riffpad.ai 只转发密文，不保留消息历史；不写日志、不存内容。",
      },
      {
        title: "默认只读",
        description:
          "查看是被动的；批准、拒绝、暂停和发指令都是显式操作，daemon 还带一键熔断。",
      },
    ],
  },
  faq: {
    label: "faq",
    title: "常见问题",
    items: [
      {
        q: "支持哪些编程 CLI？",
        a: "目前是 Claude Code 和 Codex（结构化输出 + hooks）；DeepSeek CLI、Kimi、GLM 在路线图上，tmux/PTY 是通用兜底。",
      },
      {
        q: "代码和 API key 放在哪里？",
        a: "都在你的电脑上。Riffpad 不存储任何 API key 明文，daemon 只负责把会话转发到手机。",
      },
      {
        q: "中继能读到我的会话吗？",
        a: "不能。密钥在 daemon 与手机之间直接交换，中继只能看到密文，而且不保留任何内容。",
      },
      {
        q: "手机端默认只读吗？",
        a: "是。同意、拒绝、改条件、发指令，每一个远程动作都需要你显式点击。",
      },
      {
        q: "现在的进度如何？",
        a: "M0 闭环已验证：附着 Claude 会话 → 手机收到审批卡 → 同意 → agent 继续。MVP（M1）进行中。",
      },
    ],
  },
  cta: {
    title: "别再守在终端前面。",
    description: "抢先体验，从地铁上批准操作。",
    button: "发送邮件 hi@riffpad.ai",
    note: "不打扰，我们公开构建。",
  },
  footer: {
    tagline: "AI 编程 Agent 的口袋遥控器。",
    github: "GitHub",
    docs: "文档",
    contact: "联系",
    copyright: "© 2026 Riffpad",
    rights: "本地优先 · 端到端加密 · 零知识",
  },
  languages: {
    en: "EN",
    zh: "中",
  },
};

export const messages = { en, zh };
export type Messages = typeof en;

export const DEFAULT_LANGUAGE: Language = "en";

export function getBrowserLanguage(): Language {
  if (typeof navigator === "undefined") return DEFAULT_LANGUAGE;
  const lang = navigator.language || (navigator as Navigator & { userLanguage?: string }).userLanguage;
  return lang?.toLowerCase().startsWith("zh") ? "zh" : DEFAULT_LANGUAGE;
}
