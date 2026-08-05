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

- `DESIGN.md` 是 Riffpad 自有的 **Console-Mobile** 设计规范：终端控制台 × 手机遥控器。
- 保留克制的终端气质（等宽字体、控制台表面、`//` 与 `●` 等少量符号），但刻意与 opencode 的 manpage 风格区分：使用品牌琥珀色、方角卡片、bento 网格，chrome 中不使用 `:~$` / `~/` / `[按钮]` 等装饰。
- 深色主题、中英双语、CJK 无衬线回退均为规范的一部分。
- 设计系统决策记录在仓库根目录 `design-system/riffpad/`（由 ui-ux-pro-max skill 生成）。
- 字体使用 Geist Mono 可变字体（OFL License，见 `app/fonts/GeistMono-OFL.txt`）。
