import { normalizeClassProgress } from "@/lib/class-progress.mjs";

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

export interface NudgeResponse {
  status: string;
  student: string;
  channel: string;
}

async function fetchJSON<T>(path: string): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    headers: { Authorization: `Bearer ${getToken()}` },
    cache: "no-store",
  });
  if (!res.ok) {
    throw new Error(`Failed to load ${path}: ${res.status}`);
  }
  return (await res.json()) as T;
}

async function postJSON<T>(path: string): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${getToken()}`,
      "Content-Type": "application/json",
    },
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(text || `Failed to post ${path}: ${res.status}`);
  }
  return (await res.json()) as T;
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

export async function sendStudentNudge(studentId: string): Promise<NudgeResponse> {
  return postJSON(`/api/admin/students/${studentId}/nudge`);
}

function getToken(): string {
  if (typeof window !== "undefined") {
    return localStorage.getItem("pai_token") || "";
  }
  return "";
}
