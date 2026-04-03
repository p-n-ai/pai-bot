"use client";

import type { AuthSession, AuthUser } from "@/lib/api";

export function readStoredUser(): AuthUser | null {
  return null;
}

export function readStoredAccessToken(): string {
  return "";
}

export function readStoredRefreshToken(): string {
  return "";
}

export function hasStoredSession(): boolean {
  return false;
}

export function writeStoredSession(session: AuthSession): void {
  void session;
}

export function removeStoredSession(): void {
  return;
}
