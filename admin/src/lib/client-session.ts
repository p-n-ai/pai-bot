"use client";

import type { AuthSession, AuthUser } from "@/lib/api";
import {
  ACCESS_TOKEN_COOKIE,
  ACCESS_TOKEN_KEY,
  REFRESH_TOKEN_COOKIE,
  REFRESH_TOKEN_KEY,
  USER_COOKIE,
  USER_KEY,
  buildCookieRemoval,
  buildCookieValue,
} from "@/lib/auth-session";
import { hasClientSession } from "@/lib/session-state.mjs";

export function readStoredUser(): AuthUser | null {
  if (typeof window === "undefined") return null;

  const raw = localStorage.getItem(USER_KEY);
  if (!raw) return null;

  try {
    return JSON.parse(raw) as AuthUser;
  } catch {
    return null;
  }
}

export function readStoredAccessToken(): string {
  if (typeof window === "undefined") return "";

  return localStorage.getItem(ACCESS_TOKEN_KEY) || "";
}

export function readStoredRefreshToken(): string {
  if (typeof window === "undefined") return "";

  return localStorage.getItem(REFRESH_TOKEN_KEY) || "";
}

export function hasStoredSession(): boolean {
  if (typeof window === "undefined") return false;

  return hasClientSession({
    accessToken: readStoredAccessToken(),
    user: readStoredUser(),
  });
}

export function writeStoredSession(session: AuthSession): void {
  if (typeof window === "undefined") return;

  localStorage.setItem(ACCESS_TOKEN_KEY, session.access_token);
  localStorage.setItem(REFRESH_TOKEN_KEY, session.refresh_token);
  localStorage.setItem(USER_KEY, JSON.stringify(session.user));
  document.cookie = buildCookieValue(ACCESS_TOKEN_COOKIE, session.access_token, 60 * 60 * 24 * 7);
  document.cookie = buildCookieValue(REFRESH_TOKEN_COOKIE, session.refresh_token, 60 * 60 * 24 * 7);
  document.cookie = buildCookieValue(USER_COOKIE, JSON.stringify(session.user), 60 * 60 * 24 * 7);
}

export function removeStoredSession(): void {
  if (typeof window === "undefined") return;

  localStorage.removeItem(ACCESS_TOKEN_KEY);
  localStorage.removeItem(REFRESH_TOKEN_KEY);
  localStorage.removeItem(USER_KEY);
  document.cookie = buildCookieRemoval(ACCESS_TOKEN_COOKIE);
  document.cookie = buildCookieRemoval(REFRESH_TOKEN_COOKIE);
  document.cookie = buildCookieRemoval(USER_COOKIE);
}
