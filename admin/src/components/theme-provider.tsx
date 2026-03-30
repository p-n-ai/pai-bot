"use client";

import type { ReactNode } from "react";
import { ThemeProvider as NextThemesProvider, useTheme as useNextTheme } from "next-themes";
import { useHydrated } from "@/hooks/use-hydrated";
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
      disableTransitionOnChange
    >
      {children}
    </NextThemesProvider>
  );
}

export function useTheme() {
  const isHydrated = useHydrated();
  const { resolvedTheme, setTheme } = useNextTheme();
  const theme: Theme = resolvedTheme === "dark" ? "dark" : "light";

  return {
    isHydrated,
    theme,
    toggle: () => setTheme(theme === "dark" ? "light" : "dark"),
  };
}
