import type { SchoolChoice } from "@/lib/api";

const SCHOOL_SWITCH_STATE_KEY = "pai_school_switch_state";

export type SchoolSwitchState = {
  email: string;
  currentTenantID: string;
  tenantChoices: SchoolChoice[];
};

export function buildSchoolSwitchState(email: string, currentTenantID: string, tenantChoices: SchoolChoice[]): SchoolSwitchState | null {
  if (!email.trim() || !currentTenantID.trim() || tenantChoices.length <= 1) {
    return null;
  }
  return {
    email,
    currentTenantID,
    tenantChoices,
  };
}

export function readSchoolSwitchState(): SchoolSwitchState | null {
  if (typeof window === "undefined") return null;

  const raw = localStorage.getItem(SCHOOL_SWITCH_STATE_KEY);
  if (!raw) return null;

  try {
    const parsed = JSON.parse(raw) as SchoolSwitchState;
    if (
      typeof parsed?.email !== "string" ||
      typeof parsed?.currentTenantID !== "string" ||
      !Array.isArray(parsed?.tenantChoices)
    ) {
      return null;
    }
    return parsed;
  } catch {
    return null;
  }
}

export function writeSchoolSwitchState(state: SchoolSwitchState): void {
  if (typeof window === "undefined") return;
  localStorage.setItem(SCHOOL_SWITCH_STATE_KEY, JSON.stringify(state));
}

export function clearSchoolSwitchState(): void {
  if (typeof window === "undefined") return;
  localStorage.removeItem(SCHOOL_SWITCH_STATE_KEY);
}
