"use client";

import { AnimatePresence, motion, useReducedMotion } from "framer-motion";
import { IconMoon, IconSun } from "@tabler/icons-react";
import { Button } from "@/components/ui/button";
import { useTheme } from "@/components/theme-provider";
import { cn } from "@/lib/utils";

const themeToggleEase = [0.22, 1, 0.36, 1] as const;

export function ThemeToggle({ className }: { className?: string } = {}) {
  const { theme, mounted, toggle } = useTheme();
  const prefersReducedMotion = useReducedMotion();
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
      <AnimatePresence mode="wait" initial={false}>
        <motion.span
          key={mounted && isDark ? "sun" : "moon"}
          initial={prefersReducedMotion ? { opacity: 1 } : { opacity: 0, scale: 0.82, rotate: -18, filter: "blur(8px)" }}
          animate={prefersReducedMotion ? { opacity: 1 } : { opacity: 1, scale: 1, rotate: 0, filter: "blur(0px)" }}
          exit={prefersReducedMotion ? { opacity: 1 } : { opacity: 0, scale: 0.88, rotate: 18, filter: "blur(8px)" }}
          transition={{ duration: 0.22, ease: themeToggleEase }}
          className="inline-flex"
        >
          {mounted && isDark ? <IconSun /> : <IconMoon />}
        </motion.span>
      </AnimatePresence>
    </Button>
  );
}
