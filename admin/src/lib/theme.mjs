export const THEME_STORAGE_KEY = "pai-admin-theme";

export function isTheme(value) {
  return value === "light" || value === "dark" || value === "system";
}

export function getPreferredTheme(savedTheme, systemTheme) {
  if (savedTheme === "light" || savedTheme === "dark") {
    return savedTheme;
  }

  return systemTheme === "dark" ? "dark" : "light";
}

export function toggleTheme(currentTheme) {
  return currentTheme === "dark" ? "light" : "dark";
}
