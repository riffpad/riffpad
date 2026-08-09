# CLI 命令

| 命令 | 说明 |
|---|---|
| `riffpad login [--url wss://…]` | GitHub 设备授权登录（默认）；`--username` 走密码登录 |
| `riffpad logout` | 退出并撤销 relay token |
| `riffpad auth` | 查看当前登录账号（实时校验） |
| `riffpad pair` | 打印 6 位配对码与二维码 |
| `riffpad run [claude\|codex\|kimi] [--name N] [--prompt P] [--cwd D]` | 创建并托管会话，终端保持可见可操作 |
| `riffpad attach` / `detach` | 注入/移除 Claude hooks，捕获你自己启动的会话 |
| `riffpad sessions` | 列出会话 |
| `riffpad status` | daemon 状态 |
| `riffpad setup [--remove]` | 安装/移除 Linux systemd 自启 |
| `riffpad kill` | 熔断：停止所有会话 + 撤销所有设备 |
| `riffpad update` | 检查并替换为最新版本（SHA256 校验 + 原子替换） |
| `riffpad logs` | 查看 daemon 日志 |
| `riffpad version` | 版本号 |

## 多语种

CLI 输出支持中/英文，自动跟随系统语言，可用 `--lang zh|en` 覆盖。

## 环境变量

| 变量 | 说明 |
|---|---|
| `RIFFPAD_RELAY_URL` | relay 地址（默认 `wss://api.riffpad.ai`） |
| `RIFFPAD_RELAY_USER` / `RIFFPAD_RELAY_PASSWORD` | 密码登录凭据 |
| `RIFFPAD_URL` | 本地 daemon 地址（默认 `http://127.0.0.1:8787`） |
| `RIFFPAD_DIR` | 数据目录（默认 `~/.config/riffpad`） |
