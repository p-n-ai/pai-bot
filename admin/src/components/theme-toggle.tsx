"use client";

import { useEffect, useState } from "react";
import { Moon, SunMedium } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useTheme } from "@/components/theme-provider";

export function ThemeToggle() {
  const { theme, toggle } = useTheme();
  const [mounted, setMounted] = useState(false);

  useEffect(() => {
    setMounted(true);
  }, []);

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
      className="rounded-full border-white/50 bg-white/75 text-slate-700 shadow-[0_12px_30px_rgba(15,23,42,0.12)] backdrop-blur hover:bg-white dark:border-white/10 dark:bg-slate-950/75 dark:text-slate-100 dark:hover:bg-slate-900"
    >
      {mounted ? (isDark ? <SunMedium className="size-4" /> : <Moon className="size-4" />) : <Moon className="size-4" />}
    </Button>
  );
}
