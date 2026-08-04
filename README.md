# Riffpad

口袋里的 AI 遥控器：把电脑上正在跑的 AI coding agent（Claude Code、Codex、DeepSeek CLI……）接到手机上——随时查看进度、审批操作、远程转向，人不必守在电脑前。

> 状态：新起点。旧产品（云端代码草稿本）保留在 `../riffpad-legacy` 供参考，确认新方向后可以删除。

## 产品闭环

1. 本地 daemon 监听用户电脑上的 AI CLI 会话
2. agent 等待审批 / 需要转向时，事件经加密中继推到手机
3. 手机上查看结构化进度，一键同意 / 拒绝 / 修改条件 / 下达新指令

## 仓库结构

```
riffpad/
├── apps/
│   ├── daemon/        # Go：跑在用户电脑上的本地桥接器
│   ├── relay/         # Go：WebSocket 加密中继（不落盘、不记录内容）
│   └── mobile/        # 移动端（PWA 起步，后续套原生壳）
├── packages/
│   └── protocol/      # 事件协议定义（daemon ↔ relay ↔ mobile 共用）
├── docs/
│   └── design.md      # 设计说明（动手前先读）
├── Makefile
├── AGENTS.md
└── .env.example
```

## 本地开发

```bash
make dev-daemon   # 本地 daemon
make dev-relay    # 中继服务
make dev-mobile   # 移动端 PWA
```

## 设计文档

详见 [docs/prd.md](docs/prd.md)（产品需求）、[docs/tsd.md](docs/tsd.md)（技术规格）、[docs/design.md](docs/design.md)（快速总览）。
