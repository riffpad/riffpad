# 快速开始

## 1. 安装 CLI

在电脑终端执行：

::: code-group

```bash [macOS / Linux]
curl -fsSL https://riffpad.ai/install.sh | sh
```

```powershell [Windows]
irm https://riffpad.ai/install.ps1 | iex
```

:::

Windows 安装脚本会下载最新二进制、加入 PATH，并注册登录自启任务（daemon）；`riffpad setup --remove` 可移除。

安装后确认：

```bash
riffpad version
```

## 2. 登录

```bash
riffpad login
```

会自动打开浏览器用 GitHub 授权（也支持 `--username` 密码登录）。登录成功后 daemon 自动重启。

查看当前账号：

```bash
riffpad auth
```

## 3. 配对手机

```bash
riffpad pair
```

终端会打印 6 位配对码和二维码。手机浏览器打开 <https://app.riffpad.ai>，用同一个 GitHub 账号登录，然后扫码或输入配对码。

## 4. 创建并查看会话

```bash
riffpad run --cli codex
```

也可以指定 agent 与初始指令：

```bash
riffpad run --cli claude --prompt "重构 auth 中间件"
```

会话出现在手机 App 的 Sessions 页，点进去即可查看实时进度、审批和发指令。

## 常见命令

| 命令 | 作用 |
|---|---|
| `riffpad login` / `logout` | GitHub 设备授权登录 / 退出 |
| `riffpad auth` | 查看当前登录账号 |
| `riffpad pair` | 打印配对码 + 二维码 |
| `riffpad run --cli <claude\|codex\|kimi>` | 创建并运行会话 |
| `riffpad sessions` | 列出会话 |
| `riffpad update` | 自更新二进制 |
| `riffpad kill` | 熔断：停止所有会话并撤销设备 |
