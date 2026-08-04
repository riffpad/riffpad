# Riffpad Agent Guide

## 项目

Riffpad 是 AI coding agent 的移动遥控器：本地 daemon 桥接电脑上的 CLI 会话，加密中继推送事件，手机端负责监督、审批与转向。

## 规则

1. **先读 `docs/design.md`**——协议和架构决策以它为准。
2. **最小改动**——只做被要求的事。
3. **安全优先**——中继零信任、端到端加密、移动端默认只读；任何涉及凭据/密钥的改动都要特别谨慎。
4. **测试与验证**——改动后运行相关测试并构建。
5. **协议同步**——事件协议变更必须同步更新 `packages/protocol` 与 `docs/design.md`。
6. **使用 `gh` CLI**——issue、分支、PR 用 GitHub 工作流管理。

## 代码规范

- 分支：`feature/<n>-<slug>` / `bugfix/<n>-<slug>`
- Commit：`feat(daemon): ... (#n)` / `fix(relay): ... (#n)`
- PR 标题 + 验证步骤，body 写 `Closes #n`

## Done Means

- 代码可构建、可测试
- 协议与文档一致
- PR 已评审合并
