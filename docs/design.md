# Riffpad 设计说明（v0.1）

> 本文是快速总览；产品需求见 [PRD](prd.md)，技术规格见 [TSD](tsd.md)。

## 1. 定位

AI coding agent（Claude Code、Codex、DeepSeek CLI、Kimi CLI 等）会长时间自主工作，用户不该守在电脑前。Riffpad 提供：

- **无感托管**：daemon 接管会话后，电脑终端照常显示并可操作 coding agent 的 TUI，用户感受不到差异；手机是离席时的镜像与遥控器
- **监督**：手机上实时查看 agent 的进度、工具调用、文件变更
- **审批**：agent 等待确认时推送通知，手机一键同意/拒绝/修改条件
- **转向**：文字下达新指令，远程影响本地会话
- **隐私**：agent 跑在用户自己的电脑上，云端只做加密中继，不落盘、不解密

## 2. 架构

```
┌─────────────────┐      ┌─────────────────────┐      ┌──────────────┐
│  用户电脑        │      │  云端中继 (relay)    │      │  手机 (mobile)│
│  daemon (Go)    │──WSS─▶│  WebSocket + E2EE   │──WSS─▶│  会话列表      │
│  适配器层        │      │  不落盘 / 不记录     │      │  审批卡片      │
│  tmux/PTY 兜底  │      └─────────────────────┘      │  指令输入      │
└─────────────────┘                                    └──────────────┘
```

### daemon（用户电脑）

- 主路径：按 CLI 适配器订阅结构化事件（如 Claude Code 的 `stream-json`、Codex 的 JSON 输出），渲染为结构化事件
- 托管模式：daemon 用 PTY/共享会话运行 CLI，并把 TUI 透传到当前终端（本地输入照常）；不同 CLI 的接入路径差异由适配器层吸收（Claude = PTY + hooks、Codex = `--remote` 共享 app-server、Kimi = 后续渲染）
- 兜底路径：挂接 tmux 控制模式（`tmux -CC`）或 PTY，原始字节流供移动端 xterm.js 渲染
- 设备配对：二维码扫码交换密钥，之后只接受已配对设备

### relay（云端）

- WebSocket 房间 + 事件转发，中继自身不解密内容（X25519 密钥交换 + AES-GCM）
- 不持久化会话内容；仅保存非敏感元数据（设备、在线状态）
- 国内部署时提供就近节点，支持国内推送通道

### mobile

- PWA 起步（复用熟悉的 Next.js），终端渲染用 xterm.js
- 国内生产环境需要原生壳接入厂商推送（华为/小米/OPPO/vivo）
- 默认只读；审批和指令输入为显式用户动作

## 3. 事件协议（草案）

见 `packages/protocol`。核心事件：

| 事件 | 方向 | 说明 |
|---|---|---|
| `session_start` / `session_end` | daemon → mobile | 会话生命周期 |
| `agent_status` | daemon → mobile | `running` / `waiting_input` / `done` / `error` |
| `tool_call` | daemon → mobile | 工具名、参数、状态 |
| `approval_request` | daemon → mobile | 审批请求（含操作摘要） |
| `approval_response` | mobile → daemon | `approve` / `reject` / 修改后的条件 |
| `notify` | daemon → mobile | 通知；审批过期回显带 `requestId`（level=error），供 client 纠正卡片状态 |
| `prompt` | mobile → daemon | 文字新指令 |
| `file_change` | daemon → mobile | 路径与变更摘要 |

每个事件带会话内递增的 `seq`（daemon 在事件泵处赋值，0 表示无序号）。client 检测到 seq 空洞时打警告日志并重连，靠重连回放补洞（#173）。

**缓冲丢弃策略**（#173）：各跳发送缓冲（256 条）满时——

- 关键事件（`approval_request` / `approval_response` / `session_end`）一律不丢：关闭该连接，强迫 client 重连并回放历史；
- relay 无法解密信封、无法区分事件类型，因此 relay 侧任何缓冲溢出都直接关闭对应连接（host 或 viewer）；
- 其余非关键事件丢弃时必须打 warn 日志（带 session id 与事件类型）。

daemon ↔ relay 之间另有一组不加密的路由控制帧（`/ws/host` 连接，kind 字段）：

| 帧 | 方向 | 说明 |
|---|---|---|
| `sessions` | daemon → relay | 上报本 host 的会话列表；relay 只替换该 host 的条目，不影响同账号其他 host |
| `join` / `leave` / `viewer` | relay ↔ daemon | viewer 接入、离开与加密信封转发 |
| `superseded` | relay → daemon | 同一 host 凭据在别处新连接，本连接被顶替；daemon 收到后停止自动重连并日志提示（防双 daemon 互踢），重启 daemon 可重试 |
| `kick` | daemon → relay | daemon 丢弃某 viewer 后要求 relay 关闭其浏览器连接（如关键事件缓冲溢出），client 重连后获得历史回放（#173） |

## 4. 安全基线

- 传输全程 E2EE，relay 无解密能力
- daemon 不存储任何 API key 的明文副本，只透传本地环境
- 移动端默认只读；远程执行指令前必须显式授权
- 提供一键熔断（daemon 关闭所有会话/撤回配对）

## 5. MVP 范围

1. daemon：Claude Code `stream-json` 适配器 + 会话管理
2. relay：WebSocket 转发 + 配对 + E2EE
3. mobile：会话列表 + `approval_request` 推送 + 一键同意/拒绝
4. 端到端验证：电脑上 agent 等待审批 → 手机收到通知 → 手机批准 → agent 继续

## 6. 非目标（当前阶段）

- 不做云端沙箱 / 云端 IDE
- 不做通用终端镜像的复杂键盘操作（读优先、控制精简）
- 不做 H5 生成、文档、游戏等与遥控器无关的形态
