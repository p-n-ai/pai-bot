"use client";

import { IconMoon, IconSun } from "@tabler/icons-react";
import { Button } from "@/components/ui/button";
import { useTheme } from "@/components/theme-provider";
import { cn } from "@/lib/utils";

export function ThemeToggle({ className }: { className?: string } = {}) {
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
      className={cn("rounded-full", className)}
    >
      {mounted && isDark ? <IconSun /> : <IconMoon />}
    </Button>
  );
}
