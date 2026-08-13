import { defineConfig } from "vitepress";
import { resolve } from "node:path";

const zhNav = [
  { text: "快速开始", link: "/guide/quickstart" },
  { text: "手机遥控", link: "/guide/remote" },
  { text: "自部署", link: "/guide/self-host" },
  { text: "CLI", link: "/reference/cli" },
  { text: "架构", link: "/reference/architecture" },
  { text: "安全", link: "/reference/security" },
  { text: "FAQ", link: "/faq" },
];

const zhSidebar = [
  {
    text: "指南",
    items: [
      { text: "快速开始", link: "/guide/quickstart" },
      { text: "手机遥控", link: "/guide/remote" },
      { text: "自部署中继", link: "/guide/self-host" },
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
];

const enNav = [
  { text: "Quickstart", link: "/en/guide/quickstart" },
  { text: "Remote", link: "/en/guide/remote" },
  { text: "Self-host", link: "/en/guide/self-host" },
  { text: "CLI", link: "/en/reference/cli" },
  { text: "Architecture", link: "/en/reference/architecture" },
  { text: "Security", link: "/en/reference/security" },
  { text: "FAQ", link: "/en/faq" },
];

const enSidebar = [
  {
    text: "Guide",
    items: [
      { text: "Quickstart", link: "/en/guide/quickstart" },
      { text: "Remote control", link: "/en/guide/remote" },
      { text: "Self-host relay", link: "/en/guide/self-host" },
    ],
  },
  {
    text: "Reference",
    items: [
      { text: "CLI", link: "/en/reference/cli" },
      { text: "Architecture", link: "/en/reference/architecture" },
      { text: "Security model", link: "/en/reference/security" },
    ],
  },
  { text: "FAQ", link: "/en/faq" },
];

export default defineConfig({
  title: "Riffpad",
  description: "AI coding agent 的手机遥控器 / Watch, approve, and steer AI coding agents from your phone",
  base: "/docs/",
  cleanUrls: true,
  ignoreDeadLinks: true,
  head: [
    ["link", { rel: "icon", type: "image/png", href: "/docs/favicon-light.png", id: "riffpad-favicon" }],
    [
      "script",
      {
        innerHTML: `(function () {
          var link = document.getElementById("riffpad-favicon");
          if (!link) return;
          var html = document.documentElement;
          function currentTheme() {
            var stored = null;
            try { stored = localStorage.getItem("vitepress-theme-appearance"); } catch (e) {}
            if (html.classList.contains("dark") || stored === "dark") return "dark";
            if (stored === "light") return "light";
            return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
          }
          function apply() {
            link.setAttribute(
              "href",
              currentTheme() === "dark" ? "/docs/favicon-dark.png" : "/docs/favicon-light.png"
            );
          }
          apply();
          if ("MutationObserver" in window) {
            new MutationObserver(apply).observe(html, {
              attributes: true,
              attributeFilter: ["class"],
            });
          }
        })();`,
      },
    ],
  ],
  // Build straight into the landing app so riffpad.ai/docs is served by the
  // same Vercel deployment.
  outDir: resolve(import.meta.dirname, "../../landing/public/docs"),
  cleanOutDir: true,
  locales: {
    root: {
      label: "中文",
      lang: "zh-CN",
      themeConfig: {
        nav: zhNav,
        sidebar: zhSidebar,
        langMenuLabel: "语言",
      },
    },
    en: {
      label: "English",
      lang: "en",
      link: "/en/",
      themeConfig: {
        nav: enNav,
        sidebar: enSidebar,
        langMenuLabel: "Language",
      },
    },
  },
  themeConfig: {
    socialLinks: [{ icon: "github", link: "https://github.com/riffpad/riffpad" }],
  },
});
