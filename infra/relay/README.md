# Riffpad Relay 部署

relay 是云端 WebSocket 中继：用户电脑上的 daemon（host）和手机（viewer）都主动连接它，
不需要端口转发。host 与已配对设备会持久化到数据目录，relay 重启不丢；会话路由为内存态。
Postgres 持久化在 M1.8。

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

# 2. 改 infra/relay/fly.toml：app 名 + REGISTRATION_KEY（强随机串）
# 3. 在仓库根目录部署
fly launch --no-deploy --name riffpad-relay --dockerfile infra/relay/Dockerfile
fly secrets set REGISTRATION_KEY="$(openssl rand -hex 24)"
fly deploy

# 4. 拿到公网地址（自动 HTTPS）
fly open
```

部署后 daemon 配置：

```bash
export RIFFPAD_RELAY_URL=wss://riffpad-relay.fly.dev
export RIFFPAD_REGISTRATION_KEY=<与 REGISTRATION_KEY 一致>
```

首次启动 daemon 会自动向 relay 注册，获得专属 hostId + hostSecret 并保存到
`~/.config/riffpad/config.json`；之后无需再带注册密钥。`riffpad pair` 会返回 relay
页面地址（https://…/?pair=CODE），手机扫码/输码即可。

## VPS + Caddy 部署

```bash
# 服务器上
useradd -r -m riffpad
cp riffpad-relay.service /etc/systemd/system/
cat > /etc/riffpad-relay.env <<EOF
REGISTRATION_KEY=$(openssl rand -hex 24)
EOF
systemctl daemon-reload && systemctl enable --now riffpad-relay

# Caddy（域名解析到服务器后自动签发证书）
apt install caddy   # 或 docker 跑 caddy
cp Caddyfile /etc/caddy/Caddyfile   # 把 relay.example.com 换成你的域名
systemctl reload caddy
```

## 同 WiFi 真机测试（零部署）

relay 默认监听所有网卡（`:9090`）。电脑和手机连同一 WiFi 后：

1. 电脑跑 `riffpad pair`，记下 6 位配对码
2. 手机浏览器打开 `http://<电脑局域网IP>:9090/?pair=<CODE>`
3. 配对后即可看到电脑上的 claude 会话并审批

> 注意：这没有 TLS，仅限可信局域网测试。

## 安全提醒

- `REGISTRATION_KEY` 必须改强随机值；daemon 首次注册后使用专属 hostSecret，不再共享密钥
- relay 数据目录（hosts/devices）要持久化；生产建议挂载独立卷
- relay 零知识：只转发加密信封，不落内容；但元数据（设备/会话）可见，公网部署建议尽早接 Postgres 与审计
- 生产多实例需要共享会话路由（Redis pub/sub 或粘性连接），单实例阶段不需要
