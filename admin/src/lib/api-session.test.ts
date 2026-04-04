import { beforeEach, describe, expect, it, vi } from "vitest";
import { buildGoogleLinkURL, buildGoogleLoginURL, clearSession, startGoogleLink } from "@/lib/api";

describe("clearSession", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  it("clears school-switch storage together with the client session store", () => {
    window.localStorage.setItem(
      "pai_school_switch_state",
      JSON.stringify({
        email: "teacher@example.com",
        currentTenantID: "tenant-a",
        tenantChoices: [{ tenant_id: "tenant-a", tenant_name: "School A", tenant_slug: "school-a" }],
      }),
    );
    clearSession();

    expect(window.localStorage.getItem("pai_school_switch_state")).toBeNull();
  });

  it("builds Google login and link URLs with a next path when provided", () => {
    expect(buildGoogleLoginURL("/dashboard")).toBe("/api/auth/google/start?next=%2Fdashboard");
    expect(buildGoogleLinkURL("/dashboard/settings")).toBe("/api/auth/google/link/start?next=%2Fdashboard%2Fsettings");
  });

  it("starts Google link with an authenticated POST and returns the redirect URL", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ url: "https://accounts.google.com/o/oauth2/v2/auth?state=abc" }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );
    vi.stubGlobal("fetch", fetchMock);

    await expect(startGoogleLink("/dashboard/settings")).resolves.toBe(
      "https://accounts.google.com/o/oauth2/v2/auth?state=abc",
    );
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/auth/google/link/start?next=%2Fdashboard%2Fsettings",
      expect.objectContaining({
        method: "POST",
        credentials: "include",
        cache: "no-store",
      }),
    );

    vi.unstubAllGlobals();
  });
});
