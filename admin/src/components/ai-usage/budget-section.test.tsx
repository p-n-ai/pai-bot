import { render, screen } from "@testing-library/react";
import { vi } from "vitest";
import { AIUsageBudgetSection } from "@/components/ai-usage/budget-section";
import type { AIUsageView } from "@/components/ai-usage/types";

vi.mock("@/components/token-budget-editor", () => ({
  TokenBudgetEditor: () => <div>Token budget editor</div>,
}));

const view: AIUsageView = {
  totalTokens: 413,
  total_messages: 6,
  topProvider: {
    provider: "openai",
    model: "gpt-5.4-mini",
    messages: 6,
    input_tokens: 200,
    output_tokens: 213,
    total_tokens: 413,
  },
  budgetStatus: { label: "Within token budget", tone: "success" },
  budget_period_start: "2026-04-01",
  budget_period_end: "2026-05-01",
  per_student_average_tokens: 137.7,
  per_student_average_cost_usd: null,
  budgetTokenLimit: 5000,
  budgetTokenRemaining: 4600,
  daily_usage: [],
  hasDailyTrend: false,
  dailyTrendPeak: 0,
  providers: [],
  monthlyCost: null,
  budgetLimit: null,
};

describe("AIUsageBudgetSection", () => {
  it("shows the budget editor for admins", () => {
    render(<AIUsageBudgetSection view={view} canManageBudget />);

    expect(screen.getByText("Token budget editor")).toBeInTheDocument();
    expect(screen.queryByText("Budget changes require admin access.")).not.toBeInTheDocument();
  });

  it("hides the budget editor for read-only roles", () => {
    render(<AIUsageBudgetSection view={view} canManageBudget={false} />);

    expect(screen.queryByText("Token budget editor")).not.toBeInTheDocument();
    expect(screen.getByText("Budget changes require admin access.")).toBeInTheDocument();
  });
});
