export type AIUsageBudgetStatus = Readonly<{
  label: string;
  tone: string;
}>;

export type AIUsageDailyPoint = Readonly<{
  date: string;
  messages: number;
  tokens: number;
  cost_usd: number | null;
}>;

export type AIUsageProvider = Readonly<{
  provider: string;
  model: string;
  messages: number;
  input_tokens: number;
  output_tokens: number;
  total_tokens: number;
}>;

export type AIUsageView = Readonly<{
  totalTokens: number;
  total_messages: number;
  topProvider: AIUsageProvider | null;
  budgetStatus: AIUsageBudgetStatus;
  budget_period_start: string;
  budget_period_end: string;
  per_student_average_tokens: number | null;
  per_student_average_cost_usd: number | null;
  budgetTokenLimit: number | null;
  budgetTokenRemaining: number | null;
  daily_usage: readonly AIUsageDailyPoint[];
  hasDailyTrend: boolean;
  dailyTrendPeak: number;
  providers: readonly AIUsageProvider[];
  monthlyCost: number | null;
  budgetLimit: number | null;
}>;
