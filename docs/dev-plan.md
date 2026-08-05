# Riffpad 开发计划（Dev Plan）

> 最后更新：2026-08-04
> 关联文档：[PRD](prd.md)（产品需求）、[TSD](tsd.md)（技术规格）
> 用途：追踪 M0 → M3 的开发进度；每个里程碑完成时更新状态。

---

## 0. 状态图例

- `[ ]` 未开始
- `[~]` 进行中
- `[x]` 已完成
- `[!]` 阻塞 / 需要决策

任务均对应一个 GitHub Issue（创建后回填编号），分支命名 `feature/<n>-<slug>`，PR 合并后更新本文件。

---

## 1. 已完成（仓库奠基）

| 项 | 说明 | 提交 |
|---|---|---|
| 仓库换血 | 旧产品移至 `riffpad-legacy`，新 riffpad 从零初始化 | `4f7d417` |
| 产品定位 | PRD v0.1：AI coding agent 移动遥控器 | `f6d2d55` |
| 技术方案 | TSD v0.1：三层捕获、E2EE、无状态中继 | `f6d2d55` |
| 设计细节 | 密钥分级、中继可靠性、CLI 接入机制 | `55be1ae` ~ `fbda7f0` |
| 骨架 | daemon / relay Go 占位 + mobile PWA 占位 + protocol 目录 | `4f7d417` |

---

## 2. 里程碑总览

| 里程碑 | 目标 | 状态 |
|---|---|---|
| M0 | 验证核心闭环：Claude Code 等待审批 → 手机批准 → agent 继续 | `[ ]` |
| M1 | MVP：daemon + relay + PWA，E2EE，Claude Code / Codex | `[ ]` |
| M2 | 国内落地：安卓原生壳 + 厂商推送 + 国内中继，国产 CLI | `[ ]` |
| M3 | 会话分享、tmux 兜底、自部署中继、团队版 | `[ ]` |

---

## 3. M0：验证核心闭环

**目标**：用 Claude Code 打通“agent 等待审批 → 事件到手机 → 一键批准 → agent 继续”的最小闭环。
**验收**：真实用户在弱网/离席场景完成一次完整审批；本地与局域网事件延迟 < 500ms。

| # | 任务 | 状态 | 验收标准 | Issue |
|---|---|---|---|---|
| M0.1 | daemon CLI 骨架：`riffpad` / `riffpadd` 命令结构、config、日志、后台运行 | `[x]` | `riffpad daemon start/pair/status/sessions/logs/stop` 可用 | — |
| M0.2 | 事件协议 v1：`packages/protocol` 定义事件集 + 信封格式 | `[x]` | 事件类型与 TSD §4 一致；Go 类型 + 单测 | — |
| M0.3 | Claude Code L1 适配器（包装模式） | `[x]` | 解析 `stream-json`（system/assistant/user/result），转 Riffpad 事件 | — |
| M0.4 | Claude Code L2 hooks（附着模式）：`riffpad attach` 注入 hooks，PermissionRequest/Notification/PreToolUse/PostToolUse → daemon | `[x]` | 审批请求可阻塞等待外部决定；通知与工具事件进入事件流 | — |
| M0.5 | 设备配对：终端二维码 + 密钥交换 + 本地密钥存储 | `[x]` | 扫码配对成功；私钥 0600；撤销可用 | — |
| M0.6 | 本地网页端（M0 用浏览器代替 App） | `[x]` | 会话列表 + 事件流 + 审批按钮 + 文字指令注入 | — |
| M0.7 | E2EE 信封：会话密钥派生 + AES-GCM 加解密 + 单测 | `[x]` | 加解密往返测试通过；中继侧不可读 | — |
| M0.8 | M0 端到端演示（附着模式） | `[x]` | 用户自己在终端开 Claude Code 交互会话，daemon 捕捉事件，网页端批准/拒绝生效（日志确认 hook resolved allow） | — |

**M0 出口条件**：8 个任务全部完成；至少 3 个外部用户跑通一次完整闭环。

### M0 验证记录（2026-08-05）

| 项 | 结果 |
|---|---|
| daemon 构建/安装到 PATH、tmux 托管、优雅停止 | ✅ |
| `riffpad attach` 注入 hooks（matcher+hooks 格式），`detach` 还原 | ✅ |
| 用户消息（UserPromptSubmit）与 Agent 消息（MessageDisplay）进入时间线 | ✅ |
| 多 claude 会话按 session_id 区分显示 | ✅ |
| 配对、E2EE 握手、事件回放 | ✅ |
| **审批闭环**（PermissionRequest hook → 网页审批卡 → 同意/拒绝 → claude 继续） | ✅（2026-08-05，日志确认 resolved allow；Claude 显示 Allowed by PermissionRequest hook） |
| 3 个外部用户跑通完整闭环 | ⏳ 并入 M1 种子用户 |

### M0 人工验证步骤（M0.8，附着模式）

前置：本机已安装并登录 Claude Code（`claude --version`，当前 2.1.220）。

1. `make build-daemon`
2. `./apps/daemon/bin/riffpad daemon start`
3. `./apps/daemon/bin/riffpad attach`（向 `~/.claude/settings.json` 注入 hooks，自动备份）
4. 另开一个终端（建议 tmux），正常运行 `claude`，让它做一个需要权限的操作（写文件 / 执行命令）
5. 浏览器打开 http://127.0.0.1:8787，配对后会话列表出现你正在用的 Claude 会话，事件卡片流动
6. 出现审批卡片 → 点击同意/拒绝 → 终端里的 Claude 继续或被拒绝
7. 验证完执行 `./apps/daemon/bin/riffpad detach` 还原 settings

> 若 hooks 未触发：查看 `~/.config/riffpad/logs/daemon.log`，并确认 `claude` 是在交互模式（非 `-p`）下运行。

### M0 预期行为清单

**会话状态**

| 状态 | 含义 | 触发 |
|---|---|---|
| `waiting_input` | 会话已建、claude 未启动，等待第一条指令 | 创建会话且初始指令为空 |
| `running` | claude 进程存活并工作 | 收到指令并成功启动 / agent 正在运行 |
| `done` | 会话结束 | agent 返回 `result`，或进程退出被清理 |
| `error` | 会话异常结束 | agent 返回 error result |

**网页端行为**

1. 打开页面：右上角显示“服务在线”（每 5 秒探测 daemon）
2. 未配对：显示配对页；`riffpad pair` 后输入配对码完成配对
3. 已配对：显示会话列表；新建会话（初始指令可空）
4. 点击会话进入详情：右上角依次显示 连接中… → 已连接（加密）；最近事件回放出现
5. 事件以卡片呈现：会话开始、状态、Agent 消息、工具调用、文件变更、命令、通知、审批、会话结束
6. 审批卡片出现 → 点同意/拒绝 → agent 继续 → 新事件出现
7. 底部输入指令 → 发送 → agent 响应；未连接时提示“未连接，无法发送：请刷新页面并重新打开会话”
8. API 限流：出现“API 限流（rate_limit），重试 n/10…”通知卡片，恢复后 agent 继续
9. 点“停止”：claude 被杀，出现会话结束事件
10. daemon 重启：连接断开提示；刷新后列表只剩新 daemon 内存中的会话（M0 无持久化）
11. 附着模式：`riffpad attach` 后，用户照常开自己的 claude 交互会话；SessionStart hook 创建会话，PermissionRequest hook 出审批卡，SessionEnd 结束会话

**CLI 行为**

- `riffpad daemon start`：启动后台 daemon，打印“daemon started at …”
- `riffpad daemon stop`：优雅停止，打印“daemon stopped”
- `riffpad status`：输出端口、startedAt、会话数
- `riffpad pair`：打印 6 位配对码 + 二维码
- `riffpad run --prompt "…"`：创建会话并打印 id 与网页 URL
- `riffpad logs`：输出 daemon 日志尾部

**M0 已知边界（非缺陷）**

- 会话在内存中，daemon 重启即丢（M1 引入持久化）
- 控制只有 stop；pause/resume 未实现
- 审批只支持同意/拒绝，条件编辑字段预留未用
- 网页端仅限本机访问；手机远程需要 M1 的 relay
- 包装模式（网页端开会话）在 Claude 2.1.220 下无法拦截权限（`--permission-prompt-tool` 已移除，`-p` 下 hooks 不触发），审批走附着模式
- 附着模式指令注入经 tmux send-keys 已支持（需要把 claude 放进 tmux）；终端画面仍留 M3

---

## 4. M1：MVP

**目标**：可对外使用的 MVP——daemon + 云端中继 + 手机 PWA，支持 Claude Code 与 Codex，E2EE 全链路。
**验收**：跨网络（手机蜂窝网）可用；推送 < 5s；10 个种子用户持续使用。

### 4.1 daemon

| # | 任务 | 状态 | 验收标准 | Issue |
|---|---|---|---|---|
| M1.1 | 会话管理：启动/跟踪/恢复/多会话 | `[ ]` | daemon 重启后会话可恢复；多会话并行 | — |
| M1.2 | Codex L1 适配器（`codex exec --json`） | `[ ]` | 事件流正确映射到协议事件 | — |
| M1.3 | Codex L2 hooks（PermissionRequest 等） | `[ ]` | 审批可经 hook 转发；信任流程走通 | — |
| M1.4 | 断线重连 + 本地回放 | `[ ]` | 手机断线重连后向 daemon 补齐缺口 | — |
| M1.5 | 设备管理：多手机、撤销、一键熔断 | `[ ]` | 撤销后 token 立即失效；熔断关闭全部连接 | — |

### 4.2 relay

| # | 任务 | 状态 | 验收标准 | Issue |
|---|---|---|---|---|
| M1.6 | WebSocket Hub + 房间路由 + 心跳重连 | `[x]` | Host/Viewer 路由、join/leave、会话同步（本地 E2E 验证通过） | — |
| M1.7 | 用户 auth + 配对 API + per-host 注册密钥 | `[x]` | 用户注册/登录/登出/me；host/device 绑定 owner；daemon `riffpad relay login` | — |
| M1.8 | 元数据存储（SQLite，GORM；可换 Postgres） | `[x]` | users/hosts/devices/sessions 落库；relay 重启不丢 | — |
| M1.9 | 部署：VPS + nginx + TLS + 健康检查 | `[x]` | relay 已上线 https://api.riffpad.ai（Let's Encrypt 自动续期）；systemd 托管 | — |

### 4.3 mobile（PWA）

| # | 任务 | 状态 | 验收标准 | Issue |
|---|---|---|---|---|
| M1.10 | PWA 骨架：Next.js + 会话列表 + 会话详情 | `[ ]` | 手机竖屏可用 | — |
| M1.11 | 审批卡片 + 指令输入（文字） | `[ ]` | 一键批准/拒绝/编辑条件；指令送达 | — |
| M1.12 | Web Push（VAPID）：等待审批/完成/出错 | `[ ]` | 推送 < 5s；payload 零敏感内容 | — |
| M1.13 | E2EE 客户端实现（密钥存储 + 加解密） | `[ ]` | 与 daemon 互通；私钥存 Keychain/Keystore | — |

### 4.4 质量与上线

| # | 任务 | 状态 | 验收标准 | Issue |
|---|---|---|---|---|
| M1.14 | 单元测试 + Playwright e2e（审批闭环、断线重连、E2EE） | `[ ]` | CI 全绿 | — |
| M1.15 | 种子用户招募与反馈收集 | `[ ]` | ≥ 10 个用户；留存数据可看 | — |

**M1 出口条件**：M1.1–M1.15 全部完成；种子用户可自助安装 daemon 并完成全流程。

### M1 验证记录（2026-08-05，relay 地基）

| 项 | 结果 |
|---|---|
| relay Hub：Host/Viewer 连接、join/leave 路由、会话同步 | ✅ 单测 + 本地实测 |
| daemon 以主机身份接入 relay，attach 会话自动广播 | ✅ |
| relay 配对：daemon 转发创建码，网页在 relay 端认领 | ✅ |
| 端到端经 relay：配对 → 会话回放 → 实时消息 → 审批 allow | ✅ Node WebCrypto 客户端实测 |
| per-host 注册密钥、relay 重启持久化、daemon 免密钥重连、配对 IP 限流 | ✅ 单测 + 本地实测 |
| 用户账号（注册/登录/登出）、owner 隔离、SQLite 元数据 | ✅ 单测 + 本地实测 |
| 生产部署：api.riffpad.ai + HTTPS + 公网 E2E（配对→回放→审批） | ✅ 2026-08-06 实测 |

---

## 5. M2：国内落地

**目标**：国内用户体验完整——安卓原生壳、厂商推送、国内中继、国产 CLI。
**验收**：国内用户手机全流程可用；DeepSeek / Kimi / GLM 适配器可用。

| # | 任务 | 状态 | 验收标准 | Issue |
|---|---|---|---|---|
| M2.1 | 安卓原生壳（Capacitor / Expo） | `[ ]` | 上架可安装；后台保活 | — |
| M2.2 | 厂商推送（华为/小米/OPPO/vivo）+ FCM 兜底 | `[ ]` | 国内网络推送 < 5s | — |
| M2.3 | 国内中继节点（腾讯云 / 阿里云）+ 就近路由 | `[ ]` | 国内事件延迟 < 500ms（P95） | — |
| M2.4 | DeepSeek CLI 适配器 | `[ ]` | 结构化事件可用；退化到 L3 兜底 | — |
| M2.5 | Kimi / GLM 适配器 | `[ ]` | 同上 | — |
| M2.6 | 隐私承诺页 + 合规（零知识说明、数据出境、ICP 相关） | `[ ]` | 法务/合规确认 | — |
| M2.7 | 付费：免费层 / Pro 定价与支付接入 | `[ ]` | 付费转化 > 5% | — |

---

## 6. M3：后续

| # | 任务 | 状态 | 验收标准 | Issue |
|---|---|---|---|---|
| M3.1 | 只读会话分享链接 | `[ ]` | 实时同步；可撤销 | — |
| M3.2 | tmux / PTY 兜底（L3）正式实现 | `[ ]` | 任意 CLI 可监控 | — |
| M3.3 | 自部署中继 | `[ ]` | 一键部署文档 + 镜像 | — |
| M3.4 | 局域网直连 / WebRTC（relay 退化信令） | `[ ]` | 局域网内无中继可用 | — |
| M3.5 | Windows daemon | `[ ]` | 安装包 + 自启动 | — |
| M3.6 | 团队版（多人共享设备/会话） | `[ ]` | 权限模型可用 | — |
| M3.7 | 无 tmux 注入：Kimi ACP / Codex app-server / Claude host 控制协议 | `[x]` | 三个适配器已实现并真机实测（回复/事件流/审批协议均通）：`riffpad run --cli kimi/codex/claude` | #55 #52 |
| M3.8 | daemon 无感化启动：CLI 懒启动 + Linux systemd 自启 | `[x]` | `run/sessions/pair/attach` 自动拉起 daemon（文件锁防双实例）；`riffpad setup` 安装 systemd user service，`--remove` 卸载 | #57 |
| M3.9 | Web 客户端 React 重写（apps/client-beta）+ app.riffpad.ai 上线 | `[x]` | Vite+React+TS；daemon/relay 内嵌同一产物；app.riffpad.ai HTTPS 已部署 | #59 |

---

## 7. 更新规则

1. 更新本文件：勾选完成项、标注进行中、记录阻塞
2. 任务完成时在 Issue 关闭并回填编号，PR 合并后更新对应行
3. 技术方案变化先改 TSD，再改本计划
4. 阻塞项用 `[!]` 标注并写明原因与需要的决策

## 8. 主要风险

| 风险 | 应对 |
|---|---|
| CLI 输出格式变化 | 适配器 pin 版本；协议层稳定；可降级 L3 |
| hooks 信任流程增加用户摩擦 | M0 就验证 Codex/Claude 的信任交互，写入 onboarding |
| 官方移动端竞争 | 聚焦跨 CLI + 本地桥接 + 国内体验 |
| E2EE 实现错误 | 独立安全 review + 已知答案测试向量 |
| 种子用户不足 | M0 结束前开始招募，优先已有 AI CLI 用户群 |
