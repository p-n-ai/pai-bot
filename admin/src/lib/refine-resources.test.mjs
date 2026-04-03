import test from "node:test";
import assert from "node:assert/strict";

import { getAdminResources } from "./refine-resources.mjs";

test("getAdminResources exposes the admin workspace routes to Refine", () => {
  const resources = getAdminResources();

  assert.deepEqual(
    resources.map((item) => item.name),
    ["dashboard", "ai-usage", "retrieval-lab", "students", "parents"],
  );
  assert.equal(resources.find((item) => item.name === "students")?.show, "/students/:id");
  assert.equal(resources.find((item) => item.name === "parents")?.show, "/parents/:id");
});
