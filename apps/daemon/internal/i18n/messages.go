package i18n

var messages = map[string]map[string]string{
	"en": {
		"usage": `riffpad — AI agent remote control

Usage:
  riffpad daemon start          start the background daemon (same binary)
  riffpad daemon stop           stop the daemon
  riffpad status                show daemon status
  riffpad pair [--local]        print a pairing code and QR (requires login;
                                --local allows a local-only code without login)
  riffpad sessions              list sessions
  riffpad auth                  show the logged-in relay account
  riffpad run [codex|claude|kimi] [--name N] [--prompt P] [--cwd D]
  riffpad login [--url wss://…]  log in to Riffpad cloud (GitHub OAuth by default;
                                --username … for password login)
  riffpad logout                clear the saved login token
  riffpad setup                 install daemon auto-start (Linux systemd user service)
  riffpad kill                  stop all sessions and revoke all devices
  riffpad update                check for updates and replace this binary
  riffpad logs                  tail daemon logs
  riffpad version
`,
		"daemon_already_running":      "daemon already running at %s",
		"daemon_started":              "daemon started at %s",
		"daemon_not_reachable":        "daemon not reachable at %s",
		"daemon_start_hint":           "daemon not reachable at %s (run: riffpad daemon start)",
		"daemon_not_running":          "daemon is not running",
		"daemon_stopped":              "daemon stopped",
		"daemon_did_not_stop":         "daemon did not stop",
		"daemon_restarted":            "Daemon restarted.",
		"daemon_restart_failed":       "Failed to restart daemon",
		"daemon_restart_wait_failed":  "daemon did not become reachable after restart",
		"daemon_stop_no_pid":          "daemon did not stop and no usable daemon.pid file was found; kill the process manually",
		"daemon_stop_pid_mismatch":    "pid %d from daemon.pid is not a riffpad daemon; refusing to kill it",
		"daemon_stop_kill_failed":     "force-killing the daemon failed: %v",
		"daemon_start_wait_failed":    "daemon did not become reachable; check %s",
		"session_url":                 "session: %s\nOpen %s to view",
		"run_failed_status":           "Failed to start session (HTTP %d)",
		"pair_code":                   "Pairing code: %s\nEnter it in the phone/browser (or scan the QR)",
		"pair_local":                  "Pairing code: %s\nLocal mode (not logged in): open this URL in a browser on THIS machine:\n%s",
		"pair_requires_login":         "Not logged in. Run `riffpad login` first, or use `riffpad pair --local` for a local-only code.",
		"pair_failed_status":          "Failed to get pairing code (HTTP %d)",
		"pair_login_expired":          "Relay login has expired. Please log in again: riffpad login",
		"pair_waiting_host":           "Waiting for the daemon to connect to the cloud relay…",
		"config_file_healed":          "Warning: %s was corrupted; rebuilt with defaults. The original was backed up to %s",
		"keys_regenerated_warn":       "WARNING: identity keys were regenerated. All previously paired devices are now invalid — pair them again (riffpad pair).",
		"attach_injected":             "Injected Claude Code hooks (backup at settings.json.riffpad.bak)",
		"attach_keep_existing":        "Existing hooks detected; they are kept and riffpad only adds/updates its own entries.",
		"attach_next":                 "Now open your claude normally (tmux recommended); the daemon captures the session and approvals.",
		"attach_verify":               "When done, run: riffpad detach",
		"detach_restored":             "Removed riffpad hooks from Claude Code settings; your other hooks and settings are untouched.",
		"login_password":              "Password: ",
		"login_failed":                "login failed",
		"login_failed_status":         "login failed (status %d)",
		"login_success":               "Logged in as %s; token saved to config.",
		"login_oauth_open":            "Open this URL and authorize with GitHub:\n%s\n\nAuthorization code: %s",
		"login_oauth_timeout":         "authorization timed out; please try again",
		"login_oauth_failed":          "starting device login failed",
		"login_oauth_failed_status":   "device login failed (status %d)",
		"login_oauth_poll_failed":     "checking authorization failed",
		"login_host_check_failed":     "could not verify host ownership",
		"login_host_check_warn":       "could not verify host ownership with relay (%v); keeping current host credentials",
		"login_restarted":             "Daemon restarted; new login is now active.",
		"login_restart_failed":        "Login saved, but restarting the daemon failed: %v",
		"auth_not_logged_in":          "Not logged in. Run: riffpad login",
		"auth_logged_in":              "Logged in as %s (relay: %s)",
		"auth_logged_in_cached":       "Logged in as %s (relay: %s, relay unreachable — showing cached login)",
		"auth_token_invalid":          "Saved login for %s is no longer valid (relay: %s). Run: riffpad login",
		"auth_relay_error":            "could not verify login with relay: %s",
		"logout_done":                 "Logged out.",
		"kill_done":                   "Kill switch engaged: %d sessions stopped, all devices revoked.",
		"setup_removed":               "Removed riffpad systemd user service.",
		"setup_installed":             "Installed and enabled %s",
		"setup_done":                  "The daemon will start at login and restart after crashes; you can now run riffpad run/attach/pair directly.",
		"update_current":              "Current version: %s",
		"update_latest":               "Latest version: %s",
		"update_up_to_date":           "Already up to date.",
		"update_downloading":          "Downloading %s …",
		"update_checksum_failed":      "Checksum verification failed; update aborted (original file unchanged).",
		"update_backup_failed":        "Backup failed, update aborted: %v",
		"update_replace_failed":       "Replace failed: %v",
		"update_done":                 "Updated to %s (old binary backed up at %s)",
		"update_restart_hint":         "If the daemon is running, run `riffpad daemon stop && riffpad daemon start` to apply the new version.",
		"update_platform_unsupported": "update does not support %s yet",
		"update_arch_unsupported":     "update does not support architecture %s yet",
		"checksum_missing":            "checksum file has no entry for %s",
		"download_failed":             "download %s: %s",
		"latest_release_failed":       "failed to fetch latest release: %s",
		"release_no_tag":              "release is missing tag_name",
		"resolve_data_dir":            "resolve data dir: %v",
		"unsupported_cli":             "unsupported cli %q",
		"usage_daemon":                "usage: riffpad daemon start|stop|restart",
		"usage_login":                 "usage: riffpad login [--url wss://…] | riffpad logout",
		"usage_run":                   "usage: riffpad run [codex|claude|kimi] [--name N] [--prompt P] [--cwd D]",
		"attach_deprecated":           "Attach mode is deprecated and disabled. Use `riffpad run claude` instead.",
	},
	"zh": {
		"usage": `riffpad — AI agent 远程遥控

用法：
  riffpad daemon start          启动后台 daemon（同一二进制）
  riffpad daemon stop           停止 daemon
  riffpad status                查看 daemon 状态
  riffpad pair [--local]        打印配对码和二维码（需先登录；
                                --local 允许未登录时生成仅本机可用的码）
  riffpad sessions              列出会话
  riffpad auth                  查看当前登录的 relay 账号
  riffpad run [codex|claude|kimi] [--name N] [--prompt P] [--cwd D]
                                创建并启动会话
  riffpad login [--url wss://…]  登录 Riffpad 云服务（默认 GitHub 授权；
                                --username … 使用密码登录）
  riffpad logout                清除登录 token
  riffpad setup                 安装 daemon 自启（Linux systemd user service）
  riffpad kill                  熔断：停止所有会话并撤销所有设备
  riffpad update                检查更新并替换当前二进制
  riffpad logs                  查看 daemon 日志
  riffpad version
`,
		"daemon_already_running":      "daemon 已在 %s 运行",
		"daemon_started":              "daemon 已在 %s 启动",
		"daemon_not_reachable":        "无法连接 daemon：%s",
		"daemon_start_hint":           "无法连接 daemon：%s（运行：riffpad daemon start）",
		"daemon_not_running":          "daemon 未在运行",
		"daemon_stopped":              "daemon 已停止",
		"daemon_did_not_stop":         "daemon 未能停止",
		"daemon_restarted":            "daemon 已重启。",
		"daemon_restart_failed":       "重启 daemon 失败",
		"daemon_restart_wait_failed":  "重启后 daemon 未能就绪",
		"daemon_stop_no_pid":          "daemon 未能停止，且未找到可用的 daemon.pid 文件，请手动结束进程",
		"daemon_stop_pid_mismatch":    "daemon.pid 中的 pid %d 不是 riffpad daemon，已拒绝强杀",
		"daemon_stop_kill_failed":     "强制结束 daemon 失败：%v",
		"daemon_start_wait_failed":    "daemon 未能就绪；请检查 %s",
		"session_url":                 "会话：%s\n打开 %s 查看",
		"run_failed_status":           "启动会话失败（HTTP %d）",
		"pair_code":                   "配对码：%s\n在手机/浏览器输入此配对码（或扫描二维码）",
		"pair_requires_login":         "未登录。请先运行 `riffpad login`，或使用 `riffpad pair --local` 生成仅本机可用的配对码。",
		"pair_local":                  "配对码：%s\n本地模式（未登录）：请在本机浏览器打开以下链接：\n%s",
		"pair_failed_status":          "获取配对码失败（HTTP %d）",
		"pair_login_expired":          "登录已过期，请重新运行 riffpad login",
		"pair_waiting_host":           "正在等待 daemon 连接云端中继…",
		"config_file_healed":          "警告：%s 已损坏，已用默认值重建；原文件已备份到 %s",
		"keys_regenerated_warn":       "警告：身份密钥已重新生成，所有已配对设备将失效，请重新配对（riffpad pair）。",
		"attach_injected":             "已注入 Claude Code hooks（备份在 settings.json.riffpad.bak）",
		"attach_keep_existing":        "检测到已有 hooks：将全部保留，riffpad 只新增/更新自己的条目。",
		"attach_next":                 "现在正常打开你的 claude（建议放在 tmux 里），daemon 会自动捕捉会话与审批。",
		"attach_verify":               "验证完运行: riffpad detach",
		"detach_restored":             "已从 Claude Code settings 移除 riffpad hooks，你的其他 hooks 和配置保持不变。",
		"login_password":              "密码: ",
		"login_failed":                "登录失败",
		"login_failed_status":         "登录失败（状态 %d）",
		"login_success":               "已登录 %s，token 已保存到配置。",
		"login_oauth_open":            "请在浏览器中打开下面的链接并用 GitHub 授权：\n%s\n\n授权码：%s",
		"login_oauth_timeout":         "授权超时，请重新发起登录。",
		"login_oauth_failed":          "发起设备登录失败",
		"login_oauth_failed_status":   "设备登录失败（状态 %d）",
		"login_oauth_poll_failed":     "查询授权状态失败",
		"login_host_check_failed":     "无法校验主机归属",
		"login_host_check_warn":       "无法向 relay 校验主机归属（%v），保留当前主机凭据",
		"login_restarted":             "daemon 已重启，新登录已生效。",
		"login_restart_failed":        "登录已保存，但重启 daemon 失败：%v",
		"auth_not_logged_in":          "未登录。运行：riffpad login",
		"auth_logged_in":              "当前登录：%s（relay：%s）",
		"auth_logged_in_cached":       "当前登录：%s（relay：%s，relay 不可达，显示缓存）",
		"auth_token_invalid":          "保存的登录 %s 已失效（relay：%s）。请重新运行 riffpad login",
		"auth_relay_error":            "无法向 relay 校验登录状态：%s",
		"logout_done":                 "已退出登录。",
		"kill_done":                   "已熔断：停止 %d 个会话，撤销所有设备。",
		"setup_removed":               "已移除 riffpad systemd user 服务。",
		"setup_installed":             "已安装并启用 %s",
		"setup_done":                  "daemon 将随登录自启，崩溃后自动重启；以后可直接运行 riffpad run/attach/pair。",
		"update_current":              "当前版本: %s",
		"update_latest":               "最新版本: %s",
		"update_up_to_date":           "已是最新版本。",
		"update_downloading":          "下载 %s …",
		"update_checksum_failed":      "校验失败，已中止更新（原文件未改动）。",
		"update_backup_failed":        "备份失败，已中止更新: %v",
		"update_replace_failed":       "替换失败: %v",
		"update_done":                 "已更新到 %s（旧版本备份: %s）",
		"update_restart_hint":         "如果 daemon 正在运行，请执行 `riffpad daemon stop && riffpad daemon start` 让新版本生效。",
		"update_platform_unsupported": "update 暂不支持 %s",
		"update_arch_unsupported":     "update 暂不支持架构 %s",
		"checksum_missing":            "校验和文件里没有 %s",
		"download_failed":             "下载 %s: %s",
		"latest_release_failed":       "获取最新版本失败: %s",
		"release_no_tag":              "release 缺少 tag_name",
		"resolve_data_dir":            "解析数据目录失败: %v",
		"unsupported_cli":             "不支持的 CLI %q",
		"usage_daemon":                "用法：riffpad daemon start|stop|restart",
		"usage_login":                 "用法：riffpad login [--url wss://…] | riffpad logout",
		"usage_run":                   "用法：riffpad run [codex|claude|kimi] [--name N] [--prompt P] [--cwd D]",
		"attach_deprecated":           "附着模式已弃用并关闭。请改用 `riffpad run claude`。",
	},
}
