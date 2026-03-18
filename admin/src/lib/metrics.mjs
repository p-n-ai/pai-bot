import { normalizeAIUsage } from "./ai-usage.mjs";

export function normalizeMetrics(input) {
  const value = input && typeof input === "object" ? input : {};

  return {
    window_days: Number.isFinite(value.window_days) ? value.window_days : 14,
    daily_active_users: Array.isArray(value.daily_active_users) ? value.daily_active_users : [],
    retention: Array.isArray(value.retention) ? value.retention : [],
    nudge_rate: {
      nudges_sent: Number.isFinite(value.nudge_rate?.nudges_sent) ? value.nudge_rate.nudges_sent : 0,
      responses_within_24h: Number.isFinite(value.nudge_rate?.responses_within_24h) ? value.nudge_rate.responses_within_24h : 0,
      response_rate: Number.isFinite(value.nudge_rate?.response_rate) ? value.nudge_rate.response_rate : 0,
    },
    ai_usage: normalizeAIUsage(value.ai_usage ?? {}),
    ab_comparison: value.ab_comparison ?? null,
  };
}
