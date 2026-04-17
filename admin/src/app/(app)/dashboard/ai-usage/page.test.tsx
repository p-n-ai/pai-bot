import { render, screen } from "@testing-library/react";
import { vi } from "vitest";
import AIUsagePage from "./page";

const { getServerAIUsageMock, getServerAuthSessionMock, getAIUsageBudgetViewModelMock } = vi.hoisted(() => ({
  getServerAIUsageMock: vi.fn(),
  getServerAuthSessionMock: vi.fn(),
  getAIUsageBudgetViewModelMock: vi.fn(),
}));

vi.mock("@/lib/server-api", () => ({
  ServerAPIError: class ServerAPIError extends Error {
    constructor(public path: string, public status: number) {
      super(`Failed to load ${path}: ${status}`);
      this.name = "ServerAPIError";
    }
  },
  getServerAIUsage: getServerAIUsageMock,
  getServerAuthSession: getServerAuthSessionMock,
}));

vi.mock("@/lib/ai-usage.mjs", () => ({
  getAIUsageBudgetViewModel: getAIUsageBudgetViewModelMock,
}));

vi.mock("@/components/ai-usage/overview-section", () => ({
  AIUsageOverviewSection: () => <div>overview</div>,
}));

vi.mock("@/components/ai-usage/daily-trend-section", () => ({
  AIUsageDailyTrendSection: () => <div>daily trend</div>,
}));

vi.mock("@/components/ai-usage/provider-breakdown-section", () => ({
  AIUsageProviderBreakdownSection: () => <div>provider breakdown</div>,
}));

vi.mock("@/components/ai-usage/budget-section", () => ({
  AIUsageBudgetSection: ({ canManageBudget }: { canManageBudget: boolean }) => (
    <div>budget:{canManageBudget ? "editable" : "readonly"}</div>
  ),
}));

describe("AIUsagePage", () => {
  beforeEach(() => {
    getServerAIUsageMock.mockReset();
    getServerAuthSessionMock.mockReset();
    getAIUsageBudgetViewModelMock.mockReset();

    getServerAIUsageMock.mockResolvedValue({});
    getAIUsageBudgetViewModelMock.mockReturnValue({});
  });

  it("keeps the budget panel read-only for platform admins", async () => {
    getServerAuthSessionMock.mockResolvedValue({
      user: {
        user_id: "platform-admin-1",
        tenant_id: "tenant-1",
        role: "platform_admin",
        name: "Platform Admin",
        email: "platform@example.com",
      },
    });

    render(await AIUsagePage());

    expect(screen.getByText("budget:readonly")).toBeInTheDocument();
  });

  it("allows tenant admins to edit the budget panel", async () => {
    getServerAuthSessionMock.mockResolvedValue({
      user: {
        user_id: "admin-1",
        tenant_id: "tenant-1",
        role: "admin",
        name: "Admin",
        email: "admin@example.com",
      },
    });

    render(await AIUsagePage());

    expect(screen.getByText("budget:editable")).toBeInTheDocument();
  });
});
