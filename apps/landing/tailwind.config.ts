import type { Config } from "tailwindcss";

const config: Config = {
  darkMode: ["selector", "[data-theme='dark']"],
  content: [
    "./app/**/*.{js,ts,jsx,tsx,mdx}",
    "./components/**/*.{js,ts,jsx,tsx,mdx}",
    "./lib/**/*.{js,ts,jsx,tsx,mdx}",
  ],
  theme: {
    extend: {
      colors: {
        canvas: "var(--canvas)",
        surface: "var(--surface)",
        "surface-muted": "var(--surface-muted)",
        ink: "var(--ink)",
        body: "var(--body)",
        mute: "var(--mute)",
        hairline: "var(--hairline)",
        "hairline-strong": "var(--hairline-strong)",
        accent: "var(--accent)",
        "accent-hover": "var(--accent-hover)",
        "accent-ink": "var(--accent-ink)",
        success: "var(--success)",
        warning: "var(--warning)",
        danger: "var(--danger)",
        info: "var(--info)",
        console: "var(--console)",
        "console-elevated": "var(--console-elevated)",
        "on-console": "var(--on-console)",
        "on-console-mute": "var(--on-console-mute)",
      },
      fontFamily: {
        mono: [
          "var(--font-geist-mono)",
          "IBM Plex Mono",
          "ui-monospace",
          "SFMono-Regular",
          "Menlo",
          "Monaco",
          "Consolas",
          "PingFang SC",
          "Hiragino Sans GB",
          "Microsoft YaHei",
          "Noto Sans CJK SC",
          "WenQuanYi Micro Hei",
          "sans-serif",
        ],
      },
      borderRadius: {
        sm: "8px",
        md: "12px",
        lg: "20px",
        full: "9999px",
      },
      boxShadow: {
        card: "0 1px 2px rgba(0,0,0,0.04), 0 8px 24px rgba(0,0,0,0.06)",
      },
      maxWidth: {
        content: "960px",
        frame: "1100px",
      },
    },
  },
  plugins: [],
};

export default config;
