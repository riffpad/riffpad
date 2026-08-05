# Riffpad Relay 部署

relay 是云端 WebSocket 中继：用户电脑上的 daemon（host）和手机（viewer）都主动连接它，
不需要端口转发。用户/主机/设备/会话元数据默认持久化到 SQLite
（`RELAY_DATA_DIR/relay.db`，WAL 模式），设置 `DATABASE_URL`（Postgres DSN）后自动切换
Postgres——代码已支持双驱动，relay 重启不丢。

## 方案对比

| 方案 | 成本 | TLS | 适合 | 备注 |
|---|---|---|---|---|
| Fly.io | 免费额度 / 约 $3-5/月 | 自动 | 海外测试、MVP | `fly deploy` 即可 |
| Railway / Render | 免费额度有限 | 自动 | 快速试跑 | 免费服务会休眠，需常驻配置 |
| VPS + Caddy | 约 $5/月 | Caddy 自动 | 长期/国内可用 | 最灵活，推荐生产 |
| Cloudflare Tunnel | 免费 | 自动 | 不想开公网端口 | WebSocket 需开启 |
| 国内云（腾讯云/阿里云）+ 备案域名 | 约 ¥50-100/月 | Caddy/Nginx | 国内正式上线 | 需 ICP 备案，周期 1-3 周 |

## Fly.io 部署

```bash
# 1. 安装 flyctl 并登录（需要 Fly 账号）
curl -L https://fly.io/install.sh | sh
fly auth login

# 2. 改 infra/relay/fly.toml：app 名（数据库在 /data 卷上）
# 3. 在仓库根目录部署
fly launch --no-deploy --name riffpad-relay --dockerfile infra/relay/Dockerfile
fly deploy

# 4. 拿到公网地址（自动 HTTPS）
fly open
```

部署后，在 relay 网页（https://…）注册账号并登录；daemon 配置：

```bash
export RIFFPAD_RELAY_URL=wss://riffpad-relay.fly.dev
export RIFFPAD_RELAY_USER=<你的用户名>
export RIFFPAD_RELAY_PASSWORD=<你的密码>
```

或执行 `riffpad relay login --url wss://riffpad-relay.fly.dev --username <你的用户名>`。
首次启动 daemon 会自动登录并注册 host（hostId + hostSecret 保存到
`~/.config/riffpad/config.json`）。`riffpad pair` 返回 relay 页面地址
（https://…/?pair=CODE），手机在已登录状态下输入即可配对。

## VPS + Caddy 部署

```bash
# 服务器上
useradd -r -m riffpad
cp riffpad-relay.service /etc/systemd/system/
cat > /etc/riffpad-relay.env <<EOF
RELAY_DATA_DIR=/var/lib/riffpad-relay
EOF
mkdir -p /var/lib/riffpad-relay && chown riffpad:riffpad /var/lib/riffpad-relay
systemctl daemon-reload && systemctl enable --now riffpad-relay

# Caddy（域名解析到服务器后自动签发证书）
apt install caddy   # 或 docker 跑 caddy
cp Caddyfile /etc/caddy/Caddyfile   # 把 relay.example.com 换成你的域名
systemctl reload caddy
```

## 同 WiFi 真机测试（零部署）

relay 默认监听所有网卡（`:9090`）。电脑和手机连同一 WiFi 后：

1. 电脑跑 `riffpad pair`，记下 6 位配对码
2. 手机浏览器打开 `http://<电脑局域网IP>:9090/`，注册/登录账号
3. 输入配对码完成配对，即可看到电脑上的 claude 会话并审批

> 注意：这没有 TLS，仅限可信局域网测试。

## 安全提醒

- 密码用 bcrypt 哈希存储；登录 token 30 天过期，登出即失效
- daemon 首次注册后使用专属 hostSecret 连接，不再共享密钥
- relay 数据目录（SQLite）必须持久化；生产建议挂载独立卷。单实例早期 SQLite 够用；
  多实例扩容或需要托管备份时切 Postgres（`DATABASE_URL`）
- relay 零知识：只转发加密信封，不落内容；但元数据（设备/会话）可见，公网部署建议尽早接 Postgres 与审计
- 生产多实例需要共享会话路由（Redis pub/sub 或粘性连接），单实例阶段不需要
