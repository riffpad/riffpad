# Riffpad 技术规格说明书 (TSD)

> 版本：v0.2
> 基于 PRD：《AI-Native 轻量化代码灵感草稿本 (Riffpad) 产品需求文档》

---

## 1. 概述

### 1.1 文档目标

本文档将 PRD 中的产品需求转化为可落地的技术实现方案，明确：
- 技术栈选型
- 系统架构与模块划分
- 核心数据模型
- 关键接口定义
- 安全与生命周期策略
- 部署与成本结构

### 1.2 设计原则

1. **Mobile-First**：前端深度适配手机/平板，优先支持语音输入。
2. **延迟敏感**：New Spark 必须在 1s 内完成分配与启动。
3. **安全优先**：Agent 可执行任意 Bash，沙箱必须内核级隔离。
4. **成本可控**：临时沙箱 48h 自动休眠，只保留结构化快照。
5. **桥接友好**：导出产物必须是下游 IDE 可直接消费的规范项目。
6. **Monorepo 组织**：前后端、沙箱镜像、共享协议、基础设施配置统一放在一个仓库，降低单人/小团队协作成本。

---

## 2. 技术栈选型

| 层级 | 技术 | 说明 |
|------|------|------|
| **应用前端** | Next.js 14 (App Router) + TypeScript + Tailwind CSS + shadcn/ui | app.riffpad.ai，PWA 支持、跨端统一 |
| **营销站点** | Next.js 14 + TypeScript + Tailwind CSS + Framer Motion | riffpad.ai，SEO 优先、静态生成、高转化率 |
| **后端 API** | Go 1.22+ + Echo / Fiber + GORM | api.riffpad.ai，部署在 Railway / Fly.io 长运行容器 |
| **认证服务** | Supabase Auth / Clerk | 邮箱/OAuth/匿名用户，JWT 会话管理 |
| **数据库** | **Supabase Postgres** (PostgreSQL 16) + GORM | 托管关系型数据、自动备份、标准 SQL |
| **缓存/队列** | **Upstash Redis** / Redis Cloud + go-redis | 会话状态、预热池索引、任务队列、WebSocket 广播 |
| **沙箱运行时** | Firecracker microVM (生产) / Docker + gVisor (开发&早期) | 内核级隔离、毫秒级启动、资源可控 |
| **文件/快照存储** | **Supabase Storage** / AWS S3 | 沙箱快照、导出包持久化，S3-compatible API |
| **AI 模型** | 国产大模型（Kimi / GLM / Qwen / DeepSeek 等）via 火山引擎 / 阿里云 / 硅基流动等 | `sashabaranov/go-openai` 统一调用，切换模型仅需改配置 |
| **实时通信** | WebSocket (gorilla/websocket) | 日志流、文件变更、预览刷新 |
| **容器编排** | Docker Compose (本地开发) / Kubernetes (仅沙箱集群生产) | 后端与 Worker 用 Railway/Fly.io，沙箱用 K8s 或裸金属 |
| **语音输入** | Web Speech API + Whisper API 回退 | 前端优先本地识别，复杂场景走云端 |

---

## 3. 系统架构

### 3.1 整体架构图

```
                              ┌─────────────────┐
                              │  LLM Providers  │
                              │ (Volcengine/    │
                              │  Aliyun/DeepSeek│
                              │  /SiliconFlow)  │
                              └────────┬────────┘
                                       │ OpenAI-compatible API
                                       ▼
┌─────────────────────────────────────────────────────────────────────┐
│                         Client (PWA)                                │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐              │
│  │  Spark Input │  │  File Tree   │  │ Preview/Web  │              │
│  │  (Voice/Text)│  │  (左侧)       │  │ Canvas (右侧) │              │
│  └──────────────┘  └──────────────┘  └──────────────┘              │
└─────────────────────┬───────────────────────────────────────────────┘
                      │ HTTPS / WebSocket
                      ▼
┌─────────────────────────────────────────────────────────────────────┐
│              Go Backend Service (Railway / Fly.io)                  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐              │
│  │  API Gateway │  │  Workspace   │  │   Export     │              │
│  │  + WebSocket │  │  Service     │  │  Service     │              │
│  └──────────────┘  └──────┬───────┘  └──────────────┘              │
│                           │                                         │
│  ┌──────────────┐  ┌──────▼───────┐  ┌──────────────┐              │
│  │  Agent LLM   │  │  Sandbox     │  │  File Sync   │              │
│  │  Orchestrator│  │  Scheduler   │  │  Service     │              │
│  └──────────────┘  └──────┬───────┘  └──────────────┘              │
│                           │                                         │
│              ┌────────────▼────────────┐                           │
│              │     Preview Proxy       │                           │
│              │  preview.riffpad.app    │                           │
│              └────────────┬────────────┘                           │
└───────────────────────────┼───────────────────────────────────────┘
                            │ gRPC / HTTP
                            ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      Sandbox Cluster                                │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │  Firecracker microVM / Docker+gVisor                         │  │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐     │  │
│  │  │  Agent   │  │  File    │  │  Bash    │  │  Web     │     │  │
│  │  │  Process │  │  System  │  │  Executor│  │  Server  │     │  │
│  │  └──────────┘  └──────────┘  └──────────┘  └──────────┘     │  │
│  └──────────────────────────────────────────────────────────────┘  │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │         Sandbox Worker / Lifecycle Worker                    │  │
│  │     (预热池维护 / 休眠 / 唤醒 / 资源清理)                      │  │
│  └──────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────┐
│                     Serverless / Managed Layer                      │
│  ┌─────────────────────┐  ┌─────────────────────┐                  │
│  │     Supabase        │  │    Upstash Redis    │                  │
│  │  ├─ Postgres        │  │                     │                  │
│  │  ├─ Auth            │  │  会话 / 预热池索引   │                  │
│  │  └─ Storage         │  │  / WebSocket room   │                  │
│  └─────────────────────┘  └─────────────────────┘                  │
└─────────────────────────────────────────────────────────────────────┘
```

### 3.2 核心模块职责

| 模块 | 职责 | 部署/依赖 |
|------|------|----------|
| **API Gateway + WebSocket** | 统一 REST / WebSocket 入口，连接前后端长连接 | Railway / Fly.io 长运行容器 |
| **Auth Service** | 基于 Supabase Auth 提供邮箱/OAuth/匿名登录；Go backend 只校验 JWT | Supabase Auth |
| **Workspace Service** | 创建/列表/删除/恢复工作区，协调沙箱分配 | Supabase Postgres + Upstash Redis |
| **Sandbox Scheduler** | 维护预热池、分配沙箱、回收休眠、资源配额 | 长运行容器，直连 Sandbox Worker |
| **Agent Orchestrator** | 将用户意图 + 沙箱状态组装成 LLM Prompt，解析 Tool Calls | 调用外部 LLM Providers |
| **File Sync Service** | 沙箱 ↔ Supabase Postgres / Supabase Storage 的文件同步与快照 | Supabase Postgres + Storage |
| **Preview Proxy** | 将沙箱内 Web 服务端口代理为可访问 URL | 长运行容器或边缘网关 |
| **Export Service** | 重构代码、生成 README、打包 Zip / 推送到 GitHub | Supabase Storage + GitHub API |
| **Sandbox Worker** | 管理 Firecracker / Docker 沙箱实例生命周期 | K8s / 裸金属 |
| **Lifecycle Worker** | 定时扫描过期 Workspace，执行休眠、唤醒、清理 | 长运行容器 |

### 3.3 Monorepo 项目结构

整个 Riffpad 采用 **轻量 Monorepo** 组织，前后端、沙箱镜像、共享协议、基础设施配置统一放在一个仓库中管理。

```
riffpad/
├── apps/
│   ├── app/                    # app.riffpad.ai 应用本体
│   │   ├── package.json
│   │   ├── next.config.js
│   │   ├── Dockerfile
│   │   └── src/
│   │
│   ├── landing/                # riffpad.ai 营销站点
│   │   ├── package.json
│   │   ├── next.config.js
│   │   └── src/
│   │       ├── app/
│   │       │   ├── page.tsx        # 首页
│   │       │   ├── pricing/page.tsx # 价格页
│   │       │   └── docs/           # 文档页
│   │       └── components/
│   │
│   └── api/                    # Go 后端（api.riffpad.ai）
│       ├── go.mod
│       ├── Dockerfile
│       ├── cmd/
│       │   ├── api/            # API 服务入口
│       │   │   └── main.go
│       │   ├── sandbox-worker/ # 沙箱调度 Worker
│       │   │   └── main.go
│       │   └── lifecycle-worker/ # 休眠/唤醒/清理 Worker
│       │       └── main.go
│       └── internal/
│           ├── config/         # 配置读取
│           ├── domain/         # 领域模型
│           ├── repository/     # 数据库访问层（GORM）
│           ├── service/        # 业务逻辑层
│           ├── handler/        # HTTP / WebSocket handler
│           ├── client/         # 外部客户端
│           ├── middleware/     # 认证、限流、CORS、日志
│           └── pkg/            # 内部工具包
│
├── packages/
│   ├── shared-types/           # 共享 OpenAPI / Protobuf 定义
│   │   └── openapi.yaml
│   ├── ui/                     # 共享 UI 组件（可选）
│   └── sandbox-images/         # 沙箱基础镜像
│       ├── node-python/
│       │   └── Dockerfile
│       └── firecracker/
│           └── build.sh
│
├── infra/
│   ├── docker-compose.yml      # 本地开发一键启动（仅沙箱 + 可选本地 DB/Redis/MinIO）
│   ├── k8s/                    # 沙箱集群 Kubernetes manifests
│   └── terraform/              # 云资源定义（可选）
│
├── .github/
│   └── workflows/              # CI/CD
│       ├── app.yml             # 应用前端 CI
│       ├── landing.yml         # 营销站 CI
│       └── api.yml             # 后端 CI
│
├── docs/
│   ├── prd.md
│   └── tsd.md
│
├── Makefile                    # 常用开发命令
├── .env.example                # Supabase / Upstash / LLM / 沙箱配置模板
├── README.md
└── .gitignore
```

### 3.4 为什么用 Monorepo

| 优势 | 说明 |
|------|------|
| **版本统一** | 前端、后端、沙箱镜像、协议定义共用一次 git 提交历史 |
| **联调方便** | 改接口时前后端同一次 PR，避免版本不一致 |
| **共享产物** | OpenAPI 定义放在 `packages/shared-types/`，前后端都能生成类型 |
| **部署统一** | `infra/` 集中管理 Docker Compose、沙箱 K8s、CI 配置；生产数据库/Auth/缓存使用托管服务 |
| **上下文切换少** | 一人开发时不用在多个仓库之间跳转 |

### 3.5 站点与 URL 路由规划

| 域名 / 路径 | 用途 | 所属 App | 技术重点 |
|------------|------|----------|----------|
| `riffpad.ai` | 营销首页 | `apps/landing` | SEO、静态生成、品牌展示 |
| `riffpad.ai/pricing` | 价格页（Pricing） | `apps/landing` | 转化率优化 |
| `riffpad.ai/docs` | 产品文档 | `apps/landing` | 内容导航、搜索 |
| `riffpad.ai/blog` | 博客（未来） | `apps/landing` | SEO、内容营销 |
| `app.riffpad.ai` | 应用本体 | `apps/app` | 复杂交互、实时通信、PWA |
| `app.riffpad.ai/w/:slug` | 具体 Workspace | `apps/app` | 沙箱IDE、文件树、预览画布 |
| `api.riffpad.ai` | Go 后端 API | `apps/api` | 高并发、低延迟、安全 |

> **为什么不用 `docs.riffpad.ai`？**  
> 早期产品文档规模较小，使用 `riffpad.ai/docs` 可以保持品牌连贯性、集中 SEO 权重、缩短用户从了解到使用的路径。当文档规模扩大（多版本、多语言、独立团队维护）时，再考虑拆分为 `docs.riffpad.ai`。

---

## 4. 数据模型

### 4.1 核心实体

以下使用 Prisma 语法描述实体关系，实际 Go 后端中使用 **GORM** 或 **sqlx** 实现对应的数据库表结构。

> **认证说明**：用户认证由 **Supabase Auth** 负责，应用层 `User` 表只保留与业务相关的扩展资料（如偏好设置、创建时间）。`id` 与 Supabase `auth.users.id` 保持一致。

```prisma
model User {
  id            String      @id // 与 supabase auth.users.id 一致
  email         String?     @unique
  anonymousId   String?     @unique // 匿名用户由 Supabase Auth 生成
  createdAt     DateTime    @default(now())
  workspaces    Workspace[]
}

model Workspace {
  id              String          @id @default(cuid())
  slug            String          @unique // 随机代号，如 amber-spark-9
  name            String?         // 用户后续可命名
  ownerId         String
  owner           User            @relation(fields: [ownerId], references: [id])
  
  status          WorkspaceStatus @default(WARM)
  // WARM: 活跃运行中
  // HIBERNATING: 已休眠，只保留快照
  // ARCHIVED: 用户主动长期保存
  
  sandboxId       String?         // 当前绑定的沙箱实例 ID
  baseImage       String          @default("riffpad-node-python")
  
  lastActiveAt    DateTime        @default(now())
  createdAt       DateTime        @default(now())
  
  files           File[]
  messages        Message[]
  snapshots       Snapshot[]
  
  resourceQuota   Json            @default("{\"cpu\":1,\"memory\":512,\"disk\":1024}")
  isPinned        Boolean         @default(false) // 长期保存，不自动休眠
}

model File {
  id            String    @id @default(cuid())
  workspaceId   String
  workspace     Workspace @relation(fields: [workspaceId], references: [id], onDelete: Cascade)
  path          String
  content       String    @db.Text
  language      String?   // 用于前端语法高亮
  isGenerated   Boolean   @default(false)
  updatedAt     DateTime  @default(now())
  
  @@unique([workspaceId, path])
}

model Message {
  id            String    @id @default(cuid())
  workspaceId   String
  workspace     Workspace @relation(fields: [workspaceId], references: [id], onDelete: Cascade)
  role          String    // user | assistant | system | tool
  content       String    @db.Text
  toolCalls     Json?     // LLM 调用的工具列表
  toolResults   Json?     // 工具执行结果
  createdAt     DateTime  @default(now())
}

model Snapshot {
  id            String    @id @default(cuid())
  workspaceId   String
  workspace     Workspace @relation(fields: [workspaceId], references: [id], onDelete: Cascade)
  version       String    // 快照版本号
  fileSnapshot  Json      // 文件路径与内容摘要
  storageKey    String    // 对象存储中的完整快照 Key
  createdAt     DateTime  @default(now())
}
```

### 4.2 沙箱状态（Upstash Redis）

使用 **Upstash Redis**（serverless Redis）存储临时状态、会话索引和预热池信息。

```
workspace:{id}:sandbox      -> sandbox instance id
workspace:{id}:preview_url  -> https://preview.riffpad.app/p/{token}
workspace:{id}:websocket    -> active socket room
sandbox:pool:warm           -> sorted set of warm sandbox ids (score = ready time)
sandbox:{id}:metrics        -> { cpu, memory, network, lastHeartbeat }
```

> 选择 Upstash 的原因：无服务器、按请求计费、与 Vercel / Railway 同区域部署时延迟可接受；当规模扩大后可迁移到自管 Redis Cluster。

---

## 5. 沙箱与 Agent 设计

### 5.1 沙箱内部结构

每个沙箱是一个最小化的 Linux 微虚拟机：

```
/sandbox/
├── workspace/          # 用户可见的工作目录
│   ├── index.html
│   ├── main.py
│   └── ...
├── .riffpad/           # Riffpad 元数据目录（用户不可见）
│   ├── agent.log
│   └── state.json
└── package.json / requirements.txt  # 根据需求动态生成
```

预装环境：
- Node.js 20 + npm
- Python 3.11 + pip
- 轻量 http-server / vite（用于预览）
- curl, git, jq 等常用命令

### 5.2 Agent 工具集

LLM 通过 Function Calling 调用以下工具，Go 后端中统一用 `openai.FunctionDefinition` 描述。工具分为六大类：文件操作、文件搜索、Bash/包管理、预览控制、用户交互、网络与任务、系统内部。

#### 5.2.1 文件操作工具

| 工具 | 用途 | 关键参数 |
|------|------|----------|
| `file_read` | 读取沙箱内指定文件内容 | `path` |
| `file_write` | 创建或覆盖文件，路径不存在则自动创建目录 | `path`, `content`, `mode` |
| `file_replace` | 替换文件中匹配的字符串 | `path`, `old`, `new`, `replace_all` |
| `file_delete` | 删除指定文件 | `path` |
| `file_list` | 列出指定目录下的文件和子目录 | `directory` |
| `read_media_file` | 读取沙箱内的图片或视频文件 | `path` |

```go
var FileTools = []openai.Tool{
	{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "file_read",
			Description: "读取沙箱内指定文本文件的内容",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]string{"type": "string"},
				},
				"required": []string{"path"},
			},
		},
	},
	{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "file_write",
			Description: "向沙箱写入或覆盖文件，若路径不存在则自动创建目录",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path":    map[string]string{"type": "string"},
					"content": map[string]string{"type": "string"},
					"mode":    map[string]string{"type": "string", "enum": []string{"overwrite", "append"}},
				},
				"required": []string{"path", "content"},
			},
		},
	},
	{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "file_replace",
			Description: "替换文件中匹配的字符串，支持单次或全部替换",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path":         map[string]string{"type": "string"},
					"old":          map[string]string{"type": "string"},
					"new":          map[string]string{"type": "string"},
					"replace_all":  map[string]string{"type": "boolean"},
				},
				"required": []string{"path", "old", "new"},
			},
		},
	},
	{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "file_delete",
			Description: "删除沙箱内指定文件",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]string{"type": "string"},
				},
				"required": []string{"path"},
			},
		},
	},
	{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "file_list",
			Description: "列出沙箱内指定目录下的文件和子目录",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"directory": map[string]string{"type": "string"},
				},
				"required": []string{"directory"},
			},
		},
	},
	{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "read_media_file",
			Description: "读取沙箱内的图片或视频文件，用于图像识别或视频理解",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]string{"type": "string"},
				},
				"required": []string{"path"},
			},
		},
	},
}
```

#### 5.2.2 文件搜索工具

| 工具 | 用途 | 关键参数 |
|------|------|----------|
| `file_glob` | 使用 glob 模式查找文件 | `pattern`, `directory` |
| `file_grep` | 基于 ripgrep 搜索文件内容 | `pattern`, `path`, `output_mode` |

```go
var SearchTools = []openai.Tool{
	{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "file_glob",
			Description: "使用 glob 模式查找文件和目录，如 src/**/*.js",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"pattern":   map[string]string{"type": "string"},
					"directory": map[string]string{"type": "string"},
				},
				"required": []string{"pattern"},
			},
		},
	},
	{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "file_grep",
			Description: "基于 ripgrep 搜索文件内容，支持正则表达式",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"pattern":     map[string]string{"type": "string"},
					"path":        map[string]string{"type": "string"},
					"output_mode": map[string]string{"type": "string", "enum": []string{"content", "files"}},
				},
				"required": []string{"pattern"},
			},
		},
	},
}
```

#### 5.2.3 Bash 与包管理工具

| 工具 | 用途 | 关键参数 |
|------|------|----------|
| `bash_exec` | 在沙箱内执行任意 Bash 命令 | `command`, `timeout`, `run_in_background` |
| `npm_install` | 安装 npm 包 | `packages`, `dev` |
| `pip_install` | 安装 pip 包 | `packages` |

`bash_exec` 是 Agent 最强大的工具，典型使用场景包括：

```bash
python main.py
node index.js
npm install express
pip install requests
ls -la workspace/
cat package.json
```

支持后台运行，用于 `npm run dev`、Mock Server 等长驻进程：

```go
{
    Type: openai.ToolTypeFunction,
    Function: &openai.FunctionDefinition{
        Name:        "bash_exec",
        Description: "在沙箱内执行 Bash 命令，支持前台和后台运行",
        Parameters: map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "command":           map[string]string{"type": "string"},
                "timeout":           map[string]string{"type": "integer"},
                "run_in_background": map[string]string{"type": "boolean"},
            },
            "required": []string{"command"},
        },
    },
}
```

> 注：`npm_install` / `pip_install` 本质上可通过 `bash_exec` 实现，但单独抽象出来能让 Agent 更准确地理解语义，也便于做权限控制、安全审计和日志记录。

```go
var ExecTools = []openai.Tool{
	{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "npm_install",
			Description: "在沙箱内安装指定的 npm 包",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"packages": map[string]interface{}{
						"type": "array",
						"items": map[string]string{"type": "string"},
					},
					"dev": map[string]string{"type": "boolean"},
				},
				"required": []string{"packages"},
			},
		},
	},
	{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "pip_install",
			Description: "在沙箱内安装指定的 pip 包",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"packages": map[string]interface{}{
						"type": "array",
						"items": map[string]string{"type": "string"},
					},
				},
				"required": []string{"packages"},
			},
		},
	},
}
```

#### 5.2.4 预览控制工具

| 工具 | 用途 | 关键参数 |
|------|------|----------|
| `preview_start` | 启动沙箱内 Web 服务，在右侧画布实时预览 | `command`, `port` |
| `preview_stop` | 停止指定端口的预览服务 | `port` |
| `preview_screenshot` | 对预览页面截图（二期） | `output_path` |

```go
var PreviewTools = []openai.Tool{
	{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "preview_start",
			Description: "启动一个本地 Web 服务用于预览，并返回可访问的预览 URL",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"command": map[string]string{"type": "string"},
					"port":    map[string]string{"type": "integer"},
				},
				"required": []string{"command", "port"},
			},
		},
	},
	{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "preview_stop",
			Description: "停止指定端口的预览服务",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"port": map[string]string{"type": "integer"},
				},
				"required": []string{"port"},
			},
		},
	},
}
```

#### 5.2.5 用户交互工具

| 工具 | 用途 | 关键参数 |
|------|------|----------|
| `ask_user` | 向用户提出结构化问题，收集偏好或澄清需求 | `question`, `options` |

典型场景：
- "你想用 React 还是 Vue？"
- "这个命令会删除文件，是否继续？"
- "你希望倒计时精确到秒还是毫秒？"

```go
var InteractionTools = []openai.Tool{
	{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "ask_user",
			Description: "向用户提出结构化问题，收集偏好或澄清需求，会暂停 Agent loop 等待回复",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"question": map[string]string{"type": "string"},
					"options": map[string]interface{}{
						"type": "array",
						"items": map[string]string{"type": "string"},
					},
				},
				"required": []string{"question"},
			},
		},
	},
}
```

#### 5.2.6 网络与任务管理工具

| 工具 | 用途 | 关键参数 |
|------|------|----------|
| `search_web` | 网络搜索，获取最新信息或文档 | `query`, `limit` |
| `fetch_url` | 获取网页主要内容 | `url` |
| `set_todo_list` | 管理当前 Workspace 的待办事项 | `todos` |
| `task_list` | 列出当前会话的后台任务 | - |
| `task_output` | 获取后台任务的输出 | `task_id` |
| `task_stop` | 停止后台任务 | `task_id` |

网络工具：

```go
var NetworkTools = []openai.Tool{
	{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "search_web",
			Description: "搜索互联网获取最新信息、文档或示例",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]string{"type": "string"},
					"limit": map[string]string{"type": "integer"},
				},
				"required": []string{"query"},
			},
		},
	},
	{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "fetch_url",
			Description: "获取指定网页的主要内容",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"url": map[string]string{"type": "string"},
				},
				"required": []string{"url"},
			},
		},
	},
}
```

任务管理工具：

```go
var TaskTools = []openai.Tool{
	{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "set_todo_list",
			Description: "管理当前 Workspace 的待办事项列表，用于跟踪复杂任务进度",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"todos": map[string]interface{}{
						"type": "array",
						"items": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"title":  map[string]string{"type": "string"},
								"status": map[string]string{"type": "string", "enum": []string{"pending", "in_progress", "done"}},
							},
						},
					},
				},
				"required": []string{"todos"},
			},
		},
	},
	{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "task_list",
			Description: "列出当前会话中正在运行或最近完成的后台任务",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{},
			},
		},
	},
	{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "task_output",
			Description: "获取指定后台任务的输出快照",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"task_id": map[string]string{"type": "string"},
				},
				"required": []string{"task_id"},
			},
		},
	},
	{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "task_stop",
			Description: "停止指定的后台任务",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"task_id": map[string]string{"type": "string"},
				},
				"required": []string{"task_id"},
			},
		},
	},
}
```

#### 5.2.7 系统内部工具

这些工具不直接暴露给 LLM，由 Agent Orchestrator 内部调用：

| 工具 | 用途 |
|------|------|
| `snapshot_create` | 关键操作前自动保存文件快照，便于回滚 |
| `snapshot_rollback` | 回滚到指定快照 |
| `context_compress` | 上下文满时生成对话摘要 |
| `sandbox_health_check` | 检测沙箱是否存活 |
| `event_emit` | 通过 WebSocket 向前端推送 Agent 事件 |

#### 5.2.8 Agent 权限控制

Riffpad 提供三档 Agent 权限级别，用户可在 Workspace 设置或全局偏好中切换：

```go
type AgentPermissionLevel string

const (
	PermissionSafe    AgentPermissionLevel = "safe"    // 安全模式：只读
	PermissionConfirm AgentPermissionLevel = "confirm" // 确认模式：副作用操作需用户确认
	PermissionAuto    AgentPermissionLevel = "auto"    // 自动模式：Agent 自主执行
)

type WorkspaceSettings struct {
	WorkspaceID     string               `json:"workspaceId"`
	AgentPermission AgentPermissionLevel `json:"agentPermission" default:"confirm"`
}
```

| 档位 | 对外名称 | 含义 |
|------|---------|------|
| **Safe** | 安全模式 | Agent 只能读取文件、分析代码、查看任务状态，不会修改任何内容 |
| **Confirm** | 确认模式 | Agent 可以执行写操作、运行命令、安装依赖，但每次执行前需要用户批量确认 |
| **Auto** | 自动模式 | Agent 可以自主调用任何工具，包括联网搜索、图片识别、后台任务等 |

##### 各档位可用工具

**Safe Mode**

```
file_read, file_list, file_glob, file_grep,
task_list, task_output
```

**Confirm Mode**

Safe 工具 + 以下需要确认的工具：

```
file_write, file_replace, file_delete,
bash_exec, npm_install, pip_install,
preview_start, preview_stop,
search_web, fetch_url,
set_todo_list, task_stop
```

**Auto Mode**

全部工具，包括：

```
read_media_file, 以及 Confirm 模式下的所有工具
```

##### 批量确认机制

Confirm 模式下，Agent 不会每个 tool call 都弹窗，而是将一轮中的副作用操作收集为批量确认请求：

```
Agent 请求执行以下操作：
  ✓ 写入文件：src/index.html
  ✓ 替换内容：src/main.js (1 处)
  ✓ 执行命令：npm install express
  ✓ 启动预览：port 5173

[ 全部允许 ]  [ 取消 ]  [ 查看详情 ]
```

用户确认后，Agent 才会继续执行；取消则回滚到本轮操作前的快照。

##### 风险控制基线

无论哪种权限级别，以下安全规则始终生效：

| 规则 | 说明 |
|------|------|
| 沙箱外写入禁止 | `file_write` / `file_replace` 只能操作 `/sandbox/workspace/` 目录 |
| 敏感目录只读 | 禁止访问 `/etc`, `/root`, `/proc`, `.riffpad/` 等系统目录 |
| 命令白名单 | `bash_exec` 的 `rm -rf /`, `curl | bash` 等高危模式默认拦截 |
| 网络代理 | `search_web` / `fetch_url` 必须通过后端代理，沙箱不直连外网 |
| 端口限制 | `preview_start` 只能使用 3000-9000 范围内的端口 |
| 资源配额 | 单沙箱 CPU/内存/磁盘/网络受 cgroups 限制 |

##### 工具权限速查表

| 工具 | Safe | Confirm | Auto | 风险控制 |
|------|:----:|:-------:|:----:|----------|
| `file_read` / `file_list` / `file_glob` / `file_grep` | ✅ | ✅ | ✅ | 禁止访问敏感目录 |
| `file_write` / `file_replace` | ❌ | ⚠️ 确认 | ✅ | 沙箱外写入禁止；覆盖前快照 |
| `file_delete` | ❌ | ⚠️ 确认 | ✅ | 仅允许删除 workspace 内文件 |
| `read_media_file` | ❌ | ❌ | ✅ | 仅读取沙箱内媒体文件 |
| `bash_exec` | ❌ | ⚠️ 确认 | ✅ | 高危命令拦截 |
| `npm_install` / `pip_install` | ❌ | ⚠️ 确认 | ✅ | 仅允许白名单 registry |
| `preview_start` / `preview_stop` | ❌ | ⚠️ 确认 | ✅ | 端口限制 3000-9000 |
| `search_web` / `fetch_url` | ❌ | ⚠️ 确认 | ✅ | 后端代理，默认关闭 |
| `ask_user` | ✅ | ✅ | ✅ | 暂停 loop 等待回复 |
| `set_todo_list` | ❌ | ✅ | ✅ | 仅修改当前 Workspace 待办 |
| `task_list` / `task_output` | ✅ | ✅ | ✅ | 只能查看当前 Workspace 任务 |
| `task_stop` | ❌ | ⚠️ 确认 | ✅ | 只能停止当前 Workspace 任务 |

完整的 Agent 可用工具集为以上各类的集合：

```go
var AgentTools []openai.Tool
AgentTools = append(AgentTools, FileTools...)
AgentTools = append(AgentTools, SearchTools...)
AgentTools = append(AgentTools, ExecTools...)
AgentTools = append(AgentTools, PreviewTools...)
AgentTools = append(AgentTools, InteractionTools...)
AgentTools = append(AgentTools, NetworkTools...)
AgentTools = append(AgentTools, TaskTools...)
```

### 5.3 LLM Provider 抽象

为了支持多家国产模型 Provider 并便于切换，Agent 层不直接调用任何一家 SDK，而是依赖统一的 **OpenAI 兼容 Client**（`sashabaranov/go-openai`）：

```go
type LLMProvider string

const (
	ProviderVolcengine  LLMProvider = "volcengine"
	ProviderAliyun      LLMProvider = "aliyun"
	ProviderSiliconFlow LLMProvider = "siliconflow"
	ProviderDeepSeek    LLMProvider = "deepseek"
	ProviderCustom      LLMProvider = "custom"
)

type LLMConfig struct {
	Provider      LLMProvider
	BaseURL       string
	APIKey        string
	Model         string
	Compatibility string // standard | legacy
}

func NewLLMClient(cfg LLMConfig) *openai.Client {
	config := openai.DefaultConfig(cfg.APIKey)
	config.BaseURL = cfg.BaseURL
	return openai.NewClientWithConfig(config)
}
```

启动时从环境变量读取配置：

```bash
LLM_PROVIDER=volcengine
LLM_BASE_URL=https://ark.cn-beijing.volces.com/api/v3
LLM_API_KEY=sk-xxx
LLM_MODEL=doubao-pro-32k-xxx
```

切换模型只需改环境变量：

```bash
# 切到阿里通义千问
LLM_PROVIDER=aliyun
LLM_BASE_URL=https://dashscope.aliyuncs.com/compatible-mode/v1
LLM_API_KEY=sk-xxx
LLM_MODEL=qwen-max-xxx
```

调用示例：

```go
resp, err := llmClient.CreateChatCompletion(
	context.Background(),
	openai.ChatCompletionRequest{
		Model:    cfg.Model,
		Messages: messages,
		Tools:    agentTools,
	},
)
```

> **创业政策补贴：** 火山引擎、阿里云、百度智能云等均对初创团队有免费额度或代金券政策，建议早期优先申请，降低模型调用成本。

### 5.4 执行流程

1. 用户输入意图（文字或语音）。
2. Agent Orchestrator 组装上下文：系统提示 + 当前文件树 + 历史消息。
3. 通过 OpenAI 兼容接口调用当前配置的国产大模型。
4. LLM 返回 Thought + Tool Calls（统一解析为内部标准格式）。
5. 串行或并行执行工具（文件写操作默认串行，Bash 独立执行）。
6. 工具结果回传给 LLM，直到 LLM 给出最终回复。
7. 前端通过 WebSocket 实时同步：文件变更、日志流、预览状态。

---

## 6. Agent 与 Chat 系统详细设计

本节详细定义 Riffpad 中 Agent 的交互模型、Chat 配置、上下文管理、以及 Agent Loop 的控制策略。

### 6.1 Chat 配置模型

每个 Workspace 关联一组 Chat 配置，用户可在设置面板或输入框附近调整。

```go
type ChatSettings struct {
    WorkspaceID     string           `json:"workspaceId"`
    ModelID         string           `json:"modelId"`         // 默认模型
    Temperature     float32          `json:"temperature"`     // 默认 0.4
    TopP            float32          `json:"topP"`            // 默认 0.95
    TopK            int              `json:"topK"`            // 默认 50
    MaxTokens       int              `json:"maxTokens"`       // 默认 4096
    ShowThinking    bool             `json:"showThinking"`    // 是否显示模型 reasoning
    ShowToolCalls   bool             `json:"showToolCalls"`   // 是否显示工具调用过程
    AutoRunCommands bool             `json:"autoRunCommands"` // 是否自动执行 Bash
}
```

MVP 阶段不向用户暴露所有滑块，提供三种预设模式：

| 模式 | Temperature | TopP | 适用场景 |
|------|-------------|------|----------|
| **编码模式** | 0.3 | 0.90 | 写代码、修 Bug，稳定可预测 |
| **头脑风暴** | 0.8 | 0.95 | 发散想法、探索方案 |
| **重构模式** | 0.2 | 0.90 | 整理已有代码、生成规范骨架 |

### 6.2 模型切换（Model Toggle）

支持 Workspace 级别默认模型 + 单次消息临时切换。

```go
type ModelPreset struct {
    ID          string `json:"id"`
    Provider    string `json:"provider"`    // volcengine | aliyun | deepseek ...
    Name        string `json:"name"`        // 展示名，如"豆包 Pro 32K"
    Description string `json:"description"` // 简短说明
    SupportsFC  bool   `json:"supportsFc"`  // 是否支持 Function Calling
}
```

切换策略：
- 默认从 Workspace 的 `ChatSettings.ModelID` 读取
- 用户可在某条消息选择"使用其他模型回答"
- 切换模型**不重置**上下文，所有 OpenAI 兼容模型共享 messages 格式
- 若目标模型不支持 Function Calling，则禁用 Agent 工具调用，降级为普通聊天

### 6.3 Thinking 显示策略

Agent 的思考过程分两层展示：

| 层级 | 内容 | 默认状态 |
|------|------|----------|
| **执行过程** | Tool Calls、文件操作、Bash 命令、预览启动 | 默认显示关键步骤 |
| **模型 Reasoning** | DeepSeek-R1 / o1 等模型的 `<think>` 内容 | 默认折叠，可展开 |

前端事件类型：

```go
type AgentEvent struct {
    Type    string      `json:"type"`    // thinking | tool_call | tool_result | file_change | preview_update | message | error
    Content string      `json:"content,omitempty"`
    Name    string      `json:"name,omitempty"`
    Args    interface{} `json:"args,omitempty"`
    Result  interface{} `json:"result,omitempty"`
    Path    string      `json:"path,omitempty"`
    URL     string      `json:"url,omitempty"`
}
```

当 `Type == "thinking"` 时，内容即为模型 reasoning 或 Agent 内部思考，按用户偏好决定是否渲染。

### 6.4 Chat History 存储

每个 Workspace 只允许一个连续的 Chat 线程，所有消息按时间顺序存储。

```go
type Message struct {
    ID          string    `gorm:"primaryKey"`
    WorkspaceID string    `gorm:"index"`
    Role        string    // system | user | assistant | tool
    Content     string    `gorm:"type:text"`
    ModelID     string    // 生成该消息的模型
    ToolCalls   JSON      // 工具调用记录
    ToolResults JSON      // 工具执行结果
    Metadata    JSON      // tokens, latency, temperature 等
    CreatedAt   time.Time
}
```

存储策略：
- 用户消息、Assistant 回复、Tool 调用记录均永久保留
- 系统提示不存库，每次请求重新组装
- 关键操作节点自动触发 Snapshot，保存当时的文件状态

### 6.5 上下文管理

Riffpad 的上下文由以下部分组成：

```
context = systemPrompt
        + conversationSummary  // 早期对话摘要
        + recentMessages(N)    // 最近 N 条完整消息
        + currentFileTree()    // 当前文件树与关键文件内容
```

#### 上下文满的处理策略

当总 token 接近模型上限时，采用**摘要 + 当前文件状态**组合策略：

1. 对早期 50% 的消息生成一段摘要
2. 保留最近 50% 的完整消息
3. 始终注入当前完整文件树内容
4. 摘要作为额外的 system message 注入

```go
type ConversationSummary struct {
    WorkspaceID   string
    Summary       string // "用户想做一个倒计时组件，已生成 index.html 和 main.js，下一步希望加入番茄钟功能"
    UpToMessageID string
}
```

为什么优先保留文件状态？因为代码生成场景中，**"现在文件长什么样"比"之前聊了什么"更重要**。

### 6.6 Agent Loop 策略

基础循环采用 ReAct（Reasoning + Acting）模式：

```
1. 用户输入进入 AgentService
2. 组装上下文（system + summary + recent messages + files）
3. 调用 LLM（流式）
4. 解析响应：
   ├─ 若含 tool_calls → 执行工具 → 将结果回传 → 回到步骤 3
   └─ 若是最终回复 → 返回给用户
5. 循环直到：
   ├─ 得到最终回复
   ├─ 达到最大步数 MaxSteps
   └─ 发生不可恢复错误或超时
```

控制参数：

```go
type AgentLoopConfig struct {
    MaxSteps             int           // 默认 15，防止死循环
    MaxToolCallsPerStep  int           // 单轮最大工具调用数，默认 5
    SingleCommandTimeout time.Duration // Bash 命令超时，默认 30s
    TotalTimeout         time.Duration // 整个 loop 超时，默认 5min
    AutoRunCommands      bool          // 是否自动执行 Bash
    RequireConfirm       []string      // 需要用户确认的工具，如 ["bash_exec"]
}
```

### 6.7 安全与异常处理

| 场景 | 检测方式 | 处理方式 |
|------|---------|----------|
| 死循环 | 同一工具连续调用超过 3 次 | 强制终止，返回已执行结果 |
| Bash 超时 | context.WithTimeout | 取消命令，标记错误，回传 LLM |
| 沙箱无响应 | 心跳检测 | 标记 unhealthy，从预热池分配新沙箱 |
| LLM 格式错误 | JSON 解析失败 | 重试 1-2 次，仍失败则提示用户 |
| 文件冲突 | 并发写检测 | 文件写操作串行化执行 |
| 资源超限 | cgroups 监控 | 终止超限沙箱，通知用户 |

### 6.8 并发与批量优化

- **文件写操作**：收集连续的 `file_write` tool calls，合并为一次沙箱 BatchWrite
- **读操作**：可并行执行
- **Bash 命令**：串行执行，避免沙箱内状态冲突
- **npm install 等长耗时命令**：异步执行，通过 WebSocket 推送进度事件

### 6.9 Chat 与 Workspace 的关系

- 一个 Workspace 只对应一个 Chat 线程
- 用户想换方向时，点击 **New Spark** 创建新 Workspace
- 提供两个进阶功能（二期）：
  - **清空对话历史**：保留文件，仅清空消息
  - **复制 Workspace**：基于当前文件状态创建新的 Workspace

---

## 7. 预览系统

### 7.1 预览模式

| 类型 | 触发方式 | 说明 |
|------|----------|------|
| **静态 HTML** | Agent 生成 `index.html` | 直接通过 Web Server 代理 |
| **单文件 React/Vue** | Agent 生成 `.jsx/.vue` + 配置 | 沙箱内启动 Vite dev server |
| **Python 可视化** | Agent 运行 Python 输出 | 截图/生成 HTML 后代理 |
| **Mock API** | Agent 启动 json-server | 代理 API 路由供前端画布调用 |

### 7.2 预览代理

```
User Canvas
    │
    ▼
https://preview.riffpad.app/p/{workspace-token}/
    │
    ▼
Preview Proxy (Nginx / Custom Gateway)
    │
    ▼
Sandbox Web Server (http://sandbox-ip:{port})
```

- 每个工作区分配唯一 preview token。
- token 与 workspace id 映射存储在 **Upstash Redis**。
- 沙箱内 Web Server 默认监听 3000/8080/5173 等端口。

---

## 8. 导出与下游桥接

### 8.1 导出流程

1. 用户点击「推向开发」。
2. Agent 读取当前沙箱全部文件，执行一次内部重构：
   - 重命名无意义文件
   - 规范化目录结构
   - 添加基础错误处理
   - 生成 `package.json` / `requirements.txt`
3. 生成 `README.md`，强制包含：
   - 产品核心逻辑（What & Why）
   - 已验证的技术路径
   - 第三方 API/依赖说明
   - 启动命令
   - 推荐的 AI 提示词上下文
4. 打包为 Zip 或推送到临时 GitHub Repo。

### 8.2 GitHub 推送

- 使用 GitHub App / OAuth 授权。
- 创建私有仓库，默认命名为 `riffpad-{slug}`。
- 写入代码 + README + `.cursorrules`（可选，用于 Cursor 继承上下文）。

### 8.3 导出产物示例

```
my-spark/
├── README.md
├── package.json
├── src/
│   ├── index.html
│   ├── main.js
│   └── style.css
└── .cursorrules
```

---

## 9. 安全设计

### 9.1 沙箱隔离

| 层级 | 措施 |
|------|------|
| **虚拟化** | Firecracker microVM，每个沙箱独立内核 |
| **网络** | 默认禁止出网；白名单放行 npm/pip registry |
| **文件系统** | OverlayFS，只读 base image + 可写层 |
| **资源** | cgroups 限制 CPU/内存/磁盘/网络 |
| **时间** | 单条 Bash 命令默认最大 30s，可配置 |

### 9.2 内容安全

- 用户上传文件大小限制：单文件 1MB，总项目 50MB。
- 禁止沙箱内访问内部元数据服务。
- LLM 输入/输出需做基础过滤，防止敏感信息泄露。

### 9.3 权限模型

#### 用户级别权限

认证由 **Supabase Auth** 提供，Go backend 通过校验 Supabase JWT 识别用户身份。

| 用户类型 | 来源 | 权限 |
|----------|------|------|
| **匿名用户** | Supabase Auth 匿名登录 | 仅可创建 3 个临时工作区，7 天后自动清理 |
| **登录用户** | Supabase Auth 邮箱/OAuth | 可长期保存、导出、GitHub 推送、调整 Agent 权限档位 |

> 应用层权限控制通过 `User` 表的 `role` 字段扩展（如 `free`, `pro`），由 Go backend 在 JWT 校验后读取。

#### Agent 权限档位

每个 Workspace 可独立设置 Agent 权限级别，详见 5.2.8 节：

| 档位 | 说明 |
|------|------|
| **Safe** | 只读模式，Agent 不会修改任何文件或执行命令 |
| **Confirm** | 默认模式，副作用操作批量请求用户确认 |
| **Auto** | 自动模式，Agent 可自主调用全部工具 |

建议新用户首次使用默认 **Confirm** 模式，建立信任后再切换到 **Auto**。

---

## 10. 生命周期与成本管理

### 10.1 状态流转

```
[创建] → WARM（运行中）
         │
         │ 48h 无活动 & 未 pinned
         ▼
      HIBERNATING（已休眠）
         │
         │ 用户重新打开 / Agent 操作
         ▼
      WARM（恢复，<3s）
         │
         │ 用户删除 / 超过 30 天未访问
         ▼
      DELETED（清理对象存储快照）
```

### 10.2 休眠实现

1. 定时任务扫描 `lastActiveAt < now() - 48h` 且 `isPinned = false` 的工作区。
2. 将文件系统打包为 tar.gz，上传对象存储。
3. 将数据库中 `File` 表保留为结构化快照（路径 + 摘要 + 存储 key）。
4. 销毁沙箱实例，释放计算资源。
5. 唤醒时：从预热池分配新沙箱，下载快照恢复文件。

### 10.3 预热池

- 后台维护 N 个已启动但未绑定的沙箱实例。
- 用户点击 New Spark 时直接分配，实现 <1s 启动。
- 沙箱集群池不足时触发扩容，使用 Kubernetes HPA（Railway/Fly.io 后端无需 HPA）。

---

## 11. API 设计（核心接口）

### 11.1 接口形式

- **REST API**：用于状态变更、文件 CRUD、导出等请求-响应式操作。
- **WebSocket**：用于 Agent 事件流、文件实时同步、沙箱日志推送等长连接场景。

### 11.2 REST 路由设计

```go
// Workspace
c.POST  ("/api/v1/workspaces",              handler.CreateWorkspace)
c.GET   ("/api/v1/workspaces",              handler.ListWorkspaces)
c.GET   ("/api/v1/workspaces/:id",          handler.GetWorkspace)
c.POST  ("/api/v1/workspaces/:id/wake",     handler.WakeWorkspace)
c.DELETE("/api/v1/workspaces/:id",          handler.DeleteWorkspace)

// File
c.GET   ("/api/v1/workspaces/:id/files",    handler.ListFiles)
c.GET   ("/api/v1/workspaces/:id/files/*",  handler.ReadFile)
c.POST  ("/api/v1/workspaces/:id/files",    handler.WriteFile)
c.DELETE("/api/v1/workspaces/:id/files/*",  handler.DeleteFile)

// Agent
c.POST  ("/api/v1/workspaces/:id/agent/chat", handler.SendAgentMessage)

// Preview
c.POST  ("/api/v1/workspaces/:id/preview", handler.StartPreview)
c.DELETE("/api/v1/workspaces/:id/preview", handler.StopPreview)

// Export
c.POST  ("/api/v1/workspaces/:id/export/zip",    handler.ExportZip)
c.POST  ("/api/v1/workspaces/:id/export/github", handler.ExportGitHub)
```

### 11.3 WebSocket 事件

```go
// 连接时加入 workspace room
c.GET("/ws/workspaces/:id", handler.AgentWebSocket)
```

事件类型：

```go
type AgentEvent struct {
	Type    string      `json:"type"`    // thinking | tool_call | tool_result | file_change | preview_update | message | error
	Content string      `json:"content,omitempty"`
	Name    string      `json:"name,omitempty"`
	Args    interface{} `json:"args,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Path    string      `json:"path,omitempty"`
	URL     string      `json:"url,omitempty"`
}
```

---

## 12. 部署架构

### 12.1 生产环境

```
┌─────────────────────────────────────────────────────────────────────┐
│                         Vercel                                      │
│  ┌─────────────────────┐  ┌─────────────────────┐                  │
│  │  riffpad.ai         │  │  app.riffpad.ai     │                  │
│  │  (营销站 + Docs)    │  │  (应用前端 PWA)     │                  │
│  │  apps/landing       │  │  apps/app           │                  │
│  └─────────────────────┘  └──────────┬──────────┘                  │
└──────────────────────────────────────┼──────────────────────────────┘
                                       │ HTTPS / WebSocket
                                       ▼
┌─────────────────────────────────────────────────────────────────────┐
│              Go Backend Service (Railway / Fly.io)                  │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │  API Gateway + WebSocket + Agent Orchestrator + Sandbox     │   │
│  │  Scheduler + File Sync + Preview Proxy + Export Service     │   │
│  └─────────────────────────────────────────────────────────────┘   │
└──────────────────────────────────┬──────────────────────────────────┘
                                   │
          ┌────────────────────────┼────────────────────────┐
          │                        │                        │
          ▼                        ▼                        ▼
┌─────────────────┐  ┌─────────────────────────┐  ┌─────────────────┐
│   Supabase      │  │      Upstash Redis      │  │  LLM Providers  │
│  ├─ Postgres    │  │  (会话 / 预热池索引      │  │  (Volcengine/   │
│  ├─ Auth        │  │   / WebSocket room)     │  │   Aliyun/       │
│  └─ Storage     │  │                         │  │   DeepSeek)     │
└─────────────────┘  └─────────────────────────┘  └─────────────────┘
                                   │
                                   │ gRPC / HTTP
                                   ▼
┌─────────────────────────────────────────────────────────────────────┐
│              Sandbox Cluster (Kubernetes / 裸金属)                   │
│  ┌─────────────────────┐  ┌─────────────────────────────────────┐  │
│  │  Firecracker /      │  │  Sandbox Worker + Lifecycle Worker  │  │
│  │  Docker + gVisor    │  │  (长运行，维护预热池与休眠唤醒)        │  │
│  └─────────────────────┘  └─────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────┘
```

### 12.2 各站点部署平台

| 站点 | 域名 | 所属 App | 推荐平台 | 说明 |
|------|------|----------|----------|------|
| **营销站** | `riffpad.ai` | `apps/landing` | Vercel / Cloudflare Pages | 静态导出，SEO 友好，全球 CDN |
| **应用前端** | `app.riffpad.ai` | `apps/app` | Vercel + Edge | PWA、需要动态渲染、靠近后端 |
| **后端 API** | `api.riffpad.ai` | `apps/api` | **Railway / Fly.io** | 长运行容器，处理 WebSocket、沙箱调度 |
| **数据库/Auth/存储** | - | - | **Supabase** | 托管 Postgres + Auth + Storage |
| **缓存** | - | - | **Upstash Redis** | Serverless Redis，会话与预热池索引 |
| **沙箱集群** | - | `packages/sandbox-images` | **Kubernetes / Hetzner 裸金属** | 仅沙箱需要专用计算资源 |

> 营销站和应用前端都使用 Next.js，但部署目标不同：营销站强调静态生成和 SEO，应用前端强调实时交互和 PWA 体验。
> 
> **为什么后端不用 Vercel Serverless Functions？** WebSocket 长连接、沙箱调度、预热池维护都需要长运行进程，不适合纯 serverless。Railway / Fly.io 提供长运行容器且运维简单，是早期最佳折中。

### 12.3 本地开发

本地开发采用 **混合模式**：沙箱用 Docker Compose 本地启动，数据库/Auth/缓存可用本地容器或直连 Supabase 项目。

```bash
# 启动本地沙箱基础设施
docker compose -f infra/docker-compose.yml up -d

# 启动 Go 后端
cd apps/api && go run ./cmd/api

# 启动应用前端（app.riffpad.ai）
cd apps/app && pnpm dev

# 启动营销站点（riffpad.ai）
cd apps/landing && pnpm dev
```

**方案 A：完全本地开发（无需 Supabase 账号）**

`infra/docker-compose.yml` 包含：
- PostgreSQL（端口 5432）
- Redis（端口 6379）
- MinIO（端口 9000/9001）
- Sandbox Worker（开发期用 Docker 简化）

> 此时 Auth 使用简化的本地 JWT 方案，仅用于开发测试。

**方案 B：连接 Supabase 开发环境（推荐）**

配置环境变量（详见下方完整 `.env.example`）：

```bash
SUPABASE_URL=https://your-project.supabase.co
SUPABASE_ANON_KEY=xxx
SUPABASE_SERVICE_ROLE_KEY=xxx
DATABASE_URL=postgresql://postgres:[password]@db.your-project.supabase.co:5432/postgres
REDIS_URL=rediss://default:[password]@your-redis.upstash.io
S3_ENDPOINT=https://your-project.supabase.co/storage/v1/s3
S3_BUCKET=riffpad-snapshots
```

> 开发期使用 Supabase 的好处：与生产环境一致，避免本地/生产行为差异。

前端通过 `NEXT_PUBLIC_API_URL=http://localhost:8080` 调用后端。

### 12.4 环境变量配置（`.env.example`）

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

### 12.5 常用命令（Makefile）

```makefile
# Makefile 示例
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

---

## 13. 关键技术指标与验收

| 指标 | 目标 | 验证方式 |
|------|------|----------|
| New Spark 启动时间 | P95 < 800ms | 预热池分配 + API 调用计时 |
| 休眠唤醒时间 | < 3s | 沙箱恢复 + 文件下载计时 |
| 单沙箱 Bash 执行隔离 | 无法访问宿主机 / 内部服务 | 安全渗透测试 |
| 导出产物可用性 | Cursor 打开即可运行 | 端到端测试 |
| 移动端语音输入 | 支持主流浏览器 | 真机测试 |

---

## 14. 风险与待决策项

| 风险 | 影响 | 缓解方案 |
|------|------|----------|
| Firecracker 运维复杂度高 | 早期开发成本高 | 开发期用 Docker+gVisor，稳定后迁移；后端本身不上 K8s |
| LLM Tool Calling 可靠性 | Agent 误操作文件 | 增加操作确认、沙箱快照回滚 |
| 成本控制 | 沙箱运行消耗资源 | 严格的配额、自动休眠、匿名用户限制 |
| 预览代理延迟 | 用户体验差 | CDN + 边缘节点就近代理 |
| GitHub 授权复杂度 | 用户流失 | 同时提供 Zip 下载作为零授权替代 |
| **Supabase vendor lock-in** | 未来迁移成本 | Postgres 是标准的，Auth/Storage 可用 Clerk/S3 替代 |
| **Serverless Redis 成本** | Upstash 按请求计费，高频场景可能较贵 | 早期够用，规模扩大后迁移到自管 Redis |

### 待决策

1. 是否采用 WebContainer 技术在浏览器端跑沙箱？（可进一步降低服务器成本，但生态有限。）
2. 是否支持多语言 Agent（Python/Node/Go）？首期建议聚焦 Node + Python。
3. 是否提供付费订阅解锁更多沙箱资源与长期保存？
4. **后端长运行容器平台选择**：Railway vs Fly.io vs Render？（Railway 更简单，Fly.io 全球边缘更好。）
5. **Supabase 还是 Clerk + 自管 Postgres？** Supabase 一体化更简单，Clerk 认证更专业但需单独接 Postgres。

---

## 15. 下一阶段任务拆分

### 阶段一：搭建 Monorepo 骨架

1. **初始化仓库结构**
   - 创建 `apps/`、`packages/`、`infra/`、`docs/` 目录
   - 根目录添加 `Makefile`、`.gitignore`、README

2. **应用前端 Next.js 项目（`apps/app/`）**
   - 创建项目，配置 Tailwind + shadcn/ui
   - 实现 New Spark 首页输入框
   - 配置 PWA 基础
   - 绑定 `app.riffpad.ai` 域名逻辑

3. **后端 Go 项目（`apps/api/`）**
   - 初始化 Go module，选择 Echo 或 Fiber
   - 配置 **Supabase Postgres** + GORM + **Upstash Redis**
   - 集成 **Supabase Auth**（邮箱/OAuth/匿名用户），Go backend 只校验 JWT
   - 实现 Workspace CRUD

4. **营销站点 Next.js 项目（`apps/landing/`）**
   - 创建项目，配置 Tailwind + 动画库
   - 实现 riffpad.ai 首页
   - 实现 `/pricing` 价格页
   - 实现 `/docs` 文档页（早期可先用 Markdown/MDX）
   - 配置静态导出和 SEO

5. **共享协议（`packages/shared-types/`）**
   - 定义 OpenAPI 3.0 规范
   - 前端生成 TypeScript 类型，后端生成 Go 类型（可选）

6. **本地基础设施（`infra/docker-compose.yml`）**
   - 简化沙箱（必选）
   - 可选本地 PostgreSQL / Redis / MinIO（用于完全离线开发）

7. **前后端联调**
   - 配置 CORS
   - 前端通过 REST 调用后端
   - 统一通过 Makefile 一键启动

### 阶段二：沙箱与 Agent

8. **实现简化沙箱**
   - Docker 沙箱（开发期）
   - 文件读写 API
   - Bash 执行 API

9. **接入 LLM Agent**
   - `sashabaranov/go-openai` 集成国产模型
   - Function Calling 工具定义
   - WebSocket 实时同步 Agent 事件

### 阶段三：核心体验

10. **实现预览画布**
    - HTML/JS 预览代理
    - 右侧预览窗切换

11. **实现导出功能**
    - Zip 打包
    - README 自动生成
    - GitHub 推送（二期）

12. **Mobile-First UI**
    - 三栏/单栏响应式布局
    - 语音输入集成

### 阶段四：生命周期与成本

13. **生命周期与休眠**
    - 快照机制
    - 自动休眠任务
    - 唤醒恢复

---

*本 TSD 为 v0.2 版本，随着实现推进可迭代细化。*
