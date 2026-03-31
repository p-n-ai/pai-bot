"use client";

import { Moon, SunMedium } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useTheme } from "@/components/theme-provider";

export function ThemeToggle() {
  const { theme, mounted, toggle } = useTheme();
  const isDark = theme === "dark";
  const label = mounted ? (isDark ? "Switch to light theme" : "Switch to dark theme") : "Toggle theme";

  return (
    <Button
      type="button"
      variant="outline"
      size="icon-sm"
      onClick={toggle}
      aria-label={label}
      title={label}
      className="rounded-full border-slate-200/70 bg-slate-50/72 text-slate-700 shadow-[0_12px_30px_rgba(15,23,42,0.1)] backdrop-blur hover:bg-slate-50/88 dark:border-white/12 dark:bg-slate-900/58 dark:text-slate-100 dark:hover:bg-slate-900/72"
    >
      {mounted && isDark ? <SunMedium className="size-4" /> : <Moon className="size-4" />}
    </Button>
  );
}
