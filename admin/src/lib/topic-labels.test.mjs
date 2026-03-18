import assert from "node:assert/strict";
import test from "node:test";
import { formatTopicLabel } from "./topic-labels.mjs";

test("formatTopicLabel converts dash-delimited ids into readable labels", () => {
  assert.equal(formatTopicLabel("linear-equations"), "Linear Equations");
  assert.equal(formatTopicLabel("form-1-algebra"), "Form 1 Algebra");
});

test("formatTopicLabel ignores empty segments", () => {
  assert.equal(formatTopicLabel("linear--equations-"), "Linear Equations");
});
