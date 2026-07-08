"use client";

import { useTheme } from "./ThemeProvider";
import { Sun, Moon } from "./Icons";

interface ThemeToggleProps {
  className?: string;
  showLabel?: boolean;
}

export function ThemeToggle({ className, showLabel }: ThemeToggleProps = {}) {
  const { theme, setTheme } = useTheme();
  const next = theme === "light" ? "dark" : "light";
  const label = theme === "light" ? "Light" : "Dark";

  return (
    <button
      onClick={() => setTheme(next)}
      className={`flex h-11 items-center gap-2 rounded-md px-3 py-2 text-sm font-semibold text-body transition hover:text-foreground${
        className ? ` ${className}` : ""
      }`}
      aria-label={`Current theme: ${label}. Click to switch to ${next} mode.`}
      title={label}
    >
      {theme === "dark" ? (
        <Moon className="h-4 w-4" />
      ) : (
        <Sun className="h-4 w-4" />
      )}
      <span className={showLabel ? "" : "hidden sm:inline"}>{label}</span>
    </button>
  );
}
