"use client";

import { useEffect, useMemo, useState, type ReactNode } from "react";
import { ThemeProvider as NextThemesProvider, useTheme as useNextTheme } from "next-themes";
import { THEME_STORAGE_KEY } from "@/lib/theme.mjs";

type Theme = "light" | "dark";

export function ThemeProvider({ children }: { children: ReactNode }) {
  return (
    <NextThemesProvider
      attribute="class"
      defaultTheme="system"
      enableSystem
      enableColorScheme
      storageKey={THEME_STORAGE_KEY}
    >
      {children}
    </NextThemesProvider>
  );
}

export function useTheme(): { theme: Theme; mounted: boolean; toggle: () => void } {
  const { resolvedTheme, setTheme } = useNextTheme();
  const [mounted, setMounted] = useState(false);

  useEffect(() => {
    setMounted(true);
  }, []);

  const theme = useMemo<Theme>(() => (resolvedTheme === "dark" ? "dark" : "light"), [resolvedTheme]);

  const toggle = () => {
    setTheme(theme === "dark" ? "light" : "dark");
  };

  return { theme, mounted, toggle };
}
