import { normalizeClassProgress } from "@/lib/class-progress.mjs";
import {
  ACCESS_TOKEN_COOKIE,
  ACCESS_TOKEN_KEY,
  buildCookieRemoval,
  buildCookieValue,
  REFRESH_TOKEN_KEY,
  SESSION_CHANGED_EVENT,
  USER_COOKIE,
  USER_KEY,
} from "@/lib/auth-session";
import { readJSONResponse } from "@/lib/http-response.mjs";
import { hasClientSession } from "@/lib/session-state.mjs";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

export interface Student {
  id: string;
  name: string;
  external_id: string;
  channel: string;
  form: string;
  created_at: string;
}

export interface ProgressItem {
  topic_id: string;
  mastery_score: number;
  ease_factor: number;
  interval_days: number;
  next_review_at: string | null;
  last_studied_at: string | null;
}

export interface ClassProgress {
  students: {
    id: string;
    name: string;
    topics: Record<string, number>;
  }[];
  topic_ids: string[];
}

export interface StudentConversation {
  id: string;
  timestamp: string;
  role: "student" | "assistant";
  text: string;
}

export interface ParentProfile {
  id: string;
  name: string;
  email: string;
  child_ids: string[];
  created_at: string;
}

export interface WeeklyStats {
  days_active: number;
  messages_exchanged: number;
  quizzes_completed: number;
  needs_review_count: number;
}

export interface EncouragementSuggestion {
  headline: string;
  text: string;
}

export interface ParentSummary {
  parent: ParentProfile;
  child: Student;
  streak: { current: number; longest: number; total_xp: number };
  weekly_stats: WeeklyStats;
  mastery: ProgressItem[];
  encouragement: EncouragementSuggestion;
}

export interface AIProviderUsage {
  provider: string;
  model: string;
  messages: number;
  input_tokens: number;
  output_tokens: number;
  total_tokens: number;
}

export interface AIUsageSummary {
  total_messages: number;
  total_input_tokens: number;
  total_output_tokens: number;
  providers: AIProviderUsage[];
}

export interface NudgeResponse {
  status: string;
  student: string;
  channel: string;
}

export interface AuthUser {
  user_id: string;
  tenant_id: string;
  tenant_slug?: string;
  tenant_name?: string;
  role: "student" | "teacher" | "parent" | "admin" | "platform_admin";
  name: string;
  email: string;
}

export interface AuthSession {
  access_token: string;
  refresh_token: string;
  access_expires_at: string;
  refresh_expires_at: string;
  user: AuthUser;
}

export interface TenantChoice {
  tenant_id: string;
  tenant_slug: string;
  tenant_name: string;
}

export class LoginError extends Error {
  code: "tenant_required" | "generic";
  tenants: TenantChoice[];

  constructor(message: string, options?: { code?: "tenant_required" | "generic"; tenants?: TenantChoice[] }) {
    super(message);
    this.name = "LoginError";
    this.code = options?.code ?? "generic";
    this.tenants = options?.tenants ?? [];
  }
}

async function fetchJSON<T>(path: string): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    headers: { Authorization: `Bearer ${getToken()}` },
    cache: "no-store",
  });
  if (!res.ok) {
    throw new Error(`Failed to load ${path}: ${res.status}`);
  }
  return (await readJSONResponse(res)) as T;
}

async function postJSON<T>(path: string): Promise<T> {
  return postJSONWithBody<T>(path, undefined);
}

async function postJSONWithBody<T>(path: string, body?: unknown): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${getToken()}`,
      "Content-Type": "application/json",
    },
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(text || `Failed to post ${path}: ${res.status}`);
  }
  return (await readJSONResponse(res)) as T;
}

export async function getClassProgress(classId: string): Promise<ClassProgress> {
  return normalizeClassProgress(await fetchJSON(`/api/admin/classes/${classId}/progress`)) as ClassProgress;
}

export async function getStudentDetail(studentId: string): Promise<{
  student: Student;
  progress: ProgressItem[];
  streak: { current: number; longest: number; total_xp: number };
}> {
  return fetchJSON(`/api/admin/students/${studentId}`);
}

export async function getStudentConversations(studentId: string): Promise<StudentConversation[]> {
  return fetchJSON(`/api/admin/students/${studentId}/conversations`);
}

export async function getParentSummary(parentId: string): Promise<ParentSummary> {
  return fetchJSON(`/api/admin/parents/${parentId}`);
}

export async function getAIUsage(): Promise<AIUsageSummary> {
  return fetchJSON(`/api/admin/ai/usage`);
}

export async function sendStudentNudge(studentId: string): Promise<NudgeResponse> {
  return postJSON(`/api/admin/students/${studentId}/nudge`);
}

export async function login(input: {
  tenant_id?: string;
  email: string;
  password: string;
}): Promise<AuthSession> {
  const res = await fetch(`${API_BASE}/api/auth/login`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(input),
  });

  if (!res.ok) {
    const raw = await res.text();

    try {
      const payload = JSON.parse(raw) as { error?: string; tenants?: TenantChoice[] };
      if (res.status === 400 && Array.isArray(payload.tenants) && payload.tenants.length > 0) {
        throw new LoginError(payload.error || "Select a tenant to continue", {
          code: "tenant_required",
          tenants: payload.tenants,
        });
      }
      throw new LoginError(payload.error || `Login failed: ${res.status}`);
    } catch (error) {
      if (error instanceof LoginError) {
        throw error;
      }
      throw new LoginError(raw || `Login failed: ${res.status}`);
    }
  }

  return (await readJSONResponse(res)) as AuthSession;
}

export function persistSession(session: AuthSession): void {
  if (typeof window === "undefined") return;

  localStorage.setItem(ACCESS_TOKEN_KEY, session.access_token);
  localStorage.setItem(REFRESH_TOKEN_KEY, session.refresh_token);
  localStorage.setItem(USER_KEY, JSON.stringify(session.user));
  document.cookie = buildCookieValue(ACCESS_TOKEN_COOKIE, session.access_token, 60 * 60 * 24 * 7);
  document.cookie = buildCookieValue(USER_COOKIE, JSON.stringify(session.user), 60 * 60 * 24 * 7);
  window.dispatchEvent(new Event(SESSION_CHANGED_EVENT));
}

export function clearSession(): void {
  if (typeof window === "undefined") return;

  localStorage.removeItem(ACCESS_TOKEN_KEY);
  localStorage.removeItem(REFRESH_TOKEN_KEY);
  localStorage.removeItem(USER_KEY);
  document.cookie = buildCookieRemoval(ACCESS_TOKEN_COOKIE);
  document.cookie = buildCookieRemoval(USER_COOKIE);
  window.dispatchEvent(new Event(SESSION_CHANGED_EVENT));
}

export function getStoredUser(): AuthUser | null {
  if (typeof window === "undefined") return null;

  const raw = localStorage.getItem(USER_KEY);
  if (!raw) return null;

  try {
    return JSON.parse(raw) as AuthUser;
  } catch {
    return null;
  }
}

export function getStoredAccessToken(): string {
  if (typeof window === "undefined") return "";

  return localStorage.getItem(ACCESS_TOKEN_KEY) || "";
}

export function hasStoredSession(): boolean {
  if (typeof window === "undefined") return false;

  return hasClientSession({
    accessToken: getStoredAccessToken(),
    user: getStoredUser(),
  });
}

export async function logout(): Promise<void> {
  if (typeof window === "undefined") return;

  const refreshToken = localStorage.getItem(REFRESH_TOKEN_KEY) || "";

  try {
    if (refreshToken) {
      await postJSONWithBody("/api/auth/logout", { refresh_token: refreshToken });
    }
  } finally {
    clearSession();
  }
}

function getToken(): string {
  if (typeof window !== "undefined") {
    return getStoredAccessToken();
  }
  return "";
}
