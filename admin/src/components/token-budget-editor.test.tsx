import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { vi } from "vitest";
import { TokenBudgetEditor } from "@/components/token-budget-editor";

const { refresh, upsertTokenBudgetWindow } = vi.hoisted(() => ({
  refresh: vi.fn(),
  upsertTokenBudgetWindow: vi.fn(),
}));

vi.mock("next/navigation", () => ({
  useRouter: () => ({
    refresh,
  }),
}));

vi.mock("@/lib/api", () => ({
  upsertTokenBudgetWindow,
}));

describe("TokenBudgetEditor", () => {
  it("renders the token budget fields", () => {
    render(<TokenBudgetEditor initialBudgetTokens={250000} initialPeriodStart="2026-04-01" initialPeriodEnd="2026-04-30" />);

    expect(screen.getByLabelText("Token limit")).toHaveValue(250000);
    expect(screen.getByLabelText("Start date")).toHaveValue("2026-04-01");
    expect(screen.getByLabelText("End date")).toHaveValue("2026-04-30");
    expect(screen.getByRole("button", { name: "Save token budget" })).toBeInTheDocument();
  });

  it("submits the token budget window and refreshes the page", async () => {
    upsertTokenBudgetWindow.mockResolvedValue({});

    render(<TokenBudgetEditor initialBudgetTokens={null} initialPeriodStart="2026-04-01" initialPeriodEnd="2026-04-30" />);

    fireEvent.change(screen.getByLabelText("Token limit"), { target: { value: "300000" } });
    fireEvent.click(screen.getByRole("button", { name: "Save token budget" }));

    await waitFor(() =>
      expect(upsertTokenBudgetWindow).toHaveBeenCalledWith({
        budget_tokens: 300000,
        period_start: "2026-04-01",
        period_end: "2026-04-30",
      }),
    );
    expect(refresh).toHaveBeenCalled();
  });
});
