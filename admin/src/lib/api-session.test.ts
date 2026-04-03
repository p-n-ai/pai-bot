import { beforeEach, describe, expect, it } from "vitest";
import { clearSession } from "@/lib/api";
import { ACCESS_TOKEN_KEY, REFRESH_TOKEN_KEY, USER_KEY } from "@/lib/auth-session";
import { SCHOOL_SWITCH_STATE_COOKIE } from "@/lib/school-switch-state";

describe("clearSession", () => {
  beforeEach(() => {
    window.localStorage.clear();
    document.cookie = `${SCHOOL_SWITCH_STATE_COOKIE}=; Path=/; Max-Age=0; SameSite=Lax`;
    document.cookie = "pai_admin_access=; Path=/; Max-Age=0; SameSite=Lax";
    document.cookie = "pai_admin_refresh=; Path=/; Max-Age=0; SameSite=Lax";
    document.cookie = "pai_admin_user=; Path=/; Max-Age=0; SameSite=Lax";
  });

  it("clears auth and school-switch storage together", () => {
    window.localStorage.setItem(ACCESS_TOKEN_KEY, "access-token");
    window.localStorage.setItem(REFRESH_TOKEN_KEY, "refresh-token");
    window.localStorage.setItem(USER_KEY, JSON.stringify({ user_id: "teacher-1", email: "teacher@example.com" }));
    window.localStorage.setItem(
      "pai_school_switch_state",
      JSON.stringify({
        email: "teacher@example.com",
        currentTenantID: "tenant-a",
        tenantChoices: [{ tenant_id: "tenant-a", tenant_name: "School A", tenant_slug: "school-a" }],
      }),
    );
    document.cookie = `${SCHOOL_SWITCH_STATE_COOKIE}=${encodeURIComponent(
      JSON.stringify({ email: "teacher@example.com", currentTenantID: "tenant-a", tenantChoices: [] }),
    )}; Path=/; Max-Age=60; SameSite=Lax`;

    clearSession();

    expect(window.localStorage.getItem(ACCESS_TOKEN_KEY)).toBeNull();
    expect(window.localStorage.getItem(REFRESH_TOKEN_KEY)).toBeNull();
    expect(window.localStorage.getItem(USER_KEY)).toBeNull();
    expect(window.localStorage.getItem("pai_school_switch_state")).toBeNull();
    expect(document.cookie).not.toContain(SCHOOL_SWITCH_STATE_COOKIE);
    expect(document.cookie).not.toContain("pai_admin_refresh");
  });
});
