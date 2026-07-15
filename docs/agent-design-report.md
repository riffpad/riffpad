# Riffpad Agent Loop 设计报告

> 参考仓库：[earendil-works/pi](https://github.com/earendil-works/pi) 的 `packages/agent` 与 `packages/ai`
> 目标：为 Riffpad 设计一个极简、可扩展、事件驱动的 Agent Loop

---

## 1. 从 pi 学到的核心设计思想

### 1.1 AgentMessage 与 LLM Message 分离

pi 在内部使用 `AgentMessage`，只在调用 LLM 前通过 `convertToLlm()` 转换成 provider 能理解的 `Message[]`。

```
AgentMessage[] → transformContext() → AgentMessage[] → convertToLlm() → Message[] → LLM
```

这样可以在 transcript 中保留 UI 消息、状态标记、自定义元数据，而不会污染模型上下文。

**对 Riffpad 的启示：**
- 数据库 `messages` 表存储 `AgentMessage` 的 JSON 表示。
- 调用 OpenAI 兼容接口前，再转换成标准 messages/tools 格式。
- 未来可以扩展 `snapshot`、`preview_url` 等自定义消息类型，无需改 LLM 调用层。

### 1.2 细粒度事件流

pi 的 loop  emit 完整生命周期事件：

```
agent_start → turn_start → message_start → message_update → message_end
→ tool_execution_start → tool_execution_update → tool_execution_end
→ turn_end → agent_end
```

**对 Riffpad 的启示：**
- 这些事件天然映射到 TSD 中已定义的 `AgentEvent` 类型。
- 后端通过 WebSocket 转发事件；前端订阅并实时更新文件树、日志、预览画布。
- 事件是单向数据流，便于回放、调试、测试。

### 1.3 工具执行模式

pi 支持 `parallel` 与 `sequential`，并允许单个工具覆盖执行模式：

- `parallel`：适合独立的 `file_read` / `file_list`。
- `sequential`：适合有副作用的 `file_write` → `bash_exec` 链。

**对 Riffpad 的启示：**
- MVP 阶段文件写操作默认串行，避免并发写冲突。
- 读操作可以并行，提升响应速度。
- `bash_exec` 必须串行，因为沙箱内状态会互相影响。

### 1.4 beforeToolCall / afterToolCall 钩子

pi 在参数校验后、执行前提供 `beforeToolCall`，可用于权限拦截；执行后提供 `afterToolCall`，可用于审计、改写结果、触发终止。

**对 Riffpad 的启示：**
- `beforeToolCall` 直接对应 TSD 中的 Confirm / Safe / Auto 权限档位。
- `afterToolCall` 可用于：
  - 自动创建快照（snapshot_create）
  - 记录文件变更事件
  - 统计 token / 耗时

### 1.5 可插拔的 ExecutionEnv

pi 把文件系统与命令执行抽象成 `ExecutionEnv`（`FileSystem + Shell`），本地实现是 `NodeExecutionEnv`，未来可替换为 Docker / SSH / 浏览器等。

**对 Riffpad 的启示：**
- 这是 TSD 中“先做 Agent loop，再补 Docker 沙箱”策略的理论基础。
- 先实现 `LocalSandbox`（限定在 `/tmp/riffpad-sandbox-{id}`），再实现 `DockerSandbox`。
- Agent 代码只依赖接口，不依赖实现。

### 1.6 Result<T, E> 与错误处理

pi 的工具和文件操作返回 `Result<T, E>`，预期失败不会抛异常；只有真正的运行时异常才 throw。错误会被包装成 tool result 回传给 LLM。

**对 Riffpad 的启示：**
- Go 后端可以沿用 Go 的 `(T, error)` 模式。
- 工具执行失败时，把错误信息作为 `tool_result` 内容返回给 LLM，让模型自己决定重试或换方案。

### 1.7 AbortSignal 全程传递

pi 从 loop 到 provider stream 到 tool execute 都传递 `AbortSignal`，支持用户中途取消。

**对 Riffpad 的启示：**
- WebSocket 断开时触发 `AbortController.abort()`。
- 长耗时命令（如 `npm install`）需要监听 signal 并清理子进程。

### 1.8 transformContext 做上下文压缩

pi 在每次 LLM 调用前允许 `transformContext()`，用于裁剪旧消息、注入外部上下文。

**对 Riffpad 的启示：**
- 实现上下文压缩策略：保留最近 N 条消息 + 当前完整文件树 + 早期摘要。
- 这是 TSD §6.5 中“摘要 + 当前文件状态”组合策略的落地位置。

---

## 2. Riffpad Agent Loop 设计原则

基于 pi 的思想和 Riffpad 的 TSD，提出以下设计原则：

### 原则 1：接口优先

先定义 `Sandbox` 接口，再实现 `LocalSandbox`。Agent Loop 只依赖 `Sandbox` 接口。

```go
type Sandbox interface {
    ID() string
    ReadFile(path string) (string, error)
    WriteFile(path, content string) error
    ListFiles(dir string) ([]FileInfo, error)
    Exec(command string, opts ExecOptions) (ExecResult, error)
    Destroy() error
}
```

### 原则 2：事件驱动

Agent Loop 不直接写 WebSocket，而是把事件写到 channel / callback。handler 层负责把事件转发到 WebSocket room。

```go
type AgentEvent struct {
    Type    string
    Content string
    Name    string
    Args    any
    Result  any
    Path    string
    URL     string
}
```

### 原则 3：流式响应

Assistant 的文本回复通过 SSE / WebSocket chunk 实时推送，而不是等完整响应再返回。

### 原则 4：权限钩子

通过 `beforeToolCall` 实现 Confirm 模式的批量确认：

1. Agent 一轮返回多个 tool call。
2. `beforeToolCall` 收集所有副作用操作，生成确认请求。
3. 前端用户确认后，才真正执行；取消则回滚快照。

### 原则 5：最小工具集

MVP 只暴露 4 个工具：

- `file_read`
- `file_write`
- `file_list`
- `bash_exec`

后续再按 TSD 扩展 `file_replace`、`file_delete`、`preview_start` 等。

### 原则 6：工具错误即反馈

工具执行失败不中断 loop，而是把错误包装成 `tool_result` 返回给 LLM，让模型继续决策。

### 原则 7：可取消

每个 `prompt` / `continue` 运行都绑定 `context.Context`（或 `AbortSignal` 的 Go 等价物），用户可随时中止。

---

## 3. 建议的模块划分

```
apps/api/internal/
├── agent/
│   ├── loop.go           # AgentLoop 核心：turn 循环、事件 emit
│   ├── orchestrator.go   # 组装 system prompt、上下文、工具
│   ├── events.go         # AgentEvent 定义与序列化
│   └── tools.go          # 工具定义与执行分发
├── sandbox/
│   ├── sandbox.go        # Sandbox 接口
│   ├── local.go          # LocalSandbox 实现
│   └── docker.go         # DockerSandbox 占位（未来实现）
└── handler/
    └── agent.go          # WebSocket 接入、room 管理
```

---

## 4. 推荐实现顺序

1. **Sandbox 接口 + LocalSandbox**
   - 限定工作目录 `/tmp/riffpad-sandbox-{workspaceID}`
   - 命令白名单 + 路径校验
   - 单元测试

2. **工具定义**
   - 用 OpenAI function definition 格式定义 `file_read`、`file_write`、`file_list`、`bash_exec`

3. **Agent Loop**
   - 调用 LLM（OpenAI 兼容接口）
   - 解析 tool_calls
   - 串行/并行执行工具
   - 事件 emit

4. **WebSocket handler**
   - `/ws/workspaces/:id`
   - 把 AgentEvent 推送到前端

5. **前端接入**
   - 订阅 WebSocket
   - 展示 thinking / tool_call / file_change / message 事件

6. **权限档位**
   - Confirm 模式批量确认
   - Safe / Auto 模式

---

## 5. 与 TSD 的对照

| 本报告设计 | TSD 章节 |
|-----------|---------|
| Sandbox 接口 | §5.1 沙箱内部结构、§9.1 沙箱隔离 |
| Agent Loop turn 循环 | §6.6 Agent Loop 策略 |
| 事件驱动 / WebSocket | §6.1、§11.3 WebSocket 事件 |
| 工具定义 | §5.2 Agent 工具集 |
| 上下文压缩 | §6.5 上下文管理 |
| 权限档位 | §5.2.8 Agent 权限控制 |
| 工具错误处理 | §6.7 安全与异常处理 |

---

## 6. 结论

pi 的最大启示是：**Agent Loop 应该围绕“消息抽象 + 事件流 + 可插拔执行环境”来设计**，而不是把 LLM 调用、工具执行、UI 更新焊死在一起。

Riffpad 可以直接采纳这个分层：

- `Sandbox` 接口屏蔽本地/Docker 差异。
- `Agent Loop` 负责与 LLM 对话、调度工具、emit 事件。
- `WebSocket handler` 只负责把事件桥接到前端。
- 前端只负责渲染事件流。

这样既能快速做出 Agent-first 的本地演示，又能平滑过渡到 Docker / Firecracker 沙箱，不破坏现有代码。
