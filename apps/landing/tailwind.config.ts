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
        success: "rgb(var(--success) / <alpha-value>)",
        warning: "rgb(var(--warning) / <alpha-value>)",
        danger: "rgb(var(--danger) / <alpha-value>)",
        info: "rgb(var(--info) / <alpha-value>)",
        console: "var(--console)",
        "console-elevated": "var(--console-elevated)",
        "on-console": "var(--on-console)",
        "on-console-mute": "var(--on-console-mute)",
        // Hero TUI pane — a real nvim session, always Catppuccin Mocha in
        // both page themes, so these are fixed values, not theme vars.
        "tui-bg": "#1e1e2e",
        "tui-surface": "#313244",
        "tui-fg": "#cdd6f4",
        "tui-mute": "#6c7086",
        "tui-mauve": "#cba6f7",
        "tui-green": "#a6e3a1",
        "tui-blue": "#89b4fa",
        "tui-red": "#f38ba8",
        "device-frame": "var(--device-frame)",
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
        sm: "0px",
        md: "0px",
        lg: "0px",
        full: "9999px",
      },
      boxShadow: {
        card: "var(--shadow-card)",
        device: "0 24px 60px -20px rgba(0,0,0,0.45)",
      },
      maxWidth: {
        content: "960px",
        frame: "1200px",
      },
    },
  },
  plugins: [],
};

export default config;
