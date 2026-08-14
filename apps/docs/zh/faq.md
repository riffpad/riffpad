# FAQ

## 需要一直开着电脑吗？

是。agent 跑在你自己的电脑上，Riffpad 只是把手机变成遥控器；电脑休眠后会话无法继续。

## 我的数据会上传到云端吗？

会话内容不会。只有加密信封经过 relay，relay 不解密、不落盘；消息历史存在电脑本地。

## 支持哪些 coding CLI？

Claude Code、Codex、Kimi CLI 已支持；DeepSeek / GLM 在路线图上。

## 一个 daemon 能配对多少设备？

不限。每台设备独立管理、可单独撤销。

## 会话中途断网/重启 daemon 会怎样？

客户端会自动重连并去重回放；daemon 重启后会恢复未结束的会话（Codex 可自动重连 TUI，Kimi/Claude 只读恢复）。

## 如何反馈问题？

去 <https://github.com/riffpad/riffpad/issues> 提 issue，或联系 hi@riffpad.ai。
