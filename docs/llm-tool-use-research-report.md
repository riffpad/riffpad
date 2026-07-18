# LLM Tool-Use 调研报告：DeepSeek V4 Flash 问题与稳定替代方案

> 调研目标：确认 DeepSeek V4-Flash 在 Agent/Tool-Use 场景下是否存在已知缺陷，并筛选出适合 Riffpad 的替代模型。

---

## 1. 结论摘要

**DeepSeek V4-Flash / V4-Pro 在 multi-turn tool calling 场景下存在多个已确认的 bug/limitation**，不是 Riffpad 代码独有的问题。主要表现包括：

1. thinking mode 下 `reasoning_content` 必须在轮次之间回传，否则 400
2. tool result 回传后模型可能返回空响应（`completion_tokens=0`）
3. 流式 + function calling 组合下响应不稳定/为空
4. 参数类型错误（boolean 被序列化成字符串）
5. 某些框架下必须先使用其他模型调用一次工具，DeepSeek 才能正常工作

**建议：Riffpad 默认不应以 DeepSeek V4-Flash 作为生产 Agent 模型**。优先选择以下经过大规模验证的模型：

| 模型/平台 | 工具调用稳定性 | 中文能力 | 推荐场景 |
|---|---|---|---|
| OpenAI GPT-4o / GPT-4.1 | 极高 | 强 | 有 key 时的首选 |
| Anthropic Claude 3.5/4 Sonnet | 极高 | 强 | 编码/复杂 Agent |
| 火山方舟 Doubao-Pro / Doubao-Seed-Code | 高 | 极强 | 国内首选 |
| 阿里云百炼 Qwen-Max / Qwen-Plus | 高 | 极强 | 国内生态完整 |
| 智谱 GLM-4 / GLM-4.7 | 中高 | 强 | 企业 function calling |

---

## 2. DeepSeek V4-Flash 已知问题（有公开 issue）

### 2.1 `reasoning_content` 必须在多轮 tool calling 中回传

**来源**：[n8n-io/n8n#29661](https://github.com/n8n-io/n8n/issues/29661)、[稀土掘金](https://juejin.cn/post/7643478523496267816)

当 `deepseek-v4-flash` 在 thinking mode 下执行 tool call 时，assistant message 会同时包含：
- `tool_calls`
- `reasoning_content`

如果下一轮请求没有把 `reasoning_content` 原样带回去，DeepSeek API 直接返回 400：

```text
The reasoning_content in the thinking mode must be passed back to the API
```

**对 Riffpad 的影响**：
- 我们目前通过 `LLM_ENABLE_THINKING=false` 禁用了 thinking，因此不会触发这个 400
- 但如果用户开启 thinking 且 agent loop 没有保留 `reasoning_content`，就会中断

### 2.2 Tool result 回传后返回空响应

**来源**：[deepseek-ai/DeepSeek-V3#1453](https://github.com/deepseek-ai/DeepSeek-V3/issues/1453)

`deepseek-v4-pro` 在流式 + function calling 模式下，**间歇性返回完全空响应**：
- `content=""`
- `reasoning_content=""`
- `completion_tokens=0`
- HTTP 200（没有错误码）

关键特征：
- 只发生在 tool message 被喂回之后的轮次
- 正常响应 latency 15-70s，空响应 1-2s 就回来
- 一旦触发，后续同会话轮次持续为空
- 非确定性，同一 prompt 有时正常有时空

**这与我们在 Riffpad 中观察到的现象高度吻合**：第二轮 LLM 调用在 tool result 回传后挂死或超时。

### 2.3 非流式 tool calling 中 boolean 参数被错误序列化

**来源**：[vllm-project/vllm#41122](https://github.com/vllm-project/vllm/issues/41122)

`DeepSeek V4 Flash` 通过 vLLM 部署时，boolean 参数会被返回成 JSON 字符串 `"true"` 而不是 JSON boolean `true`，导致下游工具 schema 验证失败。

### 2.4 工具调用"预热"问题

**来源**：[open-webui/open-webui#25153](https://github.com/open-webui/open-webui/issues/25153)

在 Open WebUI 中，如果第一个使用的模型就是 DeepSeek V4-Pro，tool calling 会失败； workaround 是先用其他模型（如 GPT）调用一次工具，再切回 DeepSeek。

### 2.5 模型偶尔不走 `tool_calls[]` 而直接吐 XML

**来源**：[memohai/Memoh#437](https://github.com/memohai/Memoh/issues/437)

DeepSeek V4 Pro 偶尔会直接在 `content` 里输出 XML 格式的工具调用：

```xml
<tool_call id="call_00_xxx" name="some_tool">
{"param1": "value1"}
</tool_call>
```

而不是标准的 `tool_calls[]` 结构化字段，导致框架无法识别并执行工具。

### 2.6 thinking/action 不一致导致无限 tool-calling 循环

**来源**：[NousResearch/hermes-agent#37255](https://github.com/NousResearch/hermes-agent/issues/37255)

`deepseek-v4-pro` 在完成任务后，thinking block 认为应该停止并输出最终答案，但 action generation 继续调用无意义的工具，导致无限循环。

---

## 3. 我们与公开 issue 的对应关系

| Riffpad 观察到的现象 | 对应公开 issue / 根因 |
|---|---|
| 中文问题 + 中文 system/tools，第二轮挂死 60s+ | 2.2 空响应/无推理问题 + 中文场景更容易触发 |
| 英文问题 + 英文 system/tools，调研任务可完成 | 2.2 非确定性，英文场景触发概率较低 |
| 首 token 等待时间极不稳定（1s ~ 120s） | 2.2 空响应前的正常响应 latency 15-70s，异常时 1-2s |
| `enable_thinking=false` 仍偶尔很慢 | DeepSeek V4 thinking 是 hybrid mode，即使关闭也可能残留内部 reasoning |

---

## 4. 稳定替代方案对比

### 4.1 国际模型（效果最佳）

| 模型 | 工具调用 | 中文 | 速度 | 备注 |
|---|---|---|---|---|
| **GPT-4o** | 行业标杆 | 优秀 | 快 | BFCL 长期领先，最稳定 |
| **GPT-4.1 / GPT-5-mini** | 优秀 | 优秀 | 快 | OpenAI 最新一代 |
| **Claude 3.5/4 Sonnet** | 优秀 | 优秀 | 中等 | 编码/Agent 场景极强 |
| **Gemini 1.5 Pro / 2.5** | 优秀 | 良好 | 快 | 长上下文 + 工具 |

### 4.2 国内模型（合规/ latency 优势）

| 平台/模型 | 工具调用 | 中文 | 速度 | 备注 |
|---|---|---|---|---|
| **火山方舟 Doubao-Pro** | 高 | 极强 | 快 | 字节跳动内部大规模验证，function calling 文档完善 |
| **阿里云百炼 Qwen-Max** | 高 | 极强 | 快 | 通义千问生态完整，Qwen-Agent 大量实践 |
| **阿里云百炼 Qwen-Plus** | 高 | 强 | 快 | 性价比更高的选择 |
| **智谱 GLM-4 / GLM-4.7** | 中高 | 强 | 中等 | 被评价为"stable enterprise function calling" |
| **Kimi K2.5** | 中高 | 强 | 中等 | 长文本 + 工具 |

### 4.3 不推荐用于生产 Agent 的模型

| 模型 | 原因 |
|---|---|
| DeepSeek V4-Flash / V4-Pro | 多轮 tool calling 存在多个已知 bug |
| DeepSeek-R1 / reasoning 系列 | 不支持 tool_choice，reasoning 必须回传 |
| 小规模开源模型（<14B） | tool calling 准确率不足 |

---

## 5. 对 Riffpad 的建议

### 5.1 短期

1. **默认模型改为非 DeepSeek**：在 `.env.example` 中把默认示例从 DeepSeek 改为火山方舟 Doubao-Pro 或阿里云 Qwen-Plus
2. **保留 DeepSeek 支持但标注风险**：允许用户继续使用，但文档中说明 multi-turn tool calling 已知不稳定
3. **保持 15s/60s 超时策略**：已实现的超时机制对任何模型都是合理的安全网
4. **保留 reasoning_content 流式**：当用户使用 thinking 模型时，确保 `reasoning_delta` 事件被保留并回传

### 5.2 中期

1. **模型路由/降级**：当某模型连续失败时，自动切换到备选模型
2. **provider health check**：启动时快速测试当前 provider 的 tool calling 是否可用
3. **把 `reasoning_content` 保留到 message history**：为支持 DeepSeek thinking mode 做准备

### 5.3 长期

1. **支持多 provider 配置**：主模型 + fallback 模型
2. **按任务复杂度选模型**：简单问答用轻量模型，复杂 Agent 用强模型
3. **接入模型性能监控**：记录每轮 tool calling 的 latency、成功率

---

## 6. 参考链接

- [n8n DeepSeek V4-Flash tool calling 400 问题](https://github.com/n8n-io/n8n/issues/29661)
- [DeepSeek V4 多轮 Tool-Calling 报 reasoning_content 400 排查](https://juejin.cn/post/7643478523496267816)
- [DeepSeek V4-Pro tool result 后空响应 bug](https://github.com/deepseek-ai/DeepSeek-V3/issues/1453)
- [DeepSeek V4 boolean 参数被序列化为字符串](https://github.com/vllm-project/vllm/issues/41122)
- [Open WebUI DeepSeek 工具调用需预热问题](https://github.com/open-webui/open-webui/issues/25153)
- [Memoh DeepSeek V4 吐 XML 而非 tool_calls](https://github.com/memohai/Memoh/issues/437)
- [Hermes Agent DeepSeek V4 无限 tool-calling 循环](https://github.com/NousResearch/hermes-agent/issues/37255)
- [火山方舟 Function Calling 文档](https://www.volcengine.com/docs/82379/1262342)
- [阿里云百炼 Function Calling 文档](https://www.alibabacloud.com/help/en/model-studio/qwen-function-calling)
- [中国 LLM Agent & Tool Calling 指南 2026](https://www.swifthorseai.com/en/articles/china-llm-agent-tool-calling-2026)
