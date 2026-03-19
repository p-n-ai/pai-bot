import test from "node:test";
import assert from "node:assert/strict";

import { getMetricsViewModel } from "./metrics-view.mjs";

test("getMetricsViewModel derives the latest metrics snapshot", () => {
  const model = getMetricsViewModel({
    daily_active_users: [{ date: "2026-03-18", users: 12 }, { date: "2026-03-19", users: 20 }],
    retention: [{ cohort_date: "2026-03-10", day_7_rate: 0.45 }],
    ai_usage: { total_input_tokens: 100, total_output_tokens: 40 },
  });

  assert.equal(model.latestDAU, 20);
  assert.equal(model.latestRetention?.day_7_rate, 0.45);
  assert.equal(model.totalTokens, 140);
  assert.equal(model.dauPeak, 20);
  assert.equal(model.hasDailyActivity, true);
  assert.equal(model.hasRetention, true);
});

test("getMetricsViewModel falls back safely when metrics are missing", () => {
  const model = getMetricsViewModel(null);

  assert.equal(model.latestDAU, 0);
  assert.equal(model.latestRetention, null);
  assert.equal(model.totalTokens, 0);
  assert.equal(model.dauPeak, 1);
  assert.equal(model.hasDailyActivity, false);
  assert.equal(model.hasRetention, false);
});
