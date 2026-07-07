"use client";

import { useTheme } from "./ThemeProvider";
import { Sun, Moon } from "./Icons";

export function ThemeToggle() {
  const { theme, setTheme } = useTheme();
  const next = theme === "light" ? "dark" : "light";
  const label = theme === "light" ? "Light" : "Dark";

  return (
    <button
      onClick={() => setTheme(next)}
      className="flex items-center gap-2 rounded-md border border-hairline bg-surface px-3 py-2 text-sm font-semibold text-body transition hover:text-foreground"
      aria-label={`Current theme: ${label}. Click to switch to ${next} mode.`}
      title={label}
    >
      {theme === "dark" ? (
        <Moon className="h-4 w-4" />
      ) : (
        <Sun className="h-4 w-4" />
      )}
      <span className="hidden sm:inline">{label}</span>
    </button>
  );
}
