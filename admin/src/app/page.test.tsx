import { render, screen } from "@testing-library/react";
import { vi } from "vitest";
import RootPage from "./page";

const { getServerAuthSessionMock, getServerPostAuthPathMock } = vi.hoisted(() => ({
  getServerAuthSessionMock: vi.fn(),
  getServerPostAuthPathMock: vi.fn(),
}));

vi.mock("@/lib/server-api", () => ({
  getServerAuthSession: getServerAuthSessionMock,
  getServerPostAuthPath: getServerPostAuthPathMock,
}));

describe("RootPage", () => {
  beforeEach(() => {
    getServerAuthSessionMock.mockReset();
    getServerPostAuthPathMock.mockReset();
  });

  it("renders the landing page with login CTA for signed-out visitors", async () => {
    getServerAuthSessionMock.mockResolvedValue(null);

    render(await RootPage({ searchParams: Promise.resolve({}) }));

    expect(screen.getByRole("heading", { name: /learn math in chat\./i })).toBeInTheDocument();
    expect(screen.getAllByRole("link", { name: "Sign in" })[0]).toHaveAttribute("href", "/login");
    expect(getServerPostAuthPathMock).not.toHaveBeenCalled();
  });

  it("keeps the landing page visible and points signed-in users to their workspace", async () => {
    getServerAuthSessionMock.mockResolvedValue({
      user: {
        user_id: "teacher-1",
        tenant_id: "tenant-1",
        role: "teacher",
        name: "Teacher",
        email: "teacher@example.com",
      },
    });
    getServerPostAuthPathMock.mockResolvedValue("/dashboard");

    render(await RootPage({ searchParams: Promise.resolve({ next: "/dashboard/classes" }) }));

    expect(screen.getAllByRole("link", { name: "Open workspace" })[0]).toHaveAttribute("href", "/dashboard");
    expect(screen.queryByRole("link", { name: "Sign in" })).not.toBeInTheDocument();
    expect(getServerPostAuthPathMock).toHaveBeenCalledWith(
      expect.objectContaining({ role: "teacher" }),
      "/dashboard/classes",
    );
  });
});
