import { cookies } from "next/headers";
import { normalizeClassProgress } from "@/lib/class-progress.mjs";
import { readJSONResponse } from "@/lib/http-response.mjs";
import { normalizeAIUsage } from "@/lib/ai-usage.mjs";
import { normalizeMetrics } from "@/lib/metrics.mjs";
import type { AIUsageSummary, AuthSession, ClassProgress, MetricsSummary, ParentSummary } from "@/lib/api";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

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
    throw new Error(`Failed to load ${path}: ${res.status}`);
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
