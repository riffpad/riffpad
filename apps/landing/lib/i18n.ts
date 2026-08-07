export type Language = "en" | "zh";

const en = {
  hero: {
    title1: "Your AI coding agents,",
    title2: "in your pocket.",
    description:
      "Riffpad bridges Claude Code, Codex and other CLI agents to your phone — watch progress, approve actions and steer sessions without staying chained to the desk.",
    ctaPrimary: "Get Started",
    ctaSecondary: "Read the docs",
    note: "local daemon · end-to-end encrypted · zero-knowledge relay",
  },
  install: {
    unix: "macOS / Linux",
    windows: "Windows",
    copy: "copy",
    copied: "copied ✓",
  },
  architecture: {
    label: "architecture",
    title: "One daemon, three pipes.",
    description:
      "The CLI agents stay on your computer. The daemon captures their stream, the relay forwards encrypted envelopes, and your phone drives approvals and steering.",
    clientTitle: "client",
    clientCaption: "your phone",
    relayTitle: "relay",
    relayCaption: "zero-knowledge",
    hostTitle: "daemon",
    hostCaption: "your computer",
    flowEvents: "events",
    flowCommands: "approvals",
    moreClis: "and more…",
  },
  mockup: {
    hint: "Live demo — send a message from the phone and watch the Mac react.",
    mac: {
      lines: [
        { text: "Refactor the auth middleware to use the new session store", tone: "user" },
        { text: "Thought for 4s, read 3 files, listed 1 directory (ctrl+o to expand)", tone: "think" },
        { text: "Edit(src/auth/middleware.ts)", tone: "tool" },
        { text: "+42 −18 lines (ctrl+o to expand)", tone: "sub" },
        { text: "Edit(src/auth/session.ts)", tone: "tool" },
        { text: "+27 −9 lines (ctrl+o to expand)", tone: "sub" },
        { text: "Bash(npm test)", tone: "tool" },
        { text: "42 passed · 0 failed · 1.8s", tone: "sub" },
        { text: "src/old.ts is unused now — I'll delete it.", tone: "agent" },
        { text: "Bash(rm src/old.ts)", tone: "tool" },
        { text: "Waiting…", tone: "sub" },
      ],
      approvalTitle: "Bash command",
      approvalCmd: "rm src/old.ts",
      approvalDesc: "Delete the unused file",
      approvalQuestion: "Do you want to proceed?",
      approvalOpt1: "1. Yes",
      approvalOpt2: "2. Yes, always allow deletes in src/",
      approvalOpt3: "3. No",
      approvalFooter: "Esc to cancel · Tab to amend · or decide from your phone",
      fromPhone: "⏎ phone",
      approvedLine: "approved from phone · rm src/old.ts",
      rejectedLine: "rejected from phone · agent paused",
    },
    phone: {
      title: "riffpad",
      synced: "synced",
      session: "claude · api · 37263d",
      badges: ["cwd: ~/projects/api", "cli: claude"],
      agentMsg1: "Middleware switched to the new session store — 42 tests green.",
      tools: ["Edit src/auth/middleware.ts", "Edit src/auth/session.ts", "$ npm test"],
      agentMsg2: "src/old.ts is unused now — I'll delete it.",
      pendingTool: "$ rm src/old.ts",
      approval: "approval request",
      summary: "Bash：rm src/old.ts",
      approve: "Approve",
      reject: "Reject",
      approved: "Approved",
      rejected: "Rejected",
      input: "Send a message…",
      presets: [
        {
          send: "Approve the deletion",
          ack: "Approved — continuing the refactor.",
          term: [
            { text: "approval granted · rm src/old.ts", tone: "ok" },
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
          send: "Keep the file for now",
          ack: "Got it — keeping src/old.ts, nothing deleted.",
          term: [
            { text: "new instruction · keep src/old.ts", tone: "info" },
            { text: "refactor continues · no files deleted", tone: "ok" },
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
  how: {
    label: "how-it-works",
    title: "Three hops. Zero cloud custody.",
    subtitle: "The loop is short, and every hop is encrypted.",
    steps: [
      {
        title: "Install the daemon",
        description:
          "One binary with adaptors for the coding CLIs you already use. It wraps or attaches to your sessions and owns the keys locally.",
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
    flow: {
      daemonLabel: "daemon · plaintext",
      clientLabel: "client · plaintext",
      plainLines: ["Bash(rm src/old.ts)", "42 passed · 0 failed · 1.8s"],
      encryptLabel: "encrypt · x25519 + aes-256-gcm",
      decryptLabel: "decrypt · keys never leave devices",
      relayLabel: "relay · zero-knowledge",
      relayNote: "ciphertext only",
    },
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
    description: "Open the app, pair your daemon, and approve from the subway instead of the desk chair.",
    button: "Get Started",
    note: "GitHub sign-in · end-to-end encrypted · zero-knowledge relay",
  },
  footer: {
    tagline: "The pocket remote for your AI coding agents.",
    github: "GitHub",
    docs: "Docs",
    discord: "Discord",
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
    ctaPrimary: "开始使用",
    ctaSecondary: "阅读文档",
    note: "本地 daemon · 端到端加密 · 零知识中继",
  },
  install: {
    unix: "macOS / Linux",
    windows: "Windows",
    copy: "复制",
    copied: "已复制 ✓",
  },
  architecture: {
    label: "系统架构",
    title: "一个 daemon，三条通道。",
    description:
      "CLI agent 留在你的电脑上：daemon 捕获事件流，relay 转发加密信封，手机负责审批与转向。",
    clientTitle: "client",
    clientCaption: "你的手机",
    relayTitle: "relay",
    relayCaption: "零知识",
    hostTitle: "daemon",
    hostCaption: "你的电脑",
    flowEvents: "事件",
    flowCommands: "审批",
    moreClis: "等更多…",
  },
  mockup: {
    hint: "可交互演示：在手机上发一条消息，看 Mac 终端实时响应。",
    mac: {
      lines: [
        { text: "把 auth 中间件迁移到新的 session store", tone: "user" },
        { text: "Thought for 4s, read 3 files, listed 1 directory (ctrl+o to expand)", tone: "think" },
        { text: "Edit(src/auth/middleware.ts)", tone: "tool" },
        { text: "+42 −18 lines (ctrl+o to expand)", tone: "sub" },
        { text: "Edit(src/auth/session.ts)", tone: "tool" },
        { text: "+27 −9 lines (ctrl+o to expand)", tone: "sub" },
        { text: "Bash(npm test)", tone: "tool" },
        { text: "42 passed · 0 failed · 1.8s", tone: "sub" },
        { text: "src/old.ts 已经不再使用——我准备把它删掉。", tone: "agent" },
        { text: "Bash(rm src/old.ts)", tone: "tool" },
        { text: "Waiting…", tone: "sub" },
      ],
      approvalTitle: "Bash command",
      approvalCmd: "rm src/old.ts",
      approvalDesc: "删除不再使用的文件",
      approvalQuestion: "Do you want to proceed?",
      approvalOpt1: "1. Yes",
      approvalOpt2: "2. Yes, always allow deletes in src/",
      approvalOpt3: "3. No",
      approvalFooter: "Esc 取消 · Tab 修改 · 或在手机上决定",
      fromPhone: "⏎ 手机",
      approvedLine: "手机端已批准 · rm src/old.ts",
      rejectedLine: "手机端已拒绝 · agent 暂停",
    },
    phone: {
      title: "riffpad",
      synced: "已同步",
      session: "claude · api · 37263d",
      badges: ["cwd: ~/projects/api", "cli: claude"],
      agentMsg1: "中间件已迁移到新的 session store——42 个测试全绿。",
      tools: ["Edit src/auth/middleware.ts", "Edit src/auth/session.ts", "$ npm test"],
      agentMsg2: "src/old.ts 已经不再使用——我准备把它删掉。",
      pendingTool: "$ rm src/old.ts",
      approval: "审批请求",
      summary: "Bash：rm src/old.ts",
      approve: "同意",
      reject: "拒绝",
      approved: "已同意",
      rejected: "已拒绝",
      input: "发送消息…",
      presets: [
        {
          send: "同意删除",
          ack: "已批准——继续重构。",
          term: [
            { text: "批准通过 · rm src/old.ts", tone: "ok" },
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
          send: "先留着这个文件",
          ack: "好的——保留 src/old.ts，不删除。",
          term: [
            { text: "新指令 · 保留 src/old.ts", tone: "info" },
            { text: "重构继续 · 未删除文件", tone: "ok" },
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
  how: {
    label: "如何工作",
    title: "三跳，零云端托管",
    subtitle: "链路很短，每一跳都加密。",
    steps: [
      {
        title: "安装 daemon",
        description:
          "单二进制，内置各主流 coding CLI 的适配器；包装或附着你的会话，密钥只留在本机。",
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
    label: "安全",
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
    flow: {
      daemonLabel: "daemon · 明文",
      clientLabel: "client · 明文",
      plainLines: ["Bash(rm src/old.ts)", "42 passed · 0 failed · 1.8s"],
      encryptLabel: "加密 · x25519 + aes-256-gcm",
      decryptLabel: "解密 · 密钥不出设备",
      relayLabel: "relay · 零知识",
      relayNote: "只有密文",
    },
  },
  faq: {
    label: "问答",
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
    description: "打开 App、配对 daemon，从地铁上批准操作。",
    button: "开始使用",
    note: "GitHub 登录 · 端到端加密 · 零知识中继",
  },
  footer: {
    tagline: "AI 编程 Agent 的口袋遥控器。",
    github: "GitHub",
    docs: "文档",
    discord: "Discord",
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
