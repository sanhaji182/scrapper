"use client";

import { useTheme } from "./theme-provider";

export function ThemeToggle() {
  const { theme, toggleTheme } = useTheme();

  return (
    <button type="button" onClick={toggleTheme} className="ghost-button px-3 py-2 text-xs" aria-label="Toggle theme" suppressHydrationWarning>
      {theme === "dark" ? "Light" : "Dark"}
    </button>
  );
}
