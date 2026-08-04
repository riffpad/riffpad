# Riffpad Protocol

daemon ↔ relay ↔ mobile 共用的事件协议（草案）。

## 事件集

```jsonc
// daemon → mobile
{
  "type": "approval_request",
  "sessionId": "s_123",
  "requestId": "req_456",
  "action": "file_delete",
  "summary": "删除 src/old.ts",
  "options": ["approve", "reject"],
  "timestamp": 1760000000000
}

// mobile → daemon
{
  "type": "approval_response",
  "requestId": "req_456",
  "decision": "approve"
}
```

完整事件集见 `docs/design.md` §3。正式实现时用 JSON Schema 或 protobuf 生成 Go / TypeScript 类型。
