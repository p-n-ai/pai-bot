import test from "node:test";
import assert from "node:assert/strict";

import { getParentViewModel } from "./parent-view.mjs";

test("getParentViewModel keeps mastery rows and encouragement copy from the summary", () => {
  const model = getParentViewModel({
    mastery: [{ topic_id: "algebra", mastery_score: 0.7 }],
    encouragement: { headline: "Keep going", text: "Celebrate this week's work." },
  });

  assert.equal(model.hasMastery, true);
  assert.equal(model.masteryRows.length, 1);
  assert.equal(model.encouragementHeadline, "Keep going");
  assert.equal(model.encouragementText, "Celebrate this week's work.");
});

test("getParentViewModel falls back when no summary is available", () => {
  const model = getParentViewModel(null);

  assert.equal(model.hasMastery, false);
  assert.equal(model.masteryRows.length, 0);
  assert.match(model.encouragementHeadline, /suggested encouragement/i);
});
