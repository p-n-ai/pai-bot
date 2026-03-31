import { normalizeAIUsage } from "./ai-usage.mjs";

function isRecord(value) {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function readNumber(value) {
  return Number.isFinite(value) ? value : null;
}

function normalizeABVariant(input, fallbackLabel) {
  const value = isRecord(input) ? input : {};

  return {
    label: typeof value.label === "string" && value.label ? value.label : fallbackLabel,
    users: Number.isFinite(value.users) ? value.users : 0,
    retention_rate: Number.isFinite(value.retention_rate) ? value.retention_rate : 0,
    challenge_participation_rate: Number.isFinite(value.challenge_participation_rate)
      ? value.challenge_participation_rate
      : 0,
    leaderboard_engagement_rate: Number.isFinite(value.leaderboard_engagement_rate)
      ? value.leaderboard_engagement_rate
      : 0,
    nudge_response_rate: Number.isFinite(value.nudge_response_rate) ? value.nudge_response_rate : 0,
  };
}

export function normalizeMetrics(input) {
  const value = input && typeof input === "object" ? input : {};
  const abComparison = isRecord(value.ab_comparison) ? value.ab_comparison : null;

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
    ab_comparison: abComparison
      ? {
          experiment_key: typeof abComparison.experiment_key === "string" ? abComparison.experiment_key : "",
          window_days: Number.isFinite(abComparison.window_days) ? abComparison.window_days : null,
          metric_name: typeof abComparison.metric_name === "string" ? abComparison.metric_name : "",
          variant_a: normalizeABVariant(abComparison.variant_a, "Variant A"),
          variant_b: normalizeABVariant(abComparison.variant_b, "Variant B"),
          winner: typeof abComparison.winner === "string" ? abComparison.winner : "",
          delta_retention_rate: readNumber(abComparison.delta_retention_rate),
          delta_challenge_participation_rate: readNumber(abComparison.delta_challenge_participation_rate),
          delta_leaderboard_engagement_rate: readNumber(abComparison.delta_leaderboard_engagement_rate),
          delta_nudge_response_rate: readNumber(abComparison.delta_nudge_response_rate),
        }
      : null,
  };
}
