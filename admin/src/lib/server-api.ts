import { cookies } from "next/headers";
import { ACCESS_TOKEN_COOKIE } from "@/lib/auth-session";
import { normalizeClassProgress } from "@/lib/class-progress.mjs";
import { readJSONResponse } from "@/lib/http-response.mjs";
import { normalizeAIUsage } from "@/lib/ai-usage.mjs";
import type { AIUsageSummary, ClassProgress, ParentSummary } from "@/lib/api";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

async function fetchServerJSON<T>(path: string): Promise<T> {
  const cookieStore = await cookies();
  const accessToken = cookieStore.get(ACCESS_TOKEN_COOKIE)?.value || "";

  const res = await fetch(`${API_BASE}${path}`, {
    headers: accessToken ? { Authorization: `Bearer ${accessToken}` } : undefined,
    cache: "no-store",
  });

  if (!res.ok) {
    throw new Error(`Failed to load ${path}: ${res.status}`);
  }

  return (await readJSONResponse(res)) as T;
}

export async function getServerClassProgress(classID: string): Promise<ClassProgress> {
  return normalizeClassProgress(await fetchServerJSON(`/api/admin/classes/${classID}/progress`)) as ClassProgress;
}

export async function getServerAIUsage(): Promise<AIUsageSummary> {
  return normalizeAIUsage(await fetchServerJSON(`/api/admin/ai/usage`)) as AIUsageSummary;
}

export async function getServerParentSummary(parentID: string): Promise<ParentSummary> {
  return fetchServerJSON(`/api/admin/parents/${parentID}`);
}
