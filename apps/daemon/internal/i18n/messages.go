package i18n

var messages = map[string]map[string]string{
	"en": {
		"usage": `riffpad — AI agent remote control

Usage:
  riffpad daemon start          start the background daemon (same binary)
  riffpad daemon stop           stop the daemon
  riffpad status                show daemon status
  riffpad pair                  print a pairing code and QR
  riffpad sessions              list sessions
  riffpad run [--name N] [--prompt P] [--cwd D] [--cli claude|kimi|codex]
  riffpad attach                inject Claude Code hooks so the daemon captures your own CLI session
  riffpad detach                remove injected hooks
  riffpad login [--url wss://… --username …]
                                log in to Riffpad cloud (relay)
  riffpad logout                clear the saved login token
  riffpad setup                 install daemon auto-start (Linux systemd user service)
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
		"daemon_start_wait_failed":    "daemon did not become reachable; check %s",
		"session_url":                 "session: %s\nOpen %s to view",
		"pair_code":                   "Pairing code: %s\nEnter it in the phone/browser (or scan the QR)",
		"attach_injected":             "Injected Claude Code hooks (backup at settings.json.riffpad.bak)",
		"attach_next":                 "Now open your claude normally (tmux recommended); the daemon captures the session and approvals.",
		"attach_verify":               "When done, run: riffpad detach",
		"detach_restored":             "Restored Claude Code settings; hooks removed.",
		"login_password":              "Password: ",
		"login_failed":                "login failed",
		"login_failed_status":         "login failed (status %d)",
		"login_success":               "Logged in as %s; token saved to config.",
		"logout_done":                 "Logged out.",
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
		"usage_daemon":                "usage: riffpad daemon start|stop",
		"usage_login":                 "usage: riffpad login|logout",
		"usage_run":                   "usage: riffpad run [--name N] [--prompt P] [--cwd D] [--cli claude|kimi|codex]",
	},
	"zh": {
		"usage": `riffpad — AI agent 远程遥控

用法：
  riffpad daemon start          启动后台 daemon（同一二进制）
  riffpad daemon stop           停止 daemon
  riffpad status                查看 daemon 状态
  riffpad pair                  打印配对码和二维码
  riffpad sessions              列出会话
  riffpad run [--name N] [--prompt P] [--cwd D] [--cli claude|kimi|codex]
                                创建并启动会话
  riffpad attach                注入 Claude Code hooks，让 daemon 捕获你自己启动的会话
  riffpad detach                移除注入的 hooks
  riffpad login [--url wss://… --username …]
                                登录 Riffpad 云服务（relay）
  riffpad logout                清除登录 token
  riffpad setup                 安装 daemon 自启（Linux systemd user service）
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
		"daemon_start_wait_failed":    "daemon 未能就绪；请检查 %s",
		"session_url":                 "会话：%s\n打开 %s 查看",
		"pair_code":                   "配对码：%s\n在手机/浏览器输入此配对码（或扫描二维码）",
		"attach_injected":             "已注入 Claude Code hooks（备份在 settings.json.riffpad.bak）",
		"attach_next":                 "现在正常打开你的 claude（建议放在 tmux 里），daemon 会自动捕捉会话与审批。",
		"attach_verify":               "验证完运行: riffpad detach",
		"detach_restored":             "已还原 Claude Code settings，hooks 已移除。",
		"login_password":              "密码: ",
		"login_failed":                "登录失败",
		"login_failed_status":         "登录失败（状态 %d）",
		"login_success":               "已登录 %s，token 已保存到配置。",
		"logout_done":                 "已退出登录。",
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
		"usage_daemon":                "用法：riffpad daemon start|stop",
		"usage_login":                 "用法：riffpad login|logout",
		"usage_run":                   "用法：riffpad run [--name N] [--prompt P] [--cwd D] [--cli claude|kimi|codex]",
	},
}
