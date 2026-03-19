export function getMetricsViewModel(metrics) {
  const dailyActiveUsers = Array.isArray(metrics?.daily_active_users) ? metrics.daily_active_users : [];
  const retention = Array.isArray(metrics?.retention) ? metrics.retention : [];
  const aiUsage = metrics?.ai_usage ?? null;

  return {
    latestDAU: dailyActiveUsers.at(-1)?.users ?? 0,
    latestRetention: retention.at(-1) ?? null,
    aiUsage,
    totalTokens: (aiUsage?.total_input_tokens ?? 0) + (aiUsage?.total_output_tokens ?? 0),
    dauPeak: Math.max(...dailyActiveUsers.map((point) => point.users), 1),
    hasDailyActivity: dailyActiveUsers.length > 0,
    hasRetention: retention.length > 0,
  };
}
