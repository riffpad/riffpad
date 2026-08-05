"use client";

import { useTheme } from "./ThemeProvider";

export function ThemeToggle() {
  const { theme, setTheme } = useTheme();
  const next = theme === "light" ? "dark" : "light";

  return (
    <button
      type="button"
      onClick={() => setTheme(next)}
      className="flex h-11 min-w-11 cursor-pointer items-center justify-center rounded-sm border border-hairline bg-surface-muted px-3 text-xs font-bold text-mute transition-colors hover:text-ink"
      aria-label={theme === "light" ? "Switch to dark theme" : "Switch to light theme"}
    >
      [{next}]
    </button>
  );
}
