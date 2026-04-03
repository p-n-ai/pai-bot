export const ACCESS_TOKEN_COOKIE = "pai_admin_access";
export const REFRESH_TOKEN_COOKIE = "pai_admin_refresh";
export const USER_COOKIE = "pai_admin_user";

export function parseCookieJSON<T>(value: string | undefined): T | null {
  if (!value) return null;

  try {
    return JSON.parse(decodeURIComponent(value)) as T;
  } catch {
    return null;
  }
}
