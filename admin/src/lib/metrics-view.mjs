export function getMetricsViewModel(metrics) {
  const dailyActiveUsers = Array.isArray(metrics?.daily_active_users) ? metrics.daily_active_users : [];
  const retention = Array.isArray(metrics?.retention) ? metrics.retention : [];
  const aiUsage = metrics?.ai_usage ?? null;
  const abComparison = metrics?.ab_comparison ?? null;
  const primaryABDelta =
    abComparison?.delta_retention_rate ??
    abComparison?.delta_challenge_participation_rate ??
    abComparison?.delta_leaderboard_engagement_rate ??
    abComparison?.delta_nudge_response_rate ??
    null;

  return {
    latestDAU: dailyActiveUsers.at(-1)?.users ?? 0,
    latestRetention: retention.at(-1) ?? null,
    aiUsage,
    abComparison,
    totalTokens: (aiUsage?.total_input_tokens ?? 0) + (aiUsage?.total_output_tokens ?? 0),
    dauPeak: Math.max(...dailyActiveUsers.map((point) => point.users), 1),
    hasDailyActivity: dailyActiveUsers.length > 0,
    hasRetention: retention.length > 0,
    hasABComparison: Boolean(abComparison?.variant_a && abComparison?.variant_b),
    primaryABDelta,
  };
}
