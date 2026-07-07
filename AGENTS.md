# Riffpad - AI 编码助手项目指引

> 本文件面向未来参与 Riffpad 开发的 AI 编码助手。当前项目处于**早期规划阶段**，仓库内仅有产品需求文档与技术规格说明书，尚未进入实际编码实现。阅读本指引后，请始终结合 `docs/prd.md` 与 `docs/tsd.md` 进行设计与开发，不要对未实现的结构做假设。

---

## 1. 项目概述

**Riffpad** 定位是 "AI-Native 轻量化代码灵感草稿本"。它不是生产级 IDE，而是 AI 软件工程链路最上游的灵感孵化器：

- 让用户在通勤、散步等碎片化场景下，用手机即可创建一个有状态工作区。
- 工作区内置轻量沙箱与自主 Agent，可直接读写文件、执行 Bash、运行原型。
- 验证成功后，可一键重构并打包为下游 IDE（Cursor / Claude Code 等）可直接消费的规范项目骨架。

核心口号：低摩擦捕捉灵感 → 跑通原型验证 → 一键桥接下游。

---

## 2. 当前仓库状态

截至最新检查，仓库仅包含以下文件：

```
riffpad/
├── AGENTS.md
└── docs/
    ├── prd.md   # 产品需求文档（PRD）
    └── tsd.md   # 技术规格说明书（TSD）
```

**尚未存在的内容：**

- 没有 `package.json`、`go.mod`、`pyproject.toml` 等任何工程配置文件。
- 没有 `apps/`、`packages/`、`infra/` 等 Monorepo 目录。
- 没有 `Makefile`、`.env.example`、CI 工作流或测试脚本。
- 没有 README（生产级）。

因此，任何开发工作都应先从"搭建 Monorepo 骨架"开始，并严格以 `docs/tsd.md` 第 15 章"下一阶段任务拆分"作为路线图。

---

## 3. 规划中的技术栈

| 层级 | 选型 | 说明 |
|------|------|------|
| 应用前端 | Next.js 14 (App Router) + TypeScript + Tailwind CSS + shadcn/ui | `app.riffpad.ai`，PWA、Mobile-First |
| 营销站点 | Next.js 14 + TypeScript + Tailwind CSS + Framer Motion | `riffpad.ai`，SEO、静态生成 |
| 后端 API | Go 1.22+ + Echo / Fiber + GORM | `api.riffpad.ai`，长运行容器 |
| 认证 | Supabase Auth（或 Clerk） | 邮箱 / OAuth / 匿名登录，JWT 会话 |
| 数据库 | Supabase Postgres (PostgreSQL 16) | 关系型数据、自动备份 |
| 缓存 / 队列 | Upstash Redis + go-redis | 会话、预热池索引、WebSocket room |
| 沙箱运行时 | Firecracker microVM（生产）；Docker + gVisor（开发/早期） | 内核级隔离、资源可控 |
| 文件 / 快照存储 | Supabase Storage / AWS S3 | 沙箱快照、导出包持久化 |
| AI 模型 | 国产大模型（Kimi / GLM / Qwen / DeepSeek 等） | 通过 OpenAI 兼容接口统一调用 |
| 实时通信 | WebSocket（gorilla/websocket） | Agent 事件、日志流、文件同步 |
| 容器编排 | Docker Compose（本地）；Kubernetes（沙箱生产集群） | 后端用 Railway / Fly.io |
| 语音输入 | Web Speech API + Whisper API 回退 | 前端优先本地识别 |

---

## 4. 规划中的 Monorepo 结构

最终目标采用**轻量 Monorepo** 组织，前后端、沙箱镜像、共享协议、基础设施配置统一在一个仓库：

```
riffpad/
├── apps/
│   ├── app/                    # app.riffpad.ai 应用本体 (Next.js)
│   ├── landing/                # riffpad.ai 营销站点 (Next.js)
│   └── api/                    # Go 后端 (api.riffpad.ai)
│       ├── cmd/api/
│       ├── cmd/sandbox-worker/
│       ├── cmd/lifecycle-worker/
│       └── internal/
│           ├── config/         # 配置读取
│           ├── domain/         # 领域模型
│           ├── repository/     # 数据库访问 (GORM)
│           ├── service/        # 业务逻辑
│           ├── handler/        # HTTP / WebSocket handler
│           ├── client/         # 外部客户端
│           ├── middleware/     # 认证、限流、CORS、日志
│           └── pkg/            # 内部工具包
├── packages/
│   ├── shared-types/           # OpenAPI / Protobuf 定义
│   ├── ui/                     # 共享 UI 组件（可选）
│   └── sandbox-images/         # 沙箱基础镜像
│       ├── node-python/
│       └── firecracker/
├── infra/
│   ├── docker-compose.yml      # 本地开发一键启动
│   ├── k8s/                    # 沙箱集群 K8s manifests
│   └── terraform/              # 云资源定义（可选）
├── .github/workflows/          # CI/CD
├── docs/
│   ├── prd.md
│   └── tsd.md
├── Makefile
├── .env.example
├── README.md
└── .gitignore
```

**注意：** 当前仓库远未达到上述结构，需要从零开始逐步搭建。

---

## 5. 核心架构与模块职责

系统整体分为四层：

1. **Client（PWA）**：首页输入框、左侧文件树、右侧预览画布、下方意图交互区。
2. **Go Backend Service**：API Gateway、WebSocket、Workspace Service、Agent Orchestrator、Sandbox Scheduler、File Sync、Export Service。
3. **Sandbox Cluster**：Firecracker / Docker 沙箱实例、Sandbox Worker、Lifecycle Worker。
4. **Serverless / Managed Layer**：Supabase（Postgres / Auth / Storage）、Upstash Redis、LLM Providers。

核心模块职责详见 `docs/tsd.md` 第 3.2 节。

---

## 6. 关键设计原则

开发时必须遵循以下原则：

1. **Mobile-First**：前端深度适配手机/平板，优先支持语音输入。
2. **延迟敏感**：New Spark 必须在 1s 内完成分配与启动。
3. **安全优先**：Agent 可执行任意 Bash，沙箱必须内核级隔离。
4. **成本可控**：临时沙箱 48h 自动休眠，只保留结构化快照。
5. **桥接友好**：导出产物必须是下游 IDE 可直接消费的规范项目。
6. **Monorepo 组织**：前后端、沙箱镜像、共享协议、基础设施配置统一放在一个仓库。

---

## 7. 本地开发与构建命令

### 7.1 计划中的 Makefile 命令

由于 `Makefile` 尚未创建，后续应实现以下命令（直接引用自 `docs/tsd.md`）：

```makefile
.PHONY: dev dev-local build test

# 使用 Supabase 作为数据库/Auth/Storage（推荐）
dev:
	docker compose -f infra/docker-compose.yml up -d sandbox-worker
	cd apps/app && pnpm dev &
	cd apps/landing && pnpm dev &
	cd apps/api && go run ./cmd/api

# 完全本地开发（包含 Postgres/Redis/MinIO）
dev-local:
	docker compose -f infra/docker-compose.yml up -d
	cd apps/app && pnpm dev &
	cd apps/landing && pnpm dev &
	cd apps/api && go run ./cmd/api

build:
	cd apps/app && pnpm build
	cd apps/landing && pnpm build
	cd apps/api && go build -o bin/api ./cmd/api

test:
	cd apps/app && pnpm test
	cd apps/landing && pnpm test
	cd apps/api && go test ./...
```

### 7.2 开发模式

**方案 A：完全本地开发（无需 Supabase 账号）**

`infra/docker-compose.yml` 应包含：
- PostgreSQL（端口 5432）
- Redis（端口 6379）
- MinIO（端口 9000/9001）
- Sandbox Worker（开发期用 Docker 简化）

此时 Auth 使用简化的本地 JWT 方案，仅用于开发测试。

**方案 B：连接 Supabase 开发环境（推荐）**

配置环境变量，详见 `docs/tsd.md` 第 12.4 节 `.env.example`。

---

## 8. 代码组织约定

### 8.1 后端 Go 项目

- 使用 **GORM** 作为 ORM，数据库模型定义在 `internal/domain/`。
- 数据库访问层放在 `internal/repository/`。
- 业务逻辑层放在 `internal/service/`。
- HTTP / WebSocket handler 放在 `internal/handler/`。
- 外部客户端（Supabase、LLM、S3）放在 `internal/client/`。
- 中间件（认证、限流、CORS、日志）放在 `internal/middleware/`。
- 内部工具包放在 `internal/pkg/`。
- 三个入口：
  - `cmd/api/main.go`
  - `cmd/sandbox-worker/main.go`
  - `cmd/lifecycle-worker/main.go`

### 8.2 前端 Next.js 项目

- `apps/app/`：应用本体，PWA，动态渲染，靠近后端。
- `apps/landing/`：营销站点，静态导出，SEO 优先。
- 共享 UI 组件可逐步沉淀到 `packages/ui/`。
- 共享类型定义在 `packages/shared-types/openapi.yaml`，前后端生成类型。

### 8.3 沙箱镜像

- `packages/sandbox-images/node-python/`：Node.js 20 + Python 3.11 基础镜像。
- `packages/sandbox-images/firecracker/`：Firecracker microVM 构建脚本。

---

## 9. 测试策略

当前阶段测试策略需从零建立，建议按以下优先级：

1. **单元测试**：Go 后端 `go test ./...`，前端 `pnpm test`。
2. **API 契约测试**：基于 `packages/shared-types/openapi.yaml` 验证前后端接口一致性。
3. **沙箱隔离测试**：验证沙箱内无法访问宿主机、内部服务、敏感目录。
4. **端到端测试**：验证 New Spark → Agent 修改文件 → 预览 → 导出 Zip 的完整链路。
5. **性能基准**：New Spark P95 < 800ms，休眠唤醒 < 3s。

---

## 10. 安全与合规

### 10.1 沙箱隔离

| 层级 | 措施 |
|------|------|
| 虚拟化 | Firecracker microVM，每个沙箱独立内核 |
| 网络 | 默认禁止出网；白名单放行 npm/pip registry |
| 文件系统 | OverlayFS，只读 base image + 可写层 |
| 资源 | cgroups 限制 CPU/内存/磁盘/网络 |
| 时间 | 单条 Bash 命令默认最大 30s |

### 10.2 Agent 权限档位

每个 Workspace 可独立设置 Agent 权限：

| 档位 | 说明 |
|------|------|
| Safe | 只读模式，不会修改文件或执行命令 |
| Confirm | 默认模式，副作用操作批量请求用户确认 |
| Auto | 自动模式，可自主调用全部工具 |

### 10.3 风险控制基线

- `file_write` / `file_replace` 只能操作 `/sandbox/workspace/` 目录。
- 禁止访问 `/etc`、`/root`、`/proc`、`.riffpad/` 等系统目录。
- 高危命令（如 `rm -rf /`、`curl | bash`）默认拦截。
- `search_web` / `fetch_url` 必须走后端代理，沙箱不直连外网。
- `preview_start` 只能使用 3000-9000 端口。
- 单沙箱资源受 cgroups 限制。

---

## 11. 环境变量

后续应在根目录创建 `.env.example`，完整模板参见 `docs/tsd.md` 第 12.4 节。核心变量包括：

```bash
# Go Backend
PORT=8080
API_URL=http://localhost:8080

# Supabase
SUPABASE_URL=https://your-project.supabase.co
SUPABASE_ANON_KEY=xxx
SUPABASE_SERVICE_ROLE_KEY=xxx
DATABASE_URL=postgresql://postgres:[password]@db.your-project.supabase.co:5432/postgres

# Upstash Redis
REDIS_URL=rediss://default:[password]@your-redis.upstash.io

# Supabase Storage / S3
S3_ENDPOINT=https://your-project.supabase.co/storage/v1/s3
S3_BUCKET=riffpad-snapshots
S3_ACCESS_KEY=xxx
S3_SECRET_KEY=xxx

# LLM Provider
LLM_PROVIDER=volcengine
LLM_BASE_URL=https://ark.cn-beijing.volces.com/api/v3
LLM_API_KEY=sk-xxx
LLM_MODEL=doubao-pro-32k-xxx

# Sandbox
SANDBOX_RUNTIME=docker          # docker | firecracker
SANDBOX_WORKER_URL=http://localhost:9090
SANDBOX_IMAGE=riffpad-node-python
```

---

## 12. 开发路线图

后续开发应严格按照 `docs/tsd.md` 第 15 章执行：

### 阶段一：搭建 Monorepo 骨架
1. 创建 `apps/`、`packages/`、`infra/` 目录。
2. 初始化 `apps/app/`（Next.js + Tailwind + shadcn/ui）。
3. 初始化 `apps/api/`（Go module + Echo/Fiber + Supabase Postgres + Upstash Redis）。
4. 初始化 `apps/landing/`（Next.js + Tailwind + Framer Motion）。
5. 创建 `packages/shared-types/openapi.yaml`。
6. 创建 `infra/docker-compose.yml`。
7. 前后端联调，配置 CORS，统一 Makefile。

### 阶段二：沙箱与 Agent
8. 实现 Docker 简化沙箱、文件读写 API、Bash 执行 API。
9. 接入 LLM Agent，Function Calling 工具定义，WebSocket 实时同步。

### 阶段三：核心体验
10. 实现预览画布（HTML/JS 预览代理）。
11. 实现导出功能（Zip 打包、README 自动生成）。
12. Mobile-First UI、语音输入集成。

### 阶段四：生命周期与成本
13. 快照机制、自动休眠任务、唤醒恢复。

---

## 13. 参考资料

- `docs/prd.md`：产品需求文档，包含定位、用户画像、功能需求、非功能需求、KPIs。
- `docs/tsd.md`：技术规格说明书，包含技术栈、系统架构、数据模型、Agent 工具集、API 设计、部署架构、开发路线图。

---

*本文件基于当前仓库实际状态编写。随着项目推进，应同步更新本文件以反映最新的工程结构、命令与约定。*
