export const ACCESS_TOKEN_KEY = "pai_token";
export const REFRESH_TOKEN_KEY = "pai_refresh_token";
export const USER_KEY = "pai_user";
export const SESSION_CHANGED_EVENT = "pai-admin-session-change";

export const ACCESS_TOKEN_COOKIE = "pai_admin_access";
export const REFRESH_TOKEN_COOKIE = "pai_admin_refresh";
export const USER_COOKIE = "pai_admin_user";

export function buildCookieValue(name: string, value: string, maxAgeSeconds: number): string {
  return `${name}=${encodeURIComponent(value)}; Path=/; Max-Age=${maxAgeSeconds}; SameSite=Lax`;
}

export function buildCookieRemoval(name: string): string {
  return `${name}=; Path=/; Max-Age=0; SameSite=Lax`;
}

export function parseCookieJSON<T>(value: string | undefined): T | null {
  if (!value) return null;

  try {
    return JSON.parse(decodeURIComponent(value)) as T;
  } catch {
    return null;
  }
}
