"use client";

import { useTheme } from "./ThemeProvider";
import { Sun, Moon } from "./Icons";

export function ThemeToggle() {
  const { theme, resolvedTheme, setTheme } = useTheme();

  const next: Record<"light" | "dark" | "system", "light" | "dark" | "system"> = {
    light: "dark",
    dark: "system",
    system: "light",
  };

  return (
    <button
      onClick={() => setTheme(next[theme])}
      className="flex items-center gap-2 rounded-md border border-hairline bg-surface px-3 py-2 text-sm font-semibold text-body transition hover:text-foreground"
      aria-label={`Current theme: ${resolvedTheme}. Click to switch to ${next[theme]}.`}
      title={`Theme: ${theme}`}
    >
      {resolvedTheme === "dark" ? (
        <Moon className="h-4 w-4" />
      ) : (
        <Sun className="h-4 w-4" />
      )}
      <span className="hidden sm:inline">{theme}</span>
    </button>
  );
}
