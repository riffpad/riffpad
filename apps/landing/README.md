# Riffpad Landing

Riffpad 官网 landing page：Next.js 14（App Router）+ TypeScript + Tailwind CSS，静态导出到 `out/`，中英双语、深浅主题。

## 开发

```bash
pnpm install
pnpm --filter landing dev
pnpm --filter landing build
pnpm --filter landing lint
```

构建产物在 `apps/landing/out/`，可直接交给静态托管（Vercel / Cloudflare Pages 等）。

## 目录

```
app/          # Next.js App Router 页面与全局样式
components/   # Header / Hero / Terminal mockup / 各 section
lib/          # EN/ZH 文案字典
public/       # robots.txt / sitemap.xml
```

## 设计参考

- `DESIGN.md` 来自 [VoltAgent/awesome-design-md](https://github.com/VoltAgent/awesome-design-md/blob/main/design-md/opencode.ai/DESIGN.md)（MIT License，Copyright (c) 2026 VoltAgent），作为视觉规范基底：全站等宽字体、米白画布、1px hairline、ASCII 括号标记、单块深色终端 mockup。
- 在此基础上为 Riffpad 增加了深色主题、中英双语和品牌色（仅用于终端 mockup 内部语义状态）。
- 设计系统决策记录在仓库根目录 `design-system/riffpad/`（由 ui-ux-pro-max skill 生成）。
- 字体使用 Geist Mono 可变字体（OFL License，见 `app/fonts/GeistMono-OFL.txt`）。
