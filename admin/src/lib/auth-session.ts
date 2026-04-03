export const SESSION_COOKIE = "pai_session";

export function parseCookieJSON<T>(value: string | undefined): T | null {
  if (!value) return null;

  try {
    return JSON.parse(decodeURIComponent(value)) as T;
  } catch {
    return null;
  }
}
