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
        body: "var(--body)",
        muted: "var(--muted)",
        ash: "var(--ash)",
        hairline: "var(--hairline)",
        "hairline-soft": "var(--hairline-soft)",
        "hairline-strong": "var(--hairline-strong)",
        surface: "var(--surface)",
        "surface-soft": "var(--surface-soft)",
        "surface-doc": "var(--surface-doc)",
        "surface-dark": "var(--surface-dark)",
        "on-dark": "var(--on-dark)",
        accent: "var(--accent)",
        "accent-pressed": "var(--accent-pressed)",
        "accent-active": "var(--accent-active)",
        "on-accent": "var(--on-accent)",
        "link-blue": "var(--link-blue)",
        "link-teal": "var(--link-teal)",
        "accent-blue": "var(--accent-blue)",
        "accent-blue-soft": "var(--accent-blue-soft)",
        "accent-green": "var(--accent-green)",
        "accent-green-soft": "var(--accent-green-soft)",
        "accent-red": "var(--accent-red)",
        "accent-red-soft": "var(--accent-red-soft)",
        "accent-purple": "var(--accent-purple)",
        "accent-purple-soft": "var(--accent-purple-soft)",
      },
      fontFamily: {
        sans: ["var(--font-ibm-plex-sans)", "PingFang SC", "Microsoft YaHei", "sans-serif"],
        mono: ["var(--font-geist-mono)", "ui-monospace", "SFMono-Regular", "monospace"],
      },
      borderRadius: {
        md: "6px",
      },
    },
  },
  plugins: [],
};
export default config;
