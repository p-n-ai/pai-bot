import { normalizeClassProgress } from "@/lib/class-progress.mjs";
import { normalizeMetrics } from "@/lib/metrics.mjs";
import {
  hasStoredSession as hasPersistedSession,
  readStoredAccessToken,
  readStoredRefreshToken,
  readStoredUser,
  removeStoredSession,
  writeStoredSession,
} from "@/lib/client-session";
import { clearSchoolSwitchState } from "@/lib/school-switch-state";
import { applyAdminSessionToStore, clearAdminSessionStore } from "@/stores/app-store";
import { readJSONResponse } from "@/lib/http-response.mjs";

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
  monthly_cost_usd?: number | null;
  budget_limit_usd?: number | null;
  per_student_average_tokens?: number | null;
  per_student_average_cost_usd?: number | null;
  budget_limit_tokens?: number | null;
  budget_used_tokens?: number | null;
  budget_remaining_tokens?: number | null;
  budget_period_start?: string;
  budget_period_end?: string;
  daily_usage?: {
    date: string;
    messages: number;
    tokens: number;
    cost_usd?: number | null;
  }[];
  provider_costs?: {
    provider: string;
    cost_usd?: number | null;
  }[];
}

export interface UpsertTokenBudgetWindowInput {
  budget_tokens: number;
  period_start: string;
  period_end: string;
}

export interface MetricsSummary {
  window_days: number;
  daily_active_users: { date: string; users: number }[];
  retention: {
    cohort_date: string;
    cohort_size: number;
    day_1_rate: number;
    day_7_rate: number;
    day_14_rate: number;
  }[];
  nudge_rate: {
    nudges_sent: number;
    responses_within_24h: number;
    response_rate: number;
  };
  ai_usage: AIUsageSummary;
  ab_comparison: {
    experiment_key?: string;
    window_days?: number | null;
    metric_name?: string;
    variant_a?: {
      label?: string;
      users?: number;
      retention_rate?: number;
      challenge_participation_rate?: number;
      leaderboard_engagement_rate?: number;
      nudge_response_rate?: number;
    };
    variant_b?: {
      label?: string;
      users?: number;
      retention_rate?: number;
      challenge_participation_rate?: number;
      leaderboard_engagement_rate?: number;
      nudge_response_rate?: number;
    };
    winner?: string;
    delta_retention_rate?: number | null;
    delta_challenge_participation_rate?: number | null;
    delta_leaderboard_engagement_rate?: number | null;
    delta_nudge_response_rate?: number | null;
  } | null;
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

export interface InviteRecord {
  email: string;
  role: "teacher" | "parent" | "admin" | "platform_admin";
  invite_token: string;
  expires_at: string;
  invited_by_user_id: string;
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

function parseErrorMessage(raw: string, fallback: string): string {
  if (!raw.trim()) {
    return fallback;
  }

  try {
    const payload = JSON.parse(raw) as { error?: string };
    return payload.error || fallback;
  } catch {
    return raw;
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

export async function upsertTokenBudgetWindow(input: UpsertTokenBudgetWindowInput): Promise<AIUsageSummary> {
  return postJSONWithBody(`/api/admin/ai/budget-window`, input);
}

export async function getMetrics(): Promise<MetricsSummary> {
  return normalizeMetrics(await fetchJSON(`/api/admin/metrics`));
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
      throw new LoginError(parseErrorMessage(raw, `Login failed: ${res.status}`));
    }
  }

  return (await readJSONResponse(res)) as AuthSession;
}

export async function acceptInvite(input: {
  token: string;
  name: string;
  password: string;
}): Promise<AuthSession> {
  const res = await fetch(`${API_BASE}/api/auth/invitations/accept`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(input),
  });

  if (!res.ok) {
    const raw = await res.text();
    throw new Error(parseErrorMessage(raw, `Invite activation failed: ${res.status}`));
  }

  return (await readJSONResponse(res)) as AuthSession;
}

export async function issueInvite(input: {
  email: string;
  role: "teacher" | "parent" | "admin";
}): Promise<InviteRecord> {
  return postJSONWithBody("/api/admin/invites", input);
}

export async function switchTenantSession(tenantID: string, password: string): Promise<AuthSession> {
  if (typeof window === "undefined") {
    throw new Error("Tenant switching is only available in the browser");
  }

  const refreshToken = readStoredRefreshToken() || "";
  if (!refreshToken || !tenantID.trim() || !password.trim()) {
    throw new Error("A stored session is required to switch schools");
  }

  const res = await fetch(`${API_BASE}/api/auth/switch-tenant`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      refresh_token: refreshToken,
      tenant_id: tenantID,
      password,
    }),
  });

  if (!res.ok) {
    const raw = await res.text();
    throw new Error(parseErrorMessage(raw, `Tenant switch failed: ${res.status}`));
  }

  return (await readJSONResponse(res)) as AuthSession;
}

export function persistSession(session: AuthSession): void {
  if (typeof window === "undefined") return;

  writeStoredSession(session);
  applyAdminSessionToStore(session);
}

export function clearSession(): void {
  if (typeof window === "undefined") return;

  removeStoredSession();
  clearSchoolSwitchState();
  clearAdminSessionStore();
}

export function getStoredUser(): AuthUser | null {
  return readStoredUser();
}

export function getStoredAccessToken(): string {
  return readStoredAccessToken();
}

export function hasStoredSession(): boolean {
  return hasPersistedSession();
}

export async function logout(): Promise<void> {
  if (typeof window === "undefined") return;

  const refreshToken = readStoredRefreshToken() || "";

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
