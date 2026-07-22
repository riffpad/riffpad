# 多 Workspace 与持久化设计

> 状态：设计稿（design doc），Phase 1 实现中。
> 参考：TSD §4 数据模型、§7 环境配置。
> 术语：统一叫 **workspace**，没有 project 概念；前端 UI 文案也只使用 workspace。

## 1. 现状盘点

当前实现与 TSD 的差距：

| 方面 | 现状 | TSD 规划 |
|------|------|----------|
| Workspace 元数据 | `WorkspaceManager` 内存 map，重启即丢 | Postgres（Supabase） |
| 消息历史 | 内存 map，重启即丢 | Postgres `Message` 表，永久保留 |
| 文件 | LocalSandbox 落在 `/tmp/riffpad-sandbox/{id}`，进程重启后目录还在但无索引，无法恢复 | File Sync + Snapshot 到对象存储 |
| Workspace 列表 | 无；前端每次自动创建新 workspace | 每个用户可管理多个 workspace |
| Auth | `ownerID = "anonymous"` 硬编码 | Supabase Auth |
| Postgres / Redis / MinIO | `.env`、docker-compose 均已配置，但 API 从未连接 | 核心基础设施 |

结论：多 Workspace 的前提就是**数据要落库**。没有持久化，列表、会话恢复、沙箱休眠都无从谈起。

## 2. 是否应该现在接入数据库？

**应该，而且只做最小的一步。**

理由：

1. 多 Workspace 的本质是"按 owner 查询 workspace 列表"——这是一个数据库查询，内存结构做不了。
2. 切回旧 workspace 时能恢复聊天记录——消息必须持久化。
3. TSD §4 已经定义了完整的 schema（User / Workspace / File / Message / Snapshot），方向早已定好，不需要重新讨论。
4. `gorm.io/gorm` 已经在 `apps/api/go.mod` 依赖里，接入成本很低。
5. 越晚接，内存层的临时逻辑越多，迁移越痛。

## 3. 概念澄清

**不做 Project 实体，也不做双实体设计。** 唯一的核心实体就是 `workspace`：

- 后端、API、数据库、前端 UI 全部叫 `workspace`。
- 未来如确需"一个 workspace 包含多个环境/分支"的概念，再加聚合层，现在不预先抽象。

## 4. 分阶段方案

### Phase 1：Postgres 持久化 + workspace 列表（MVP）

目标：重启 API 不丢 workspace；用户能看到自己的 workspace 列表，点进去能继续之前的对话。

**数据模型（TSD §4 的子集）**

```go
// workspaces
type Workspace struct {
    ID           string    `gorm:"primaryKey"`
    Slug         string    `gorm:"uniqueIndex"`
    Name         *string
    OwnerID      string    `gorm:"index"`
    Status       string    // WARM | HIBERNATING | ARCHIVED
    LastActiveAt time.Time
    CreatedAt    time.Time
}

// messages
type Message struct {
    ID          string    `gorm:"primaryKey"`
    WorkspaceID string    `gorm:"index"`
    Seq         int       // 会话内单调递增序号，保证回放顺序
    Role        string    // user | assistant | tool
    Content     string    `gorm:"type:text"`
    ToolCalls   []byte    `gorm:"type:jsonb"` // agent.ToolCall 序列化，可空
    ToolCallID  string
    Name        string
    Metadata    []byte    `gorm:"type:jsonb"`
    CreatedAt   time.Time
}
```

说明：

- `Seq` 字段：回放历史时按 `workspace_id, seq` 排序，避免依赖 created_at 精度。
- 消息持久化时机：每轮 agent run 结束后，把**新增的 messages**（run 前后的 diff）批量插入。不做逐条事件持久化，简单可靠。
- 文件**不进数据库**：Phase 1 仍依赖本地沙箱目录。

**代码结构**

```
apps/api/internal/
├── domain/          # Workspace / Message 实体（GORM tags）
├── repository/      # GORM 实现：WorkspaceRepo / MessageRepo
└── service/
    └── workspace_manager.go  # 变成 facade：DB 负责状态，内存只持有活跃沙箱
```

`WorkspaceManager` 职责收窄：

- Create/Get/List → 走 repository（写 DB）。
- Sandbox 映射仍是内存 map（活跃沙箱才需要）；进程重启后按需为已有 workspace 重建 LocalSandbox（`/tmp` 目录存在即可复用）。

**API 变更**

- 新增 `GET /api/v1/workspaces`（按 owner 列表，按 lastActiveAt 倒序）。
- 新增 `GET /api/v1/workspaces/:id/messages`（按 seq 升序回放完整历史）。
- 前端路由：`/` = workspace 列表页，`/w/:id` = 工作区视图；打开工作区先拉 messages，再连 WS。

**迁移策略**

- 开发期用 `db.AutoMigrate(&domain.Workspace{}, &domain.Message{})` 启动时自动建表，保持简单。
- 进入生产前再引入 golang-migrate 之类的正式 migration 工具（记录在案，不在本期做）。

**本地 Postgres 端口冲突**

compose 里的 Postgres 会占用 5432，与本机系统 Postgres 冲突。本期建议：

- 优先用本机已有的 Postgres，创建 `riffpad` 库，`.env` 的 `DATABASE_URL` 已指向它；
- 或者把 compose 的 Postgres 映射改为 `5433:5432` 并同步 `.env`。
- compose 继续只用于 Redis / MinIO / sandbox-worker。

### Phase 2：文件快照与沙箱恢复

- 休眠/退出时把沙箱目录打包上传到 MinIO（`S3_BUCKET=riffpad-snapshots`），DB 记录 `Snapshot` 行。
- 打开已休眠 workspace 时：从 MinIO 恢复快照 → 重建沙箱 → status 转 WARM。
- 之后才能真正说"workspace 可长期保存"。

### Phase 3：Auth 与 Redis

- 接入 Supabase Auth，ownerID 从 JWT 解析，替换 `anonymous`。
- Redis 用于：会话/presence、沙箱索引、热点缓存（workspace 列表）。本期不依赖。

## 5. 多 Workspace 的 UX 取舍（本期）

- 首页 = workspace 列表（空态引导"新建 workspace"）；工作区页 = 当前三栏 IDE。
- 每个 workspace 只有一条连续 Chat 线程（TSD 已定，不做多线程）。
- workspace 可删除（软删除先不做，直接删 workspace + messages + 沙箱目录）。重命名不在本期。
- 移动端：列表页同样可用，遵循 AGENTS.md 的 mobile-first。

## 6. 验收标准（Phase 1）

- [ ] 新建 workspace 后，`POST /api/v1/workspaces` 的结果能在 DB 中查到。
- [ ] 重启 API server 后，`GET /api/v1/workspaces` 仍列出 workspace。
- [ ] 打开旧 workspace，聊天记录完整恢复（含 tool 消息），可继续对话，上下文正确（compaction 逻辑基于 DB 回放的消息）。
- [ ] 前端列表页可创建、进入、删除 workspace。
- [ ] `make dev` / `make dev-local` 文档更新，说明 Postgres 依赖与端口冲突处理。
- [ ] TSD §4 数据模型小节同步：标注哪些字段是本期实现的子集。

估算：M（1–2 天）。Phase 2/3 另开 issue。

## 7. 风险与备注

- **消息与沙箱漂移**：消息在 DB、文件在 /tmp，重启后可能消息恢复了但文件丢了（例如 /tmp 被清理）。Phase 1 接受这个风险并在列表页展示提示；Phase 2 用快照解决。
- **并发**：同一 workspace 的 agent run 已用 `runMu` 串行化；多用户场景未来再考虑行级锁/版本号。
- **ownerID 隔离**：当前所有 workspace 都属于 `anonymous`，接入 Auth 时需要做一次匿名数据迁移（本期不管）。
