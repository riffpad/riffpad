# 手机遥控

配对完成后，打开 <https://app.riffpad.ai>（或本地 daemon 的 8787 端口页面）即可：

- **Sessions**：查看所有正在运行的会话、状态灯（WAITING FOR INPUT / RUNNING / DONE）、最近活跃时间；点卡片进入会话详情。
- **会话详情**：实时聊天流 + 工具调用日志；Agent 运行时底部出现 `Interrupt` 打断按钮；向上翻历史会自动加载更早消息（分页）。
- **审批**：Agent 等待确认时出现审批卡片，一键同意/拒绝。
- **指令**：输入框输入文字并发送（`→`），指令会加密传输到电脑上的 daemon。
- **Devices**：查看已配对设备、识别当前设备、撤销授权。

## 需要 tmux 吗？

不需要。`riffpad run` 是托管模式，daemon 直接捕获 CLI 会话；`riffpad attach` 可以把你自己启动的 Claude 会话也接进来。

如果你想要“关掉终端后会话继续跑”的持久会话，建议搭配 tmux：

```bash
tmux new -s work
riffpad run codex
# Ctrl-b d 脱离；回来后 tmux attach -t work
```
