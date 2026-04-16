import { cookies } from "next/headers";
import { normalizeClassProgress } from "@/lib/class-progress.mjs";
import { readJSONResponse } from "@/lib/http-response.mjs";
import { normalizeAIUsage } from "@/lib/ai-usage.mjs";
import { normalizeMetrics } from "@/lib/metrics.mjs";
import { canAccessPath, getDefaultRouteForUser } from "@/lib/rbac.mjs";
import type {
  AIUsageSummary,
  AuthUser,
  AuthSession,
  ClassProgress,
  ConversationExportRecord,
  MetricsSummary,
  OnboardingView,
  ParentSummary,
  UserManagementView,
  JoinClassView,
} from "@/lib/api";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

export class ServerAPIError extends Error {
  status: number;
  path: string;

  constructor(path: string, status: number) {
    super(`Failed to load ${path}: ${status}`);
    this.name = "ServerAPIError";
    this.path = path;
    this.status = status;
  }
}

async function requestCookieHeader(): Promise<string> {
  const cookieStore = await cookies();
  return cookieStore
    .getAll()
    .map(({ name, value }) => `${name}=${value}`)
    .join("; ");
}

async function fetchServerJSON<T>(path: string): Promise<T> {
  const cookieHeader = await requestCookieHeader();

  const res = await fetch(`${API_BASE}${path}`, {
    headers: cookieHeader ? { Cookie: cookieHeader } : undefined,
    cache: "no-store",
  });

  if (!res.ok) {
    throw new ServerAPIError(path, res.status);
  }

  return (await readJSONResponse(res)) as T;
}

export async function getServerAuthSession(): Promise<AuthSession | null> {
  const cookieHeader = await requestCookieHeader();

  const res = await fetch(`${API_BASE}/api/auth/session`, {
    headers: cookieHeader ? { Cookie: cookieHeader } : undefined,
    cache: "no-store",
  });
  if (res.status >= 400 && res.status < 500) {
    return null;
  }
  if (!res.ok) {
    throw new Error(`Failed to load /api/auth/session: ${res.status}`);
  }
  return (await readJSONResponse(res)) as AuthSession;
}

export async function getServerClassProgress(classID: string): Promise<ClassProgress> {
  return normalizeClassProgress(await fetchServerJSON(`/api/admin/classes/${classID}/progress`)) as ClassProgress;
}

export async function getServerAIUsage(): Promise<AIUsageSummary> {
  return normalizeAIUsage(await fetchServerJSON(`/api/admin/ai/usage`)) as AIUsageSummary;
}

export async function getServerMetrics(): Promise<MetricsSummary> {
  return normalizeMetrics(await fetchServerJSON(`/api/admin/metrics`)) as MetricsSummary;
}

export async function getServerParentSummary(parentID: string): Promise<ParentSummary> {
  return fetchServerJSON(`/api/admin/parents/${parentID}`);
}

export async function getServerUserManagement(): Promise<UserManagementView> {
  return fetchServerJSON(`/api/admin/users`);
}

export async function getServerOnboarding(): Promise<OnboardingView> {
  return fetchServerJSON(`/api/admin/onboarding`);
}

export async function getServerJoinClass(slug: string): Promise<JoinClassView> {
  return fetchServerJSON(`/api/join/${encodeURIComponent(slug)}`);
}

export async function getServerPostAuthPath(user: AuthUser, nextPath?: string | null): Promise<string> {
  if (nextPath && nextPath !== "/" && nextPath !== "/login" && canAccessPath(user, nextPath)) {
    return nextPath;
  }

  if (user.role === "admin" || user.role === "platform_admin") {
    try {
      const onboarding = await getServerOnboarding();
      if (!onboarding.onboarding) {
        return "/setup/onboard";
      }
    } catch {
      // Fall back to the normal role route when onboarding state is unavailable.
    }
  }

  return getDefaultRouteForUser(user);
}

export async function getServerConversationExport(): Promise<ConversationExportRecord[]> {
  return fetchServerJSON(`/api/admin/export/conversations`);
}
