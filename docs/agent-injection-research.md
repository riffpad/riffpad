# 无 tmux 指令注入可行性研究

> 日期：2026-08-06
> 分支：`feature/52-no-tmux-injection`
> 结论先行：**可以摆脱 tmux，但“对已运行的普通 TUI 会话注入”没有通用办法**。可行的三条路都要求改变会话启动方式：Kimi 走官方 ACP（`kimi acp`）、Codex 走官方 app-server、Claude 走 host 模式控制协议。ACP 不是万能答案，Claude Code 官方至今不支持 ACP。

---

## 1. 问题定义

现状：`riffpad attach` 后用户照常在终端（建议 tmux）里跑 Claude Code，daemon 通过 `tmux send-keys` 注入网页端指令。tmux 是当前唯一能同时做到：

1. 读已运行 CLI 的 stdout；
2. 向已运行 CLI 的 stdin 写入；
3. 不改变 CLI 的启动方式。

Linux/macOS 上没有“向任意已运行进程 stdin 注入”的通用内核机制（PTY 只能通过 master 写入，pipe 的读端不可写），所以无 tmux 方案的本质是：**让 CLI 以某种受控方式启动，把 stdin/stdout（或等价通道）交给 daemon**。

## 2. 候选通道

| 通道 | 传输 | 适用 CLI | 状态 |
|---|---|---|---|
| ACP（Agent Client Protocol） | stdio JSON-RPC（NDJSON），也可 HTTP/WS | Kimi Code 原生；Codex/Claude 需第三方 adapter | Kimi 实测可用 |
| Claude Code 控制协议 | `claude --input-format stream-json --output-format stream-json` 的 stdin/stdout 控制帧 | Claude Code | 2.1.220 实测可用 |
| Codex app-server | JSON-RPC over stdio / unix socket / ws | Codex | 0.146.1 实测可用 |
| `claude --remote-control` | 出站轮询 Anthropic API | Claude Code | 依赖 Anthropic 账号，不开放本地端口，不做主方案 |
| tmux / PTY 兜底 | 终端字节流 | 任意 CLI | 维持现状 |

### 2.1 ACP 是什么

[Agent Client Protocol](https://agentclientprotocol.com/) 是 Zed 主导的开放协议（当前稳定版 v1），用于编辑器/客户端与 coding agent 通信：客户端通过 JSON-RPC 发 `initialize` → `session/new` → `session/prompt`，agent 反向推送 `session/update`（消息、工具调用、计划）与 `session/request_permission`（审批）。协议字段尽量复用 MCP 的 JSON 表示。

对 Riffpad 的价值：**一套客户端逻辑覆盖多个 agent**——Kimi 原生支持，Codex 有官方 app-server（第三方 adapter 把 ACP 翻译成 app-server 调用），Claude 需要 PTY bridge 或 Agent SDK 封装。

## 3. 实测结果

### 3.1 Kimi Code：ACP 原生支持（✅ 完整可用）

`kimi acp` 以 stdio JSON-RPC 运行，stderr 输出日志，stdout 保持协议通道干净。实测（本机 Kimi Code）：

```json
{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":1,"clientCapabilities":{"fs":{"readTextFile":false,"writeTextFile":false},"terminal":false},"clientInfo":{"name":"riffpad-probe","version":"0.1.0"}}}
```

流程验证：

1. `initialize` → 返回 `protocolVersion: 1`、能力矩阵、`authMethods`（已登录时无需额外认证）；
2. `session/new`（带 `cwd`）→ 返回 `sessionId`；
3. `session/prompt` → 流式 `session/update`（`agent_thought_chunk` / `agent_message_chunk`），最终 `result.stopReason = end_turn`。

能力矩阵（官方文档确认）：`session/load`、`session/resume`、`session/list`、`session/cancel`、`session/set_config_option` 均有；客户端反向 RPC 支持 `session/update`、`session/request_permission`、`fs/read_text_file`、`fs/write_text_file`。**审批、取消、续接、模型切换全部在协议内**。

结论：Kimi 走 ACP 是正路，daemon 作为 ACP client spawn `kimi acp` 即可，完全不需要 tmux、不需要用户开终端。

### 3.2 Codex：app-server（✅ 完整可用）

`codex app-server` 是官方远程控制入口（0.146.1），支持 stdio / unix socket / ws。实测 stdio JSON-RPC：

```json
{"method":"initialize","id":1,"params":{"clientInfo":{"name":"riffpad-probe","version":"0.1.0"}}}
{"method":"initialized","id":null,"params":{}}
{"method":"thread/start","id":2,"params":{"cwd":"/tmp/riffpad-acp-test"}}
{"method":"turn/start","id":3,"params":{"threadId":"...","input":[{"type":"text","text":"Reply with exactly: OK"}]}}
```

验证结果：

- `thread/start` 创建会话并返回 `thread.id`，随后广播 `thread/started`；
- `turn/start` 的 `input` 是数组（不是字符串），正确格式后正常流式返回 `item/reasoning/textDelta`、`item/agentMessage/delta`、`item/completed`；
- 会话状态、token 用量、rate limit 均通过 notification 推送。

对 Riffpad 最有价值的方法：`turn/steer`（向进行中的 turn 追加输入）、`turn/interrupt`（取消）、`thread/resume`（续接）、`thread/list`（枚举）、`command/exec`（沙箱命令）。权限侧有 `permissionProfile/list` 与 `tool/requestUserInput`（实验性），**与审批按钮的映射需要单独验证**。

使用形态：用户以 `codex --remote unix://<path>`（或 `codex app-server daemon` + `codex --remote`）启动，daemon 连同一个 unix socket 即可注入。终端 TUI 保留在用户侧，双方共享会话。

### 3.3 Claude Code：host 模式控制协议（✅ 注入可用，⚠️ 审批受限）

Claude Code 2.1.220 仍无官方 ACP 命令（`claude --help` 无 acp；Anthropic 只提供 `--remote-control` 云通道）。但 CLI 自带 Agent SDK 底层控制协议，daemon 以 host 身份 spawn claude 即可无 tmux 注入：

```bash
claude --input-format stream-json --output-format stream-json --verbose --permission-mode default
```

实测要点：

1. **控制协议 initialize 格式已变**：旧文档的 `{"matcher":"*","hook_callback_ids":["hook_0"]}` 会被拒绝，2.1.220 要求 camelCase 且 matchers 为数组：

   ```json
   {"type":"control_request","request":{"subtype":"initialize","request_id":"req_1","hooks":{
     "PreToolUse":[{"matchers":["*"],"hookCallbackIds":["hook_0"]}],
     "UserPromptSubmit":[{"matchers":[""],"hookCallbackIds":["hook_1"]}],
     "PermissionRequest":[{"matchers":[""],"hookCallbackIds":["hook_perm"]}]
   },"sdk_mcp_servers":[]}}
   ```

   返回 `control_response success`。

2. **指令注入可行**：stdin 写 `{"type":"user","message":{"role":"user","content":[{"type":"text","text":"..."}]}}`，Claude 正常进入会话并返回 assistant 消息。

3. **UserPromptSubmit 钩子会回调宿主**：收到 `control_request`（`subtype: hook_callback`），宿主回复 `control_response`（`hookSpecificOutput.hookEventName` + `decision.behavior: allow`）后 prompt 放行，实测模型正常回复。

4. **⚠️ 工具权限审批在 host 模式下仍自动拒绝**：注册 `PermissionRequest` hook 后，写文件场景没有等来权限回调——`PreToolUse` 回调即使 allow，工具结果仍是 “Claude requested permissions ... but you haven't granted it yet”。这与 TSD 已有的实测结论一致：`--permission-prompt-tool stdio` 已移除，**审批拦截目前仍依赖附着模式（交互 TUI + settings.json hooks）**。

   缓解选项：
   - host 模式按用户选择预设 `--permission-mode acceptEdits`（编辑类自动放行）或 `--dangerously-skip-permissions`（全放行，需明确授权）；
   - 继续研究 2.1.x 是否还有未文档化的权限控制帧（本次未找到）。

5. host 模式的代价：**stdin/stdout 被 daemon 持有，用户终端看不到 TUI**，手机是主交互端（即 TSD 的“包装模式”）。若既要用户终端 TUI 又要无 tmux，Claude 只能靠 PTY 桥（daemon 自己创建 PTY 并转发，不再依赖 tmux，但仍算“daemon 托管启动”）。

## 4. ACP 方向评估

**方向本身是对的，但要区分 CLI**：

| CLI | ACP 原生？ | 推荐通道 | 实现成本 |
|---|---|---|---|
| Kimi Code | ✅ 原生 | `kimi acp`（daemon 作为 ACP client） | 低，一套 ACP client 即可 |
| Codex | ❌ 无 acp 命令 | `codex app-server` JSON-RPC（或第三方 `codex-acp` adapter） | 中，直连 app-server 更稳 |
| Claude Code | ❌ 官方不支持 | host 控制协议（注入）+ 附着模式 hooks（审批） | 中高，两种模式并存 |

对产品形态的影响：

- ACP/app-server/控制协议都是“**daemon 托管或共享会话**”，不是“attach 到用户已开的 TUI”。Riffpad 的附着模式（用户自己开 claude + tmux）短期内仍是 Claude 的唯一零改动方案；
- 若 Riffpad 实现一个通用 ACP client（Go 写 JSON-RPC 不难），Kimi 直接接入，Codex 可挂官方 app-server（协议不同，adapter 可选），Claude 需单独适配；
- 不推荐把 `claude --remote-control` 作为通道：它是 Anthropic 官方手机遥控，走云端轮询、依赖用户 Claude 账号，与 Riffpad 的“用户电脑 + 自有中继”定位冲突。

## 5. 建议的实施路径

按价值排序：

1. **Kimi 接入（M 系列新条目）**：daemon 内置 ACP client，`riffpad run -- kimi acp` 或自动 spawn，复用现有事件协议（`session/message`、`permission_request` 等字段可直接映射 ACP 的 `session/update`、`session/request_permission`）。
2. **Codex 接入**：daemon 连 `codex app-server` unix socket（启动方式由 `riffpad attach codex` 引导用户 `codex --remote`），实现 thread/turn 生命周期映射。
3. **Claude 包装模式增强**：把 host 控制协议做成“包装模式”的正式实现（注入 + 事件已通），审批按权限预设策略；附着模式（tmux + hooks）保留为审批路径，后续再做 PTY 桥合并体验。
4. **统一抽象**：事件协议里增加 adapter 类型字段（`claude-host` / `codex-appserver` / `kimi-acp`），移动端无感知。

## 6. 待办/风险

- [ ] 验证 Codex app-server 的权限请求如何映射到客户端（`tool/requestUserInput` 实验接口或 thread 事件），决定审批按钮实现；
- [ ] 确认 Claude 2.1.x host 模式是否有未文档化的权限控制帧（可尝试 Agent SDK 源码反推）；
- [ ] ACP 协议版本 v1 稳定，但 client-side reverse-RPC 覆盖不全（Kimi 4/9），`session/request_permission` 已实现，够 MVP；
- [ ] unix socket 在同一沙箱里曾遇到 `Operation not permitted`（环境限制），本机实机需再验证 `codex app-server --listen unix://` 与 `codex --remote` 组合；
- [ ] Kimi/Codex 均使用用户自己的模型额度，PoC 消耗正常，无额外成本。

## 附：PoC 脚本

实验脚本位于 `/tmp/riffpad-perm-probe`（Claude 权限）、`/tmp/acp-probe.py`（Kimi ACP）、`/tmp/codex-probe.py`（Codex app-server）、`/tmp/claude-host-probe.py`（Claude host 控制协议），均为一次性验证代码，未入库。
