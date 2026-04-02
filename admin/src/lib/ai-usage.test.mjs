import test from "node:test";
import assert from "node:assert/strict";

import {
  formatCompactNumber,
  formatUSD,
  getAIUsageBudgetViewModel,
  getTopProvider,
  normalizeAIUsage,
} from "./ai-usage.mjs";

test("normalizeAIUsage fills missing fields with safe defaults", () => {
  const usage = normalizeAIUsage({
    total_messages: 6,
    providers: [{ provider: "openai", model: "gpt-4o-mini", input_tokens: 10, output_tokens: 5 }],
  });

  assert.equal(usage.total_messages, 6);
  assert.equal(usage.total_input_tokens, 0);
  assert.equal(usage.total_output_tokens, 0);
  assert.equal(usage.providers[0].messages, 0);
  assert.equal(usage.providers[0].total_tokens, 15);
});

test("getTopProvider returns the highest token consumer", () => {
  const top = getTopProvider({
    providers: [
      { provider: "openai", model: "gpt-4o-mini", total_tokens: 120 },
      { provider: "anthropic", model: "claude-3-5-haiku", total_tokens: 200 },
    ],
  });

  assert.deepEqual(top, {
    provider: "anthropic",
    model: "claude-3-5-haiku",
    total_tokens: 200,
  });
});

test("formatCompactNumber keeps admin stats readable", () => {
  assert.equal(formatCompactNumber(1250), "1.3K");
});

test("normalizeAIUsage keeps optional budget fields when provided", () => {
  const usage = normalizeAIUsage({
    monthly_cost_usd: 42.75,
    budget_limit_usd: 100,
    per_student_average_tokens: 812,
    per_student_average_cost_usd: 0.34,
    daily_usage: [{ date: "2026-03-28", tokens: 1200, cost_usd: 4.5 }],
    provider_costs: [{ provider: "openai", cost_usd: 12.2 }],
  });

  assert.equal(usage.monthly_cost_usd, 42.75);
  assert.equal(usage.budget_limit_usd, 100);
  assert.equal(usage.per_student_average_tokens, 812);
  assert.equal(usage.per_student_average_cost_usd, 0.34);
  assert.deepEqual(usage.daily_usage, [{ date: "2026-03-28", messages: 0, tokens: 1200, cost_usd: 4.5 }]);
  assert.deepEqual(usage.provider_costs, [{ provider: "openai", cost_usd: 12.2 }]);
});

test("getAIUsageBudgetViewModel derives remaining budget and status", () => {
  const view = getAIUsageBudgetViewModel({
    total_input_tokens: 100,
    total_output_tokens: 50,
    monthly_cost_usd: 24,
    budget_limit_usd: 30,
    provider_costs: [{ provider: "openai", cost_usd: 24 }],
  });

  assert.equal(view.totalTokens, 150);
  assert.equal(view.remainingBudget, 6);
  assert.equal(view.usageRatio, 0.8);
  assert.equal(view.budgetStatus.label, "Near budget limit");
  assert.equal(view.hasProviderCosts, true);
});

test("getAIUsageBudgetViewModel reports pending status when budget data is missing", () => {
  const view = getAIUsageBudgetViewModel({
    total_input_tokens: 10,
    total_output_tokens: 5,
  });

  assert.equal(view.remainingBudget, null);
  assert.equal(view.usageRatio, null);
  assert.equal(view.budgetStatus.label, "No token budget configured");
  assert.equal(view.hasDailyTrend, false);
  assert.equal(view.hasPerStudentAverages, false);
});

test("getAIUsageBudgetViewModel keeps token-centric signals when budget fields are absent", () => {
  const view = getAIUsageBudgetViewModel({
    total_messages: 8,
    total_input_tokens: 120,
    total_output_tokens: 30,
    per_student_average_tokens: 50,
    daily_usage: [
      { date: "2026-03-30", messages: 3, tokens: 45 },
      { date: "2026-03-31", messages: 5, tokens: 105 },
    ],
  });

  assert.equal(view.totalTokens, 150);
  assert.equal(view.hasDailyTrend, true);
  assert.equal(view.dailyTrendPeak, 105);
  assert.equal(view.hasPerStudentAverages, true);
  assert.equal(view.per_student_average_tokens, 50);
  assert.equal(view.budgetStatus.label, "No token budget configured");
});

test("getAIUsageBudgetViewModel reports token budget status when token limits are present", () => {
  const view = getAIUsageBudgetViewModel({
    total_input_tokens: 120,
    total_output_tokens: 30,
    budget_limit_tokens: 500,
    budget_used_tokens: 425,
    budget_remaining_tokens: 75,
    budget_period_start: "2026-03-01",
    budget_period_end: "2026-03-31",
  });

  assert.equal(view.budgetTokenLimit, 500);
  assert.equal(view.budgetTokenUsed, 425);
  assert.equal(view.budgetTokenRemaining, 75);
  assert.equal(view.budgetStatus.label, "Near token budget limit");
});

test("formatUSD returns a fallback label when cost data is missing", () => {
  assert.equal(formatUSD(19.25), "$19.25");
  assert.equal(formatUSD(null), "Pending");
});
