# Riffpad 开发计划（Dev Plan）

> 最后更新：2026-08-08（按实际开发进度整理）
> 关联文档：[PRD](prd.md)（产品需求）、[TSD](tsd.md)（技术规格）、[无 tmux 注入调研](agent-injection-research.md)
> 用途：追踪 M0 → M3 的开发进度；每个里程碑完成时更新状态。

---

## 0. 状态图例

- `[ ]` 未开始
- `[~]` 进行中
- `[x]` 已完成
- `[!]` 阻塞 / 需要决策

任务均对应一个 GitHub Issue（创建后回填编号），分支命名 `feature/<n>-<slug>` / `bugfix/<n>-<slug>`，PR 合并后更新本文件。

---

## 1. 已完成（仓库奠基）

| 项 | 说明 | 提交 |
|---|---|---|
| 仓库换血 | 旧产品移至 `riffpad-legacy`，新 riffpad 从零初始化 | `4f7d417` |
| 产品定位 | PRD v0.1：AI coding agent 移动遥控器 | `f6d2d55` |
| 技术方案 | TSD v0.1：三层捕获、E2EE、无状态中继 | `f6d2d55` |
| 设计细节 | 密钥分级、中继可靠性、CLI 接入机制 | `55be1ae` ~ `fbda7f0` |
| 骨架 | daemon / relay Go 占位 + mobile PWA 占位 + protocol 目录 | `4f7d417` |

### 1.1 近期完成（2026-08，超出 M0 的交付）

| 项 | 说明 | Issue / PR |
|---|---|---|
| 无 tmux 注入调研与实现 | Claude host 控制协议、Kimi ACP、Codex app-server 三通道实测通过 | #52 #55 #56 |
| React Web 客户端 | `apps/client-beta`（Vite+React+TS），daemon/relay 内嵌同一产物 | #59 #60 |
| app.riffpad.ai 上线 | relay 内嵌 UI 经 nginx + HTTPS 对外；api.riffpad.ai 保留 API/WSS | #59 |
| 单二进制 + 一键安装 | `riffpad` 内嵌 `_daemon`；`scripts/install.sh`；GitHub Releases（v0.1.0，4 平台） | #62 #63 |
| CLI 命令完善 | `login/logout` 主命令、`update` 自更新（SHA256 校验 + 原子替换） | #64 #65 #66 |
| CLI 多语种 | i18n：`--lang` > 环境变量 > 英文兜底；zh/en 语言包 | #67 #68 |
| daemon 无感化启动 | `run/sessions/pair/attach` 懒启动（文件锁防双实例）；`riffpad setup` systemd 自启 | #57 #58 |
| Landing page | Next.js 落地页 + Vercel 部署（riffpad.ai / www），GitHub-green 视觉刷新 | #53 #70 |

---

## 2. 里程碑总览

| 里程碑 | 目标 | 状态 |
|---|---|---|
| M0 | 验证核心闭环：Claude Code 等待审批 → 手机批准 → agent 继续 | `[x]` 完成 |
| M1 | MVP：daemon 多适配器 + relay + Web 客户端，E2EE，可对外使用 | `[~]` 进行中（relay/Web 已完成，缺推送/恢复/质量项） |
| M2 | 国内落地：安卓原生壳 + 厂商推送 + 国内中继，国产 CLI | `[ ]` 未开始 |
| M3 | 分享、兜底、自部署、团队版等后续能力 | `[~]` 部分完成（6 项已完成） |

---

## 3. M0：验证核心闭环（✅ 完成）

**目标**：用 Claude Code 打通“agent 等待审批 → 事件到手机 → 一键批准 → agent 继续”的最小闭环。
**验收**：真实用户在弱网/离席场景完成一次完整审批；本地与局域网事件延迟 < 500ms。

| # | 任务 | 状态 | 验收标准 | Issue |
|---|---|---|---|---|
| M0.1 | daemon CLI 骨架：命令结构、config、日志、后台运行 | `[x]` | `riffpad daemon start/pair/status/sessions/logs/stop` 可用 | — |
| M0.2 | 事件协议 v1：`packages/protocol` 事件集 + 信封格式 | `[x]` | 事件类型与 TSD §4 一致；Go 类型 + 单测 | — |
| M0.3 | Claude Code L1 适配器（包装模式） | `[x]` | 解析 `stream-json`（system/assistant/user/result），转 Riffpad 事件 | — |
| M0.4 | Claude Code L2 hooks（附着模式）：`riffpad attach` 注入 hooks | `[x]` | 审批请求可阻塞等待外部决定；通知与工具事件进入事件流 | — |
| M0.5 | 设备配对：终端二维码 + 密钥交换 + 本地密钥存储 | `[x]` | 扫码配对成功；私钥 0600；撤销可用 | — |
| M0.6 | 本地网页端（M0 用浏览器代替 App） | `[x]` | 会话列表 + 事件流 + 审批按钮 + 文字指令注入 | — |
| M0.7 | E2EE 信封：会话密钥派生 + AES-GCM 加解密 + 单测 | `[x]` | 加解密往返测试通过；中继侧不可读 | — |
| M0.8 | M0 端到端演示（附着模式） | `[x]` | 用户自己开 Claude Code 交互会话，daemon 捕捉事件，网页端批准/拒绝生效 | — |

**M0 出口条件**：8 个任务全部完成；至少 3 个外部用户跑通一次完整闭环（外部用户验证并入 M1 种子用户）。

### M0 验证记录（2026-08-05）

| 项 | 结果 |
|---|---|
| daemon 构建/安装到 PATH、tmux 托管、优雅停止 | ✅ |
| `riffpad attach` 注入 hooks（matcher+hooks 格式），`detach` 还原 | ✅ |
| 用户消息（UserPromptSubmit）与 Agent 消息（MessageDisplay）进入时间线 | ✅ |
| 多 claude 会话按 session_id 区分显示 | ✅ |
| 配对、E2EE 握手、事件回放 | ✅ |
| **审批闭环**（PermissionRequest hook → 网页审批卡 → 同意/拒绝 → claude 继续） | ✅（日志确认 resolved allow） |

### M0 已知边界（2026-08 整理后）

- 会话状态与历史已加密持久化；daemon 重启后 Codex 会话自动重连（TUI 无感），Kimi/Claude 恢复为只读（历史可回看，agent 需手动重连）
- 控制只有 stop；pause/resume 未实现
- 审批只支持同意/拒绝，条件编辑字段预留未用
- Claude **host 托管模式**（`riffpad run --cli claude`）注入与事件已通，但工具权限审批仍被 CLI 自动拒绝（`--permission-prompt-tool` 已移除）——审批继续走附着模式 hooks 或预设 `--permission-mode`
- 附着模式指令注入依赖 tmux send-keys；终端画面兜底仍留 M3
- Kimi/Codex 托管模式审批完整（协议内 request_permission / requestApproval）

---

## 4. M1：MVP（进行中）

**目标**：可对外使用的 MVP——多适配器 daemon + 云端中继 + Web/移动端，E2EE 全链路。
**验收**：跨网络（手机蜂窝网）可用；Web MVP 完整闭环；10 个种子用户持续使用（推送由原生 app 通道在 M2.2 提供）。

### 4.1 daemon

| # | 任务 | 状态 | 验收标准 | Issue |
|---|---|---|---|---|
| M1.1 | Claude Code 适配器：host 控制协议（托管）+ 附着 hooks（attach） | `[x]` | 多轮注入、事件流、attach 审批闭环；2.1.220 实测 | #55 |
| M1.2 | Codex 适配器：`codex app-server`（thread/turn/steer + requestApproval） | `[x]` | 注入、流式事件、命令/文件/权限审批映射；真机实测 | #55 |
| M1.3 | Kimi Code 适配器：ACP client（`kimi acp`） | `[x]` | session/prompt、session/update、request_permission 全协议内；真机实测 | #55 |
| M1.4 | 会话管理：多会话 + daemon 重启恢复 | `[x]` | 多会话并行；状态/历史加密持久化（AES-GCM）；重启后 Codex 自动重连（TUI 无感），Kimi/Claude 只读恢复；`riffpad update` 自动重启并恢复 | #86 #87 #88 #89 |
| M1.5 | 断线重连 + 本地回放 | `[x]` | daemon/relay 重连 + 全量回放；客户端指数退避重连 + event.id 去重；断线期间指令/审批进入 Outbox，重连后按序补发（只补发从未写出的，避免重复执行）；列表页离线横幅；重连彻底失败引导重新配对 | #94 #95 |
| M1.6 | 设备管理：多手机、撤销、一键熔断 | `[x]` | relay 设备列表/撤销（立即断开）、host 熔断（撤销全部+断全部 viewer）；daemon `POST /api/killswitch` + CLI `riffpad kill`；web 设备卡片（撤销/熔断，二次确认）；撤销后客户端启动校验并引导重新配对 | #96 #97 |

### 4.2 relay

| # | 任务 | 状态 | 验收标准 | Issue |
|---|---|---|---|---|
| M1.7 | WebSocket Hub + 房间路由 + 心跳重连 | `[x]` | Host/Viewer 路由、join/leave、会话同步（本地 E2E 验证通过） | — |
| M1.8 | 用户 auth + 配对 API + per-host 注册密钥 | `[x]` | 用户注册/登录/登出/me；host/device 绑定 owner；daemon `riffpad login` | — |
| M1.9 | 元数据存储（SQLite，GORM；可换 Postgres） | `[x]` | users/hosts/devices/sessions 落库；relay 重启不丢 | — |
| M1.10 | 部署：VPS + nginx + TLS + 健康检查 | `[x]` | relay 上线 https://api.riffpad.ai（Let's Encrypt 自动续期）；systemd 托管 | — |

### 4.3 Web / mobile 客户端

| # | 任务 | 状态 | 验收标准 | Issue |
|---|---|---|---|---|
| M1.11 | Web 客户端：`apps/client-beta`（React/Vite/TS） | `[x]` | 本地 8787 与 app.riffpad.ai 同一产物；登录/配对/会话/审批/指令全功能 | #59 |
| M1.12 | 审批卡片 + 指令输入（文字） | `[x]` | 一键批准/拒绝（一次性按钮）；指令送达；E2EE 传输 | — |
| M1.13 | 推送（Web Push VAPID） | `[!]` | **由原生 app 推送替代（M2.2）**：Web Push 无法覆盖国内厂商推送且需服务端→浏览器通道；原生壳（M2.1）配 APNs/FCM/厂商通道是正路，跳过避免做两遍 | — |
| M1.14 | 原生移动壳（Capacitor/Expo）+ Keychain/Keystore E2EE | `[!]` | **暂缓（先 Web MVP）**：上架/备案等手续重，先用 Web 验证产品；原生壳时配 Keychain/Keystore E2EE、自动获取设备名（expo-device）并支持改名 | — |

### 4.4 质量与上线

| # | 任务 | 状态 | 验收标准 | Issue |
|---|---|---|---|---|
| M1.15 | 单元测试 + Playwright e2e（审批闭环、断线重连、E2EE） | `[x]` | CI 三 job：Go 全量测试（daemon/relay/protocol）、client-beta typecheck/build、Playwright UI 冒烟；审批闭环/E2EE 由 Go 测试与 e2e-acceptance.mjs 覆盖 | #91 #92 |
| M1.17 | MVP 体验打磨（app.riffpad.ai / 本地 8787）：onboarding、空状态、错误提示、移动端适配、配对/审批流程细节 | `[x]` | 外部用户零指导完成全流程；无明显粗糙交互；2026-08-07 经多轮 UI 打磨上线（配对引导/登录页/Dashboard/会话详情/Devices/骨架屏/主题） | #101 #106 #116 #124 #127 #128 |
| M1.18 | 账号与部署升级：GitHub OAuth 登录 + SQLite→Postgres + relay/postgres Docker Compose 容器化（app.riffpad.ai 前端继续内嵌 relay） | `[x]` | GitHub 登录可用（密码登录保留，OAuth 账号 passwordless）；relay 数据在 Postgres（`migrate-sqlite` 迁移完成）；compose 一键上线（nginx/certbot 留宿主，relay 仅监听 127.0.0.1:9090）；生产密钥在 /opt/riffpad/.env | #104 #105 |
| M1.19 | client-beta UI v2：按 design-system 重做视觉（Console-Mobile：Geist Mono / GitHub green / hairline 卡片）+ 入场/状态动画 + Web 多语种（zh/en，自动检测 + 手动切换） | `[x]` | app.riffpad.ai 与本地 8787 同一产物；深/浅色跟随系统；动画尊重 reduced-motion；中英切换即时生效；2026-08-07 经 CD 自动部署上线 | #106 #107 |
| M1.20 | 会话历史懒加载 + 分页：连接只推最近 N 条，上滑按页拉取更早历史，杜绝超长会话全量重放 | `[x]` | 连接只回放最近 100 条；WS history_query 分页读取 events.enc（limit 200）；上滑自动加载 + 滚动锚点；event.id 去重合并；加载中提示 | #125 |
| M1.21 | 静态文档站（VitePress → riffpad.ai/docs）：Quickstart / CLI 参考 / 架构 / 安全 / FAQ，随 landing 一起部署 | `[x]` | /docs 可访问（cleanUrls，Vercel 静态导出）；内容与 PRD/TSD/CLI 一致；中文版已上线 2026-08-08 | #132 #134 #135 |
| M1.16 | 种子用户招募与反馈收集 | `[ ]` | ≥ 10 个用户；留存数据可看 | — |

**M1 出口条件**：M1.1–M1.21 全部完成（M1.13 由原生推送替代、M1.14 暂缓，见上）；种子用户可自助安装 daemon（`curl -fsSL https://riffpad.ai/install.sh | sh`）并完成 Web 全流程。

### M1 验证记录（2026-08-05/06，relay 地基 + 适配器）

| 项 | 结果 |
|---|---|
| relay Hub：Host/Viewer 连接、join/leave 路由、会话同步 | ✅ 单测 + 本地实测 |
| daemon 以主机身份接入 relay，attach 会话自动广播 | ✅ |
| 端到端经 relay：配对 → 会话回放 → 实时消息 → 审批 allow | ✅ Node WebCrypto 客户端实测 |
| per-host 注册密钥、relay 重启持久化、daemon 免密钥重连、配对 IP 限流 | ✅ |
| 生产部署：api.riffpad.ai + HTTPS + 公网 E2E | ✅ 2026-08-06 实测 |
| Claude host / Kimi ACP / Codex app-server 三适配器 | ✅ 真机实测（注入 + 事件 + 审批） |
| app.riffpad.ai 公网 Web 客户端 | ✅ 2026-08-06 部署验证 |
| M1.18：GitHub OAuth + Postgres + compose | ✅ 2026-08-07 部署：systemd→compose 切换，SQLite→Postgres 迁移（users 1 / tokens 6 / hosts 1 / devices 2 / session_meta 72），GitHub 登录入口待真机验证 |

---

## 5. M2：国内落地（未开始）

**目标**：国内用户体验完整——安卓原生壳、厂商推送、国内中继、国产 CLI。
**验收**：国内用户手机全流程可用；DeepSeek / GLM 适配器可用。

| # | 任务 | 状态 | 验收标准 | Issue |
|---|---|---|---|---|
| M2.1 | 安卓原生壳（Capacitor / Expo） | `[ ]` | 上架可安装；后台保活 | — |
| M2.2 | 厂商推送（华为/小米/OPPO/vivo）+ FCM 兜底 | `[ ]` | 国内网络推送 < 5s | — |
| M2.3 | 国内中继节点（腾讯云 / 阿里云）+ 就近路由 | `[ ]` | 国内事件延迟 < 500ms（P95） | — |
| M2.4 | DeepSeek CLI 适配器 | `[ ]` | 结构化事件可用；退化到 L3 兜底 | — |
| M2.5 | GLM 适配器 | `[ ]` | 同上 | — |
| M2.6 | 隐私承诺页 + 合规（零知识说明、数据出境、ICP 相关） | `[ ]` | 法务/合规确认 | — |
| M2.7 | 付费：免费层 / Pro 定价与支付接入 | `[ ]` | 付费转化 > 5% | — |

> 注：Kimi Code 适配器已在 M1.3 完成（官方 ACP），不再列入 M2。

---

## 6. M3：后续（部分完成）

| # | 任务 | 状态 | 验收标准 | Issue |
|---|---|---|---|---|
| M3.1 | 只读会话分享链接 | `[ ]` | 实时同步；可撤销 | — |
| M3.2 | tmux / PTY 兜底（L3）正式实现 | `[ ]` | 任意 CLI 可监控 | — |
| M3.3 | 自部署中继 | `[ ]` | 一键部署文档 + 镜像 | — |
| M3.4 | 局域网直连 / WebRTC（relay 退化信令） | `[ ]` | 局域网内无中继可用 | — |
| M3.5 | Windows daemon | `[x]` | 交叉编译通过（codex Kill、daemon Flock、Setsid 拆 build tag）；CI 增加 windows/darwin 编译守卫；scripts/install.ps1 + schtasks 开机自启；release 矩阵含 windows amd64/arm64（随下个 release 发布） | #147 |
| M3.6 | 团队版（多人共享设备/会话） | `[ ]` | 权限模型可用 | — |
| M3.7 | 无 tmux 注入：Kimi ACP / Codex app-server / Claude host 控制协议 | `[x]` | 三适配器实现并真机实测；`riffpad run --cli kimi/codex/claude` | #55 #52 |
| M3.8 | daemon 无感化启动：CLI 懒启动 + Linux systemd 自启 | `[x]` | 自动拉起（文件锁防双实例）；`riffpad setup` 安装/卸载 | #57 |
| M3.9 | Web 客户端 React 重写 + app.riffpad.ai 上线 | `[x]` | Vite+React+TS；daemon/relay 内嵌同一产物；HTTPS 已部署 | #59 |
| M3.10 | 单二进制 + 一键安装 + GitHub Releases | `[x]` | `riffpad _daemon` 内嵌；install.sh + SHA256；tag v* 交叉编译 4 平台 | #62 |
| M3.11 | CLI 命令完善：`login/logout` + `update` 自更新 | `[x]` | 旧命令保留别名；update 校验/备份/原子替换 | #64 #65 |
| M3.12 | CLI 多语种（i18n） | `[x]` | zh/en 语言包；`--lang` > 环境变量 > 英文兜底 | #67 |
| M3.13 | 第三方登录：Google / Email（OAuth + 邮箱验证码） | `[ ]` | GitHub OAuth 已完成（M1.18 #105）；Google（国内可用性低）、Email（需 SMTP）暂缓 | — |
| M3.14 | 托管模式无感化：`riffpad run` 后电脑终端照常显示/操作 CLI TUI，手机同步遥控 | `[~]` | Codex：`--remote` 共享 app-server 已完成（#74 #79）；Claude：daemon PTY + hooks 收编，前台 TUI 转发待做（#210）；Kimi：后续；Ctrl-C 退出即退出，持久会话由用户 tmux（docs 强烈推荐） | #74 #79 |
| M3.15 | 容器化部署：relay + Postgres 容器化 | `[x]` | 已随 M1.18 完成：relay 仅监听 127.0.0.1:9090、Postgres 17、healthcheck/restart、nginx/certbot 留宿主；生产运行中 | — |
| M3.16 | 元数据存储 SQLite → Postgres 迁移 | `[x]` | 已随 M1.18 完成：`apps/relay/cmd/migrate-sqlite` 逐表迁移 + 行数校验，生产已切换 Postgres | — |
| M3.17 | CI/CD：main push → 自动部署 relay（GitHub Actions + SSH 受限部署密钥） | `[x]` | CI 三 job 通过后自动执行 `/usr/local/bin/riffpad-deploy.sh`（git pull --ff-only + compose up -d --build + 健康检查）；部署密钥仅能执行固定脚本；2026-08-07 已端到端验证 | #108 #112 #113 |
| M3.18 | 会话按主机组织：relay 返回 hostName，client 会话列表按主机分组/标注 | `[ ]` | 多台电脑会话可区分；分组展示；同步确认设备访问主机策略 | #151 |
| M3.19 | CLI 登录页（/device）与授权回执页 UI 优化：对齐登录页质感、深色主题卡片、中英文随 lang 渲染 | `[x]` | device 页终端头部 + 授权码 + GitHub 按钮；回执页深色卡片 + 绿色状态；单测覆盖 en 回执 | #160 |
| M3.20 | 本地模式配对体验与 LAN 自托管：`riffpad pair` 默认要求登录，`--local` 才生成仅本机可用的配对码；后续支持同一局域网内自托管（relay 退化/直连），手机可直接配对本机 daemon | `[~]` | pair 登录引导已实现（#206）；LAN 直连待 M3.4 排期 | #206 |
| M3.21 | macOS daemon 管理：`riffpad setup` 支持 launchd LaunchAgent（登录自启 + 崩溃重启）；`riffpad daemon restart` 与 `riffpad update` 自动重启在 macOS 优先走 `launchctl kickstart -k`，保持 launchd 为唯一进程账本 | `[ ]` | setup 安装/移除 LaunchAgent；restart/update 检测 launchd 托管并走 launchctl；真机验证 | — |
| M3.22 | Windows daemon 管理：`riffpad daemon restart` 与 `riffpad update` 自动重启检测 `RiffpadDaemon` 计划任务，停止旧进程后 `schtasks /Run` 由任务管理器拉起，保持任务账本一致 | `[x]` | daemonRestart 任务分支 + 单测（运行中停旧启新 / 未运行直接拉起）已实现 | #208 |
| M3.23 | 弃用 attach 模式：入口关闭（代码保留，`riffpad detach` 仍可用于清理旧 hooks），统一 `riffpad run` 托管模式；`riffpad run` 支持位置参数（`riffpad run codex`） | `[x]` | attachCmd 返回弃用提示；run 位置参数 + 单测已实现 | #214 |

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
| Claude host 模式审批受限 | 审批继续走附着模式 hooks 或预设 permission-mode；持续跟踪官方控制协议 |
| 官方移动端竞争 | 聚焦跨 CLI + 本地桥接 + 国内体验 |
| E2EE 实现错误 | 独立安全 review + 已知答案测试向量 |
| 种子用户不足 | M1 出口前开始招募，优先已有 AI CLI 用户群 |
