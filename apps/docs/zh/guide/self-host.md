# 自部署中继（Self-host Relay）

默认情况下，daemon 和手机都连接官方中继 `app.riffpad.ai`。如果你希望自己掌控中继——部署在自己的 VPS、内网或家里——只需一条命令拉起一个独立的 relay 容器，**数据完全留在你自己的服务器上**。

relay 零知识：只转发端到端加密的信封，看不到会话内容；它只保存账号 / 设备 / 会话的**元数据**。

## 一行命令

::: tip 前置
需要一台装了 Docker 的机器（VPS、家里的服务器、甚至 NAS 都行）。
:::

```bash
curl -fsSL https://riffpad.ai/selfhost.sh | sh
```

脚本会：

1. 拉取公共镜像 `ghcr.io/riffpad/relay`；
2. 在 `~/.riffpad-relay/` 生成 `docker-compose.yml`、`.env` 和数据卷 `data/`；
3. 启动 relay（默认内嵌 SQLite，无需额外数据库）；
4. 打印访问地址，以及如何把你的电脑（daemon）指向这个中继。

> 镜像首次发布后需在 [GitHub Packages](https://github.com/riffpad/riffpad/pkgs/container/relay) 把 `riffpad/relay` 设为 **public**，否则 `docker pull` 会 401。

## 公网 + 自动 HTTPS

如果你有域名、要把中继暴露到公网，加 `--domain`，脚本会额外起一个 Caddy 容器自动签发证书：

```bash
curl -fsSL https://riffpad.ai/selfhost.sh | sh -s -- --domain relay.example.com
```

先把域名的 DNS A 记录指向这台机器。Caddy 会在首次请求时自动申请 Let's Encrypt 证书（可加 `--email you@x.com` 指定 ACME 账号）。

没有域名？默认的 HTTP 模式适合**可信局域网 / 测试**。也可以自己在前面套 nginx/Caddy，见 [relay 部署 README](https://github.com/riffpad/riffpad/tree/main/infra/relay#readme)。

## 把你的电脑指向自部署中继

relay 起来后，在每台跑 daemon 的电脑上注册并登录这个中继：

```bash
# HTTP 模式（局域网 IP:端口）或 HTTPS 模式（你的域名）
export RIFFPAD_RELAY_URL=wss://relay.example.com   # HTTP 模式用 ws://<IP>:9090

riffpad relay login --url "$RIFFPAD_RELAY_URL" --username <你的用户名>
```

密码交互输入（也可用 `RIFFPAD_RELAY_PASSWORD` 环境变量）。登录后 daemon 自动重连，之后照常 `riffpad pair` 配对手机即可。

::: tip
用户名 / 密码是**你在这个自部署 relay 上**首次注册的账号——在浏览器打开 relay 地址点注册即可创建。
:::

## 升级到 Postgres（可选）

默认 SQLite 单文件够大多数自部署场景用。若要更高的并发或托管备份，切到 Postgres：编辑 `~/.riffpad-relay/docker-compose.yml`，把 relay service 换成下面这版并加一个 postgres service：

```yaml
services:
  relay:
    image: ghcr.io/riffpad/relay:latest
    restart: unless-stopped
    expose: ["9090"]
    env_file: [.env]
    volumes:
      - ./data:/data
    environment:
      RELAY_LISTEN: "0.0.0.0"
      RELAY_PORT: "9090"
      DATABASE_URL: postgres://riffpad:${POSTGRES_PASSWORD}@postgres:5432/riffpad?sslmode=disable
    depends_on:
      postgres:
        condition: service_healthy

  postgres:
    image: postgres:17-alpine
    restart: unless-stopped
    environment:
      POSTGRES_USER: riffpad
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
      POSTGRES_DB: riffpad
    volumes:
      - pg-data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U riffpad"]
      interval: 3s
      timeout: 3s
      retries: 10

volumes:
  pg-data:
```

在 `.env` 里写 `POSTGRES_PASSWORD=...`，然后 `docker compose up -d`。relay 检测到 `DATABASE_URL` 会自动建表并使用 Postgres。

## 管理、升级、备份

所有操作在安装目录 `~/.riffpad-relay/` 下进行：

```bash
cd ~/.riffpad-relay

docker compose logs -f relay     # 实时日志
docker compose restart relay     # 重启
docker compose pull && docker compose up -d   # 升级到最新镜像
docker compose down               # 停止
```

**备份**：停服后拷贝 `data/` 目录（SQLite 模式）或 dump Postgres 即可。元数据加密存储；会话内容从不落 relay 盘。

**改端口 / 域名 / 镜像 tag**：重跑安装脚本，传新参数即可——它会重新生成 compose，数据卷不动：

```bash
curl -fsSL https://riffpad.ai/selfhost.sh | sh -s -- --port 8080 --tag v0.2.5
```

## 安全说明

- 默认 HTTP 模式**无加密**，仅限可信局域网。公网务必用 `--domain`（Caddy 自动 TLS）或自己的反代。
- relay 只保存元数据（账号 / 设备 / 会话列表），会话内容是端到端加密的信封，relay 无法解密。
- `~/.riffpad-relay/.env` 含凭据，设为 `chmod 600`，不要进 git。
- 如需 GitHub 登录，在 `.env` 填 `GITHUB_CLIENT_ID` / `GITHUB_CLIENT_SECRET`，并把回调地址配成你的 relay 域名。
