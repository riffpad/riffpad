import { defineConfig } from "vitepress";
import { resolve } from "node:path";

export default defineConfig({
  lang: "zh-CN",
  title: "Riffpad",
  description: "AI coding agent 的口袋遥控器",
  base: "/docs/",
  // Build straight into the landing app so riffpad.ai/docs is served by the
  // same Vercel deployment.
  outDir: resolve(import.meta.dirname, "../../landing/public/docs"),
  cleanOutDir: true,
  themeConfig: {
    nav: [
      { text: "首页", link: "/" },
      { text: "快速开始", link: "/guide/quickstart" },
      { text: "CLI", link: "/reference/cli" },
      { text: "架构", link: "/reference/architecture" },
      { text: "安全", link: "/reference/security" },
      { text: "FAQ", link: "/faq" },
    ],
    sidebar: [
      {
        text: "指南",
        items: [
          { text: "快速开始", link: "/guide/quickstart" },
          { text: "手机遥控", link: "/guide/remote" },
        ],
      },
      {
        text: "参考",
        items: [
          { text: "CLI 命令", link: "/reference/cli" },
          { text: "系统架构", link: "/reference/architecture" },
          { text: "安全模型", link: "/reference/security" },
        ],
      },
      { text: "FAQ", link: "/faq" },
    ],
    socialLinks: [{ icon: "github", link: "https://github.com/riffpad/riffpad" }],
    search: { provider: "local" },
  },
});
