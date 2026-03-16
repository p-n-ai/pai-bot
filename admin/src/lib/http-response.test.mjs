import test from "node:test";
import assert from "node:assert/strict";

import { readJSONResponse } from "./http-response.mjs";

test("readJSONResponse returns parsed JSON when body is present", async () => {
  const response = new Response(JSON.stringify({ status: "ok" }), {
    status: 200,
    headers: { "Content-Type": "application/json" },
  });

  await assert.doesNotReject(async () => {
    const payload = await readJSONResponse(response);
    assert.deepEqual(payload, { status: "ok" });
  });
});

test("readJSONResponse returns undefined for 204 responses", async () => {
  const response = new Response(null, { status: 204 });

  const payload = await readJSONResponse(response);

  assert.equal(payload, undefined);
});

test("readJSONResponse returns undefined for empty successful bodies", async () => {
  const response = new Response("", { status: 200 });

  const payload = await readJSONResponse(response);

  assert.equal(payload, undefined);
});
