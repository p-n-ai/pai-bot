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

const mockClassProgress: ClassProgress = {
  topic_ids: ["linear-equations", "algebraic-expressions", "inequalities", "functions"],
  students: [
    {
      id: "stu_1",
      name: "Alya Sofea",
      topics: {
        "linear-equations": 0.86,
        "algebraic-expressions": 0.62,
        inequalities: 0.44,
        functions: 0.3,
      },
    },
    {
      id: "stu_2",
      name: "Hakim Firdaus",
      topics: {
        "linear-equations": 0.38,
        "algebraic-expressions": 0.57,
        inequalities: 0.21,
        functions: 0.18,
      },
    },
    {
      id: "stu_3",
      name: "Mei Lin",
      topics: {
        "linear-equations": 0.92,
        "algebraic-expressions": 0.84,
        inequalities: 0.74,
        functions: 0.59,
      },
    },
  ],
};

const mockStudentDetails: Record<
  string,
  {
    student: Student;
    progress: ProgressItem[];
    streak: { current: number; longest: number; total_xp: number };
  }
> = {
  stu_1: {
    student: {
      id: "stu_1",
      name: "Alya Sofea",
      external_id: "tg_10001",
      channel: "telegram",
      form: "Form 1",
      created_at: "2026-03-01T08:00:00Z",
    },
    progress: [
      {
        topic_id: "linear-equations",
        mastery_score: 0.86,
        ease_factor: 2.5,
        interval_days: 6,
        next_review_at: "2026-03-12T09:00:00Z",
        last_studied_at: "2026-03-09T11:20:00Z",
      },
      {
        topic_id: "algebraic-expressions",
        mastery_score: 0.62,
        ease_factor: 2.2,
        interval_days: 4,
        next_review_at: "2026-03-11T09:00:00Z",
        last_studied_at: "2026-03-09T11:45:00Z",
      },
      {
        topic_id: "inequalities",
        mastery_score: 0.44,
        ease_factor: 1.9,
        interval_days: 2,
        next_review_at: "2026-03-10T15:00:00Z",
        last_studied_at: "2026-03-08T14:10:00Z",
      },
      {
        topic_id: "functions",
        mastery_score: 0.3,
        ease_factor: 1.8,
        interval_days: 1,
        next_review_at: "2026-03-10T18:00:00Z",
        last_studied_at: "2026-03-08T14:40:00Z",
      },
    ],
    streak: { current: 5, longest: 9, total_xp: 1240 },
  },
  stu_2: {
    student: {
      id: "stu_2",
      name: "Hakim Firdaus",
      external_id: "tg_10002",
      channel: "telegram",
      form: "Form 1",
      created_at: "2026-02-27T08:00:00Z",
    },
    progress: [
      {
        topic_id: "linear-equations",
        mastery_score: 0.38,
        ease_factor: 1.9,
        interval_days: 2,
        next_review_at: "2026-03-10T15:00:00Z",
        last_studied_at: "2026-03-09T10:30:00Z",
      },
      {
        topic_id: "algebraic-expressions",
        mastery_score: 0.57,
        ease_factor: 2.1,
        interval_days: 3,
        next_review_at: "2026-03-11T09:00:00Z",
        last_studied_at: "2026-03-09T10:50:00Z",
      },
      {
        topic_id: "inequalities",
        mastery_score: 0.21,
        ease_factor: 1.7,
        interval_days: 1,
        next_review_at: "2026-03-10T18:00:00Z",
        last_studied_at: "2026-03-09T11:05:00Z",
      },
      {
        topic_id: "functions",
        mastery_score: 0.18,
        ease_factor: 1.6,
        interval_days: 1,
        next_review_at: "2026-03-10T19:00:00Z",
        last_studied_at: "2026-03-09T11:18:00Z",
      },
    ],
    streak: { current: 2, longest: 4, total_xp: 610 },
  },
  stu_3: {
    student: {
      id: "stu_3",
      name: "Mei Lin",
      external_id: "tg_10003",
      channel: "telegram",
      form: "Form 2",
      created_at: "2026-02-20T08:00:00Z",
    },
    progress: [
      {
        topic_id: "linear-equations",
        mastery_score: 0.92,
        ease_factor: 2.6,
        interval_days: 7,
        next_review_at: "2026-03-14T09:00:00Z",
        last_studied_at: "2026-03-09T09:10:00Z",
      },
      {
        topic_id: "algebraic-expressions",
        mastery_score: 0.84,
        ease_factor: 2.5,
        interval_days: 6,
        next_review_at: "2026-03-13T09:00:00Z",
        last_studied_at: "2026-03-09T09:35:00Z",
      },
      {
        topic_id: "inequalities",
        mastery_score: 0.74,
        ease_factor: 2.3,
        interval_days: 4,
        next_review_at: "2026-03-12T09:00:00Z",
        last_studied_at: "2026-03-09T09:55:00Z",
      },
      {
        topic_id: "functions",
        mastery_score: 0.59,
        ease_factor: 2.1,
        interval_days: 3,
        next_review_at: "2026-03-11T09:00:00Z",
        last_studied_at: "2026-03-09T10:15:00Z",
      },
    ],
    streak: { current: 10, longest: 15, total_xp: 2330 },
  },
};

const mockConversations: Record<string, StudentConversation[]> = {
  stu_1: [
    {
      id: "msg_1",
      timestamp: "2026-03-09T11:20:00Z",
      role: "student",
      text: "Why does the inequality sign flip when multiplying by negative numbers?",
    },
    {
      id: "msg_2",
      timestamp: "2026-03-09T11:20:12Z",
      role: "assistant",
      text: "Because multiplying by a negative reverses the order on the number line.",
    },
  ],
  stu_2: [
    {
      id: "msg_3",
      timestamp: "2026-03-09T10:30:00Z",
      role: "student",
      text: "I keep getting lost after moving terms to the other side.",
    },
    {
      id: "msg_4",
      timestamp: "2026-03-09T10:30:14Z",
      role: "assistant",
      text: "Let's keep the same operation on both sides instead of memorising a move.",
    },
  ],
  stu_3: [
    {
      id: "msg_5",
      timestamp: "2026-03-09T09:35:00Z",
      role: "student",
      text: "Can I try a harder algebraic expression problem?",
    },
    {
      id: "msg_6",
      timestamp: "2026-03-09T09:35:10Z",
      role: "assistant",
      text: "Yes. Expand 3(2x - 4) - 2(x + 5), then simplify.",
    },
  ],
};

async function fetchWithFallback<T>(path: string, fallback: T): Promise<T> {
  try {
    const res = await fetch(`${API_BASE}${path}`, {
      headers: { Authorization: `Bearer ${getToken()}` },
      cache: "no-store",
    });
    if (!res.ok) return fallback;
    return (await res.json()) as T;
  } catch {
    return fallback;
  }
}

export async function getClassProgress(classId: string): Promise<ClassProgress> {
  return fetchWithFallback(`/api/admin/classes/${classId}/progress`, mockClassProgress);
}

export async function getStudentDetail(studentId: string): Promise<{
  student: Student;
  progress: ProgressItem[];
  streak: { current: number; longest: number; total_xp: number };
}> {
  return fetchWithFallback(
    `/api/admin/students/${studentId}`,
    mockStudentDetails[studentId] || mockStudentDetails.stu_1
  );
}

export async function getStudentConversations(studentId: string): Promise<StudentConversation[]> {
  return fetchWithFallback(
    `/api/admin/students/${studentId}/conversations`,
    mockConversations[studentId] || []
  );
}

function getToken(): string {
  if (typeof window !== "undefined") {
    return localStorage.getItem("pai_token") || "";
  }
  return "";
}
