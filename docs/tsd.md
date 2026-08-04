# Riffpad 技术规格说明书（TSD）

> 版本：v0.1（2026-08-04）
> 基于：[PRD](prd.md)
> 关联：[设计总览](design.md)

---

## 1. 概述

### 1.1 文档目标

将 PRD 转化为可落地的技术方案，覆盖：技术栈、系统架构、事件协议、daemon / relay / mobile 设计、数据模型、接口、安全、部署与里程碑。

### 1.2 设计原则

1. **移动优先**：手机是主交互端，daemon 是后台进程
2. **零知识中继**：云端不掌握内容，只转发加密信封
3. **本地优先**：agent、密钥、仓库全部留在用户电脑
4. **协议先行**：先定事件协议，再实现三个端
5. **最小依赖**：daemon 单二进制分发，移动端 PWA 起步

---

## 2. 技术栈

| 层 | 技术 | 说明 |
|---|---|---|
| daemon | Go 1.25 | 单二进制，跨平台（macOS / Linux 一期；Windows P1） |
| relay | Go + Echo + gorilla/websocket | 长连接中继，部署为长运行容器 |
| mobile | Next.js 14 PWA + xterm.js | 会话列表、审批卡片、终端兜底视图 |
| 推送 | Web Push（VAPID）MVP | 国内厂商通道在原生壳阶段接入 |
| 数据库 | Postgres | 仅存元数据（用户 / 设备 / 会话状态），内容不落库 |
| 部署 | Fly.io / Railway 一期 | 国内节点二期（腾讯云 / 阿里云） |

---

## 3. 系统架构

### 3.1 架构图

```
┌────────────────────┐      ┌─────────────────────────┐      ┌──────────────────┐
│ 用户电脑 (daemon)   │      │ 云端中继 (relay)         │      │ 手机 (mobile)     │
│                    │ WSS  │                         │ WSS  │                  │
│ 适配器层           │─────▶│ WebSocket Hub           │─────▶│ 会话列表/详情     │
│  Claude Code       │      │ 配对与认证              │      │ 审批卡片          │
│  Codex             │      │ E2EE 信封转发（不解密）  │      │ 终端视图 (xterm)  │
│  DeepSeek/Kimi ... │      │ 元数据：Postgres        │      │ 指令输入          │
│  tmux/PTY 兜底     │◀─────│ 不持久化内容            │◀─────│ 推送订阅          │
└────────────────────┘      └─────────────────────────┘      └──────────────────┘
```

### 3.2 模块职责

| 模块 | 职责 | 关键点 |
|---|---|---|
| daemon / 适配器层 | 将 CLI 的结构化输出转成统一事件 | 每个 CLI 一个适配器，实现同一接口 |
| daemon / 会话管理 | 启动、跟踪、恢复本地会话 | 崩溃后可重连，会话不丢 |
| daemon / 配对 | 生成配对码 + 二维码，完成设备密钥交换 | X25519，私钥存本机 |
| relay / Hub | 按会话路由 WebSocket 消息 | 房间模型：session → 多设备 |
| relay / 认证 | 校验设备 token，防止未配对设备接入 | 短时 token + 刷新 |
| relay / 元数据 | 设备、会话状态的持久化 | Postgres，仅非敏感字段 |
| mobile / 会话视图 | 事件流渲染、审批操作、指令输入 | 移动优先，竖屏可用 |
| mobile / 推送 | 关键事件转系统通知 | Web Push → 厂商通道 |

---

## 4. 事件协议（packages/protocol）

协议是三个端的唯一契约，定义在 `packages/protocol`，用 JSON Schema 生成 Go 与 TypeScript 类型。

### 4.1 传输

- 双向 WebSocket，UTF-8 JSON 帧
- 所有消息先经 E2EE 封装；明文只保留路由所需的最小元数据
- 消息带 `id`、`sessionId`、`timestamp`、`type`

### 4.2 信封格式（草案）

```json
{
  "v": 1,
  "kind": "event" | "control",
  "sessionId": "s_123",
  "nonce": "base64",
  "ciphertext": "base64"
}
```

路由用 `sessionId`；内容在 `ciphertext` 内二次加密（AES-256-GCM，密钥来自会话级 X25519 协商）。

### 4.3 事件集

| 事件 | 方向 | 说明 |
|---|---|---|
| `session_start` / `session_end` | daemon → mobile | 会话生命周期 |
| `agent_status` | daemon → mobile | `running` / `waiting_input` / `done` / `error` |
| `tool_call` | daemon → mobile | 工具名、参数摘要、状态 |
| `file_change` | daemon → mobile | 路径、变更摘要（不传全文，P1 可加 diff） |
| `command` | daemon → mobile | bash 命令与退出码 |
| `approval_request` | daemon → mobile | 审批请求：操作摘要、选项 |
| `approval_response` | mobile → daemon | `approve` / `reject` / 修改后条件 |
| `prompt` | mobile → daemon | 文字新指令 |
| `control` | 双向 | `pause` / `resume` / `stop` / `ping` / `pong` |
| `notify` | daemon → relay | 通知事件（等待审批、完成、出错），供推送 |

### 4.4 事件示例

```json
{
  "id": "evt_1",
  "sessionId": "s_123",
  "timestamp": 1760000000000,
  "type": "approval_request",
  "requestId": "req_456",
  "action": "file_delete",
  "summary": "删除 src/old.ts",
  "options": ["approve", "reject"]
}
```

---

## 5. daemon 设计

### 5.1 适配器接口

```go
type Adapter interface {
    Start(ctx context.Context, cfg SessionConfig) error
    Stop() error
    Events() <-chan Event
    Send(ctx context.Context, msg Message) error
}
```

实现时以各 CLI 当前官方输出模式为准（如 Claude Code 的结构化输出、Codex 的 JSON 事件），输出格式变化时只改适配器，不动协议。

### 5.2 tmux / PTY 兜底

- 使用 tmux 控制模式（`tmux -CC`）或 `creack/pty` 挂接任意终端会话
- 原始字节流封装为 `terminal_output` 事件，移动端用 xterm.js 渲染
- 该路径只读优先，输入发送为显式用户动作

### 5.3 配对与密钥

1. daemon 启动生成一次性配对码与 QR（含 short-lived token）
2. 手机扫码，双方交换 X25519 公钥，建立设备级密钥
3. 会话开始时派生会话密钥（ephemeral），中继无法推导
4. 私钥与设备列表存 `~/.config/riffpad/`（0600）

### 5.4 本地配置与恢复

- 会话元数据（CLI 类型、工作目录、启动命令）落本地 JSON
- daemon 重启后自动重连会话；agent 进程由 tmux 或 daemon 托管保持存活

---

## 6. relay 设计

### 6.1 WebSocket Hub

- 连接按 `sessionId` 加入房间；一个会话可绑定多设备（daemon + 手机）
- 事件在房间内广播；`approval_response` / `prompt` 定向发给 daemon
- 心跳与断线重连：30s ping，客户端指数退避

### 6.2 认证

- 配对成功后签发设备 token（JWT，短时 + 刷新）
- 每个连接必须携带有效 token；撤销设备后 token 立即失效

### 6.3 零知识保证

- relay 只解析信封头部（`sessionId`、路由），不解密 `ciphertext`
- 消息不写盘、不写日志；只记录元数据事件（连接、会话状态变化）
- 提供 `privacy.txt` 类公开说明：中继无解密密钥

### 6.4 数据模型（元数据）

```sql
users      (id, created_at)
devices    (id, user_id, name, public_key, created_at, revoked_at)
sessions   (id, device_id, cli, status, started_at, ended_at, last_seen_at)
pairing    (code_hash, device_id, expires_at, claimed_at)
tokens     (device_id, token_hash, expires_at)
```

内容（事件、文件、指令）不落库。

---

## 7. mobile 设计

### 7.1 页面

- 会话列表：状态徽标、CLI 标识、最近事件时间
- 会话详情：事件时间线 + 审批卡片 + 指令输入框
- 终端视图：xterm.js，仅在有原始字节流时启用
- 设置：设备管理、通知偏好、撤销

### 7.2 推送

- MVP：Web Push（VAPID），service worker 处理点击直达会话
- 原生壳（Capacitor / Expo）后接安卓厂商通道与 iOS APNs
- 推送内容只含非敏感摘要（“等待审批：删除 src/old.ts”）

---

## 8. API 设计

### 8.1 REST（配对与元数据）

```
POST /v1/pairing                 # daemon 创建配对码
POST /v1/pairing/claim           # 手机认领配对
GET  /v1/devices                 # 设备列表
DELETE /v1/devices/:id           # 撤销设备
GET  /v1/sessions                # 会话列表（元数据）
```

### 8.2 WebSocket

```
WS /v1/ws?token=<deviceToken>
```

daemon 与 mobile 共用通道，按角色与配对关系授权。

---

## 9. 安全设计

### 9.1 威胁模型

| 威胁 | 缓解 |
|---|---|
| relay 被攻破 / 恶意运维 | 端到端加密，中继无密钥，内容不可读 |
| 网络中间人 | TLS + 证书固定（移动端可选） |
| 手机丢失 | 远程撤销设备；daemon 端一键熔断 |
| 恶意配对二维码 | 一次性配对码 + 短时效 + 用户确认 |
| CLI 输出格式变化 | 适配器隔离，协议层稳定 |

### 9.2 基线

- 密钥只用 X25519 + AES-256-GCM；随机数不重用
- daemon 不读取、不上传用户 shell 环境变量中的密钥
- 移动端默认只读；发送指令需显式点击
- 所有远程控制动作在本地记审计日志（仅本机）

### 9.3 隐私承诺与边界（零知识）

**承诺：Riffpad 云端（relay 运营方）在架构上无法查看用户内容。**

- 内容（事件、文件路径、命令输出、提示词、审批）仅以 AES-256-GCM 密文经过 relay；relay 只解析信封头部的路由元数据（随机 `sessionId`、时间戳）
- 私钥只存在于用户电脑（`~/.config/riffpad/`，0600）与手机安全存储（Keychain / Keystore）；云端永不持有私钥
- relay 不持久化消息、不记录事件内容日志；元数据（连接、会话状态）设保留期并最小化采集（随机 id，不含仓库路径 / 项目名 / 命令参数）
- 会话密钥每次重新协商（ephemeral），前向安全：单次密钥泄漏不影响历史会话
- 推送通道不携带敏感内容：通知仅提示“有审批请求 / 状态变化”，详情由 app 解密后展示；加密推送 payload 为 P2 优化
- 支持自部署中继（P2），用户可彻底消除对官方 relay 的信任依赖

**边界（须如实告知用户，避免过度承诺）：**

- E2EE 覆盖 daemon ↔ 手机；用户数据仍会被 AI CLI 发送给其配置的 LLM 服务商，Riffpad 不干预
- 元数据（会话活跃状态、时间、IP、设备数）对 relay 可见，只能靠最小化采集降低暴露
- 客户端供应链是信任边界：恶意客户端更新可绕过 E2EE；缓解措施为签名发布、可复现构建、开源客户端

---

## 10. 部署与成本

| 项 | 一期 | 二期 |
|---|---|---|
| relay | Fly.io / Railway 单区域 | 国内节点（腾讯云 / 阿里云）+ 就近路由 |
| 数据库 | 托管 Postgres | 同区域托管 |
| 推送 | Web Push | 厂商通道（国内） |
| 成本 | 带宽为主，事件流体积小 | 国内节点带宽 + 推送配额 |

---

## 11. 里程碑与验收

| 里程碑 | 技术验收 |
|---|---|
| M0 | Claude Code 适配器可产出事件；网页端看到“等待审批”并返回 approve |
| M1 | daemon + relay + PWA 全链路，E2EE 生效；Codex 适配器；10 个种子用户 |
| M2 | 安卓原生壳 + 厂商推送；国内中继；DeepSeek / Kimi 适配器 |
| M3 | 分享链接；tmux 兜底；多设备与会话并发 |

---

## 12. 风险与待决策

### 风险

| 风险 | 影响 | 缓解 |
|---|---|---|
| 官方移动端覆盖自家 CLI | 单一 CLI 价值被替代 | 跨 CLI 通用 + 本地桥接 + 国内体验 |
| CLI 输出格式变动 | 适配器失效 | 协议稳定，适配器隔离，PTY 兜底 |
| 安全事件 | 信任崩塌 | 零知识中继 + 默认只读 + 熔断 |
| 中继带宽成本 | 毛利下降 | 压缩 + 免费层限制 + 自部署选项 |

### 待决策

1. 协议生成方式：JSON Schema（轻）vs protobuf（强类型、重）
2. 移动端路线：PWA → Capacitor（快）vs Expo（原生能力强）
3. Windows daemon 是否一期支持（目标用户以 Mac 为主？）
