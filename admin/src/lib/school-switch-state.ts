import type { TenantChoice } from "@/lib/api";

const SCHOOL_SWITCH_STATE_KEY = "pai_school_switch_state";
export const SCHOOL_SWITCH_STATE_COOKIE = "pai_admin_school_switch_state";
const SCHOOL_SWITCH_MAX_AGE = 60 * 60 * 24 * 7;

export type SchoolSwitchState = {
  email: string;
  currentTenantID: string;
  tenantChoices: TenantChoice[];
};

export function buildSchoolSwitchState(email: string, currentTenantID: string, tenantChoices: TenantChoice[]): SchoolSwitchState | null {
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
  document.cookie = `${SCHOOL_SWITCH_STATE_COOKIE}=${encodeURIComponent(JSON.stringify(state))}; Path=/; Max-Age=${SCHOOL_SWITCH_MAX_AGE}; SameSite=Lax`;
}

export function clearSchoolSwitchState(): void {
  if (typeof window === "undefined") return;
  localStorage.removeItem(SCHOOL_SWITCH_STATE_KEY);
  document.cookie = `${SCHOOL_SWITCH_STATE_COOKIE}=; Path=/; Max-Age=0; SameSite=Lax`;
}
