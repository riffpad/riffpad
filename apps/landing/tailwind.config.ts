import type { Config } from "tailwindcss";

const config: Config = {
  content: [
    "./pages/**/*.{js,ts,jsx,tsx,mdx}",
    "./components/**/*.{js,ts,jsx,tsx,mdx}",
    "./app/**/*.{js,ts,jsx,tsx,mdx}",
    "./lib/**/*.{js,ts,jsx,tsx,mdx}",
  ],
  theme: {
    extend: {
      colors: {
        background: "var(--background)",
        foreground: "var(--foreground)",
        surface: "var(--surface)",
        "surface-elevated": "var(--surface-elevated)",
        muted: "var(--muted)",
        body: "var(--body)",
        accent: "var(--accent)",
        "accent-hover": "var(--accent-hover)",
        border: "var(--border)",
        hairline: "var(--hairline)",
      },
      fontFamily: {
        display: ["var(--font-geist-sans)", "PingFang SC", "Microsoft YaHei", "sans-serif"],
        sans: ["var(--font-geist-sans)", "PingFang SC", "Microsoft YaHei", "sans-serif"],
        mono: ["var(--font-geist-mono)", "ui-monospace", "SFMono-Regular", "monospace"],
      },
      letterSpacing: {
        tighter: "-0.05em",
        display: "-0.05em",
      },
    },
  },
  plugins: [],
};
export default config;
