"use client";

import { getClassProgress, type ClassProgress } from "@/lib/api";
import { getMockClassProgress } from "@/lib/mock-classes.mjs";

export type DashboardProgressResult = {
  progress: ClassProgress;
  source: "live" | "preview";
  issue?: string;
};

export function getDashboardProgressQueryKey(tenantID: string) {
  return ["dashboard-progress", tenantID] as const;
}

export async function fetchDashboardProgress(tenantID: string): Promise<DashboardProgressResult> {
  void tenantID;
  const liveProgress = await getClassProgress("all-students");
  return {
    progress: liveProgress,
    source: "live",
  };
}

export async function fetchPreviewDashboardProgress(): Promise<DashboardProgressResult> {
  try {
    return {
      progress: getMockClassProgress("all-students"),
      source: "preview",
    };
  } catch (error) {
    return {
      progress: { students: [], topic_ids: [] },
      source: "preview",
      issue: error instanceof Error ? error.message : "Class data is unavailable right now.",
    };
  }
}
