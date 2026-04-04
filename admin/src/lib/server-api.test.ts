import { beforeEach, describe, expect, it, vi } from "vitest";
import { getServerAuthSession } from "@/lib/server-api";

vi.mock("next/headers", () => ({
  cookies: vi.fn(async () => ({
    getAll: () => [{ name: "pai_session", value: "stale-token" }],
  })),
}));

describe("getServerAuthSession", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it("treats 400 session responses as signed-out during SSR bootstrap", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response("bad session", { status: 400 })));

    await expect(getServerAuthSession()).resolves.toBeNull();
  });

  it("returns the session payload when the backend accepts the cookie", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            expires_at: "2026-04-05T00:00:00Z",
            user: {
              user_id: "user-1",
              tenant_id: "tenant-1",
              tenant_name: "Demo School",
              role: "admin",
              name: "Admin User",
              email: "admin@example.com",
            },
          }),
          {
            status: 200,
            headers: { "Content-Type": "application/json" },
          },
        ),
      ),
    );

    await expect(getServerAuthSession()).resolves.toMatchObject({
      user: {
        user_id: "user-1",
        tenant_id: "tenant-1",
        tenant_name: "Demo School",
      },
    });
  });
});
