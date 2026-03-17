import test from "node:test";
import assert from "node:assert/strict";

import { buildParentContextLine, formatParentTopicLabel, getParentMasteryTone } from "./parent-summary.mjs";

test("formatParentTopicLabel converts topic ids into readable labels", () => {
  assert.equal(formatParentTopicLabel("linear-equations"), "Linear Equations");
});

test("getParentMasteryTone returns the correct progress color band", () => {
  assert.equal(getParentMasteryTone(0.82), "bg-emerald-500");
  assert.equal(getParentMasteryTone(0.61), "bg-lime-500");
  assert.equal(getParentMasteryTone(0.45), "bg-amber-400");
  assert.equal(getParentMasteryTone(0.2), "bg-rose-400");
});

test("buildParentContextLine prefers parent email for contact context", () => {
  assert.equal(
    buildParentContextLine({
      child: { form: "Form 1", channel: "telegram" },
      parent: { email: "parent@example.com", name: "Farah Parent" },
    }),
    "Form 1 | telegram | Parent contact parent@example.com",
  );
});
