import { defineConfig } from "vitepress";
import { resolve } from "node:path";

const enNav = [
  { text: "Quickstart", link: "/guide/quickstart" },
  { text: "Remote", link: "/guide/remote" },
  { text: "Self-host", link: "/guide/self-host" },
  { text: "CLI", link: "/reference/cli" },
  { text: "Architecture", link: "/reference/architecture" },
  { text: "Security", link: "/reference/security" },
  { text: "FAQ", link: "/faq" },
];

const enSidebar = [
  {
    text: "Guide",
    items: [
      { text: "Quickstart", link: "/guide/quickstart" },
      { text: "Remote control", link: "/guide/remote" },
      { text: "Self-host relay", link: "/guide/self-host" },
    ],
  },
  {
    text: "Reference",
    items: [
      { text: "CLI", link: "/reference/cli" },
      { text: "Architecture", link: "/reference/architecture" },
      { text: "Security model", link: "/reference/security" },
    ],
  },
  { text: "FAQ", link: "/faq" },
];

const zhNav = [
  { text: "快速开始", link: "/zh/guide/quickstart" },
  { text: "手机遥控", link: "/zh/guide/remote" },
  { text: "自部署", link: "/zh/guide/self-host" },
  { text: "CLI", link: "/zh/reference/cli" },
  { text: "架构", link: "/zh/reference/architecture" },
  { text: "安全", link: "/zh/reference/security" },
  { text: "FAQ", link: "/zh/faq" },
];

const zhSidebar = [
  {
    text: "指南",
    items: [
      { text: "快速开始", link: "/zh/guide/quickstart" },
      { text: "手机遥控", link: "/zh/guide/remote" },
      { text: "自部署中继", link: "/zh/guide/self-host" },
    ],
  },
  {
    text: "参考",
    items: [
      { text: "CLI 命令", link: "/zh/reference/cli" },
      { text: "系统架构", link: "/zh/reference/architecture" },
      { text: "安全模型", link: "/zh/reference/security" },
    ],
  },
  { text: "FAQ", link: "/zh/faq" },
];

// Client-side redirect that sends users with a Chinese language preference to
// /docs/zh/... while keeping English at the root. This mirrors the landing
// page behaviour (see apps/landing/components/LanguageProvider.tsx).
const localeRedirectScript = `(function () {
  var STORAGE_KEY = "riffpad-lang";
  var pathname = window.location.pathname;
  // Only redirect on the docs root or an English top-level path; explicit
  // /zh/ URLs are left alone.
  if (pathname.startsWith("/docs/zh")) return;
  if (!pathname.startsWith("/docs")) return;
  var stored = null;
  try { stored = localStorage.getItem(STORAGE_KEY); } catch (e) {}
  var isZh = stored === "zh" || (!stored && (navigator.language || "").toLowerCase().startsWith("zh"));
  if (!isZh) return;
  var target = pathname.replace(/^\/docs/, "/docs/zh") || "/docs/zh/";
  window.location.replace(target + window.location.search + window.location.hash);
})();`;

export default defineConfig({
  title: "Riffpad",
  description: "Watch, approve, and steer AI coding agents from your phone / AI coding agent 的手机遥控器",
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
    ["script", { innerHTML: localeRedirectScript }],
  ],
  // Build straight into the landing app so riffpad.ai/docs is served by the
  // same Vercel deployment.
  outDir: resolve(import.meta.dirname, "../../landing/public/docs"),
  cleanOutDir: true,
  locales: {
    root: {
      label: "English",
      lang: "en",
      themeConfig: {
        nav: enNav,
        sidebar: enSidebar,
        langMenuLabel: "Language",
      },
    },
    zh: {
      label: "中文",
      lang: "zh-CN",
      link: "/zh/",
      themeConfig: {
        nav: zhNav,
        sidebar: zhSidebar,
        langMenuLabel: "语言",
      },
    },
  },
  themeConfig: {
    socialLinks: [{ icon: "github", link: "https://github.com/riffpad/riffpad" }],
  },
});
