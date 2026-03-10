"use client";

import { createContext, useContext, useEffect, useSyncExternalStore, type ReactNode } from "react";
import { getPreferredTheme, isTheme, THEME_STORAGE_KEY, toggleTheme } from "@/lib/theme.mjs";

type Theme = "light" | "dark";
const THEME_EVENT = "pai-theme-change";

type ThemeContextValue = {
  theme: Theme;
  toggle: () => void;
};

const ThemeContext = createContext<ThemeContextValue | null>(null);

function getSystemTheme(): Theme {
  if (typeof window === "undefined") {
    return "light";
  }

  return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
}

function applyTheme(theme: Theme) {
  const root = document.documentElement;
  root.classList.toggle("dark", theme === "dark");
  root.style.colorScheme = theme;
}

function getThemeSnapshot(): Theme {
  if (typeof window === "undefined") {
    return "light";
  }

  const storedTheme = window.localStorage.getItem(THEME_STORAGE_KEY);
  return getPreferredTheme(isTheme(storedTheme) ? storedTheme : "system", getSystemTheme());
}

function subscribe(onStoreChange: () => void) {
  const mediaQuery = window.matchMedia("(prefers-color-scheme: dark)");
  const handleMediaChange = () => {
    const nextStoredTheme = window.localStorage.getItem(THEME_STORAGE_KEY);
    if (!nextStoredTheme || nextStoredTheme === "system") {
      onStoreChange();
    }
  };
  const handleThemeChange = () => onStoreChange();
  const handleStorage = (event: StorageEvent) => {
    if (event.key === THEME_STORAGE_KEY) {
      onStoreChange();
    }
  };

  mediaQuery.addEventListener("change", handleMediaChange);
  window.addEventListener(THEME_EVENT, handleThemeChange);
  window.addEventListener("storage", handleStorage);

  return () => {
    mediaQuery.removeEventListener("change", handleMediaChange);
    window.removeEventListener(THEME_EVENT, handleThemeChange);
    window.removeEventListener("storage", handleStorage);
  };
}

export function ThemeProvider({ children }: { children: ReactNode }) {
  const theme = useSyncExternalStore(subscribe, getThemeSnapshot, () => "light");

  useEffect(() => {
    applyTheme(theme);
  }, [theme]);

  const handleToggle = () => {
    const nextTheme = toggleTheme(theme);
    window.localStorage.setItem(THEME_STORAGE_KEY, nextTheme);
    applyTheme(nextTheme);
    window.dispatchEvent(new Event(THEME_EVENT));
  };

  return <ThemeContext.Provider value={{ theme, toggle: handleToggle }}>{children}</ThemeContext.Provider>;
}

export function useTheme() {
  const context = useContext(ThemeContext);
  if (!context) {
    throw new Error("useTheme must be used within ThemeProvider");
  }

  return context;
}
