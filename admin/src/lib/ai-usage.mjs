function isRecord(value) {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function readNumber(value) {
  return typeof value === "number" && Number.isFinite(value) ? value : null;
}

export function normalizeAIUsage(payload) {
  const source = isRecord(payload) ? payload : {};
  const providers = Array.isArray(source.providers)
    ? source.providers
        .filter(isRecord)
        .map((provider) => ({
          provider: typeof provider.provider === "string" ? provider.provider : "unknown",
          model: typeof provider.model === "string" ? provider.model : "",
          messages: typeof provider.messages === "number" ? provider.messages : 0,
          input_tokens: typeof provider.input_tokens === "number" ? provider.input_tokens : 0,
          output_tokens: typeof provider.output_tokens === "number" ? provider.output_tokens : 0,
          total_tokens:
            typeof provider.total_tokens === "number"
              ? provider.total_tokens
              : (typeof provider.input_tokens === "number" ? provider.input_tokens : 0) +
                (typeof provider.output_tokens === "number" ? provider.output_tokens : 0),
        }))
    : [];
  const daily_usage = Array.isArray(source.daily_usage)
    ? source.daily_usage
        .filter(isRecord)
        .map((point) => ({
          date: typeof point.date === "string" ? point.date : "",
          messages: typeof point.messages === "number" ? point.messages : 0,
          tokens: typeof point.tokens === "number" ? point.tokens : 0,
          cost_usd: readNumber(point.cost_usd),
        }))
        .filter((point) => point.date)
    : [];
  const provider_costs = Array.isArray(source.provider_costs)
    ? source.provider_costs
        .filter(isRecord)
        .map((item) => ({
          provider: typeof item.provider === "string" ? item.provider : "unknown",
          cost_usd: readNumber(item.cost_usd),
        }))
    : [];

  return {
    total_messages: typeof source.total_messages === "number" ? source.total_messages : 0,
    total_input_tokens: typeof source.total_input_tokens === "number" ? source.total_input_tokens : 0,
    total_output_tokens: typeof source.total_output_tokens === "number" ? source.total_output_tokens : 0,
    providers,
    monthly_cost_usd: readNumber(source.monthly_cost_usd),
    budget_limit_usd: readNumber(source.budget_limit_usd),
    per_student_average_tokens: readNumber(source.per_student_average_tokens),
    per_student_average_cost_usd: readNumber(source.per_student_average_cost_usd),
    budget_limit_tokens: readNumber(source.budget_limit_tokens),
    budget_used_tokens: readNumber(source.budget_used_tokens),
    budget_remaining_tokens: readNumber(source.budget_remaining_tokens),
    budget_period_start: typeof source.budget_period_start === "string" ? source.budget_period_start : "",
    budget_period_end: typeof source.budget_period_end === "string" ? source.budget_period_end : "",
    daily_usage,
    provider_costs,
  };
}

export function getTopProvider(usage) {
  const providers = Array.isArray(usage?.providers) ? usage.providers : [];
  if (providers.length === 0) {
    return null;
  }

  return providers.reduce((best, current) => {
    if (!best || current.total_tokens > best.total_tokens) {
      return current;
    }
    return best;
  }, null);
}

export function formatCompactNumber(value) {
  return new Intl.NumberFormat("en", { notation: "compact", maximumFractionDigits: 1 }).format(value || 0);
}

export function formatUSD(value) {
  if (typeof value !== "number" || !Number.isFinite(value)) {
    return "Pending";
  }

  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: "USD",
    minimumFractionDigits: value < 100 ? 2 : 0,
    maximumFractionDigits: 2,
  }).format(value);
}

export function getAIUsageBudgetViewModel(usage) {
  const normalized = normalizeAIUsage(usage);
  const totalTokens = normalized.total_input_tokens + normalized.total_output_tokens;
  const monthlyCost = normalized.monthly_cost_usd;
  const budgetLimit = normalized.budget_limit_usd;
  const budgetTokenLimit = normalized.budget_limit_tokens;
  const budgetTokenUsed = normalized.budget_used_tokens;
  const budgetTokenRemaining = normalized.budget_remaining_tokens;
  const remainingBudget =
    monthlyCost !== null && budgetLimit !== null ? Math.max(budgetLimit - monthlyCost, 0) : null;
  const usageRatio =
    monthlyCost !== null && budgetLimit !== null && budgetLimit > 0 ? monthlyCost / budgetLimit : null;
  const tokenUsageRatio =
    budgetTokenLimit !== null && budgetTokenUsed !== null && budgetTokenLimit > 0 ? budgetTokenUsed / budgetTokenLimit : null;
  const budgetStatus =
    tokenUsageRatio !== null
      ? tokenUsageRatio >= 1
        ? {
            label: "Token budget exceeded",
            tone: "text-rose-600 dark:text-rose-300",
          }
        : tokenUsageRatio >= 0.8
          ? {
              label: "Near token budget limit",
              tone: "text-amber-600 dark:text-amber-300",
            }
          : {
              label: "Within token budget",
              tone: "text-emerald-600 dark:text-emerald-300",
            }
      : usageRatio === null
      ? {
          label: "No token budget configured",
          tone: "text-slate-500 dark:text-slate-400",
        }
      : usageRatio >= 1
        ? {
            label: "Budget exceeded",
            tone: "text-rose-600 dark:text-rose-300",
          }
        : usageRatio >= 0.8
          ? {
              label: "Near budget limit",
              tone: "text-amber-600 dark:text-amber-300",
            }
          : {
              label: "Within budget",
              tone: "text-emerald-600 dark:text-emerald-300",
            };
  const dailyTrendPeak = normalized.daily_usage.reduce((peak, point) => Math.max(peak, point.tokens), 0);
  const providerCostTotal = normalized.provider_costs.reduce((sum, item) => sum + (item.cost_usd ?? 0), 0);

  return {
    ...normalized,
    totalTokens,
    topProvider: getTopProvider(normalized),
    monthlyCost,
    budgetLimit,
    budgetTokenLimit,
    budgetTokenUsed,
    budgetTokenRemaining,
    remainingBudget,
    usageRatio,
    tokenUsageRatio,
    budgetStatus,
    hasDailyTrend: normalized.daily_usage.length > 0,
    dailyTrendPeak,
    hasProviderCosts: normalized.provider_costs.some((item) => (item.cost_usd ?? 0) > 0),
    providerCostTotal,
    hasPerStudentAverages:
      normalized.per_student_average_tokens !== null || normalized.per_student_average_cost_usd !== null,
  };
}
