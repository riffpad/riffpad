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
        "surface-soft": "var(--surface-soft)",
        "surface-card": "var(--surface-card)",
        "surface-dark": "var(--surface-dark)",
        "surface-dark-elevated": "var(--surface-dark-elevated)",
        ink: "var(--ink)",
        "ink-deep": "var(--ink-deep)",
        body: "var(--body)",
        mute: "var(--mute)",
        stone: "var(--stone)",
        ash: "var(--ash)",
        "on-dark": "var(--on-dark)",
        "on-dark-mute": "var(--on-dark-mute)",
        hairline: "var(--hairline)",
        "hairline-strong": "var(--hairline-strong)",
        accent: "var(--accent)",
        "accent-hover": "var(--accent-hover)",
        "accent-active": "var(--accent-active)",
        warning: "var(--warning)",
        "warning-hover": "var(--warning-hover)",
        danger: "var(--danger)",
        "danger-hover": "var(--danger-hover)",
        success: "var(--success)",
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
          "monospace",
        ],
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
