import test from "node:test";
import assert from "node:assert/strict";

import { formatCompactNumber, getTopProvider, normalizeAIUsage } from "./ai-usage.mjs";

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
