#!/usr/bin/env python3
"""Behavior tests for the external uptime probe."""

from __future__ import annotations

import contextlib
import http.server
import threading
import time
import unittest

import uptime_probe


class ResponseHandler(http.server.BaseHTTPRequestHandler):
    status = 200
    content_type = "application/json"
    body = b'{"status":"ok"}'
    delay = 0.0
    redirect_to = ""
    expected_token = ""

    def do_GET(self) -> None:
        if self.path not in {"/health", "/health/ai"}:
            self.send_error(404)
            return
        if self.path == "/health/ai" and self.headers.get("Authorization") != f"Bearer {self.expected_token}":
            self.send_error(404)
            return
        time.sleep(self.delay)
        if self.redirect_to:
            self.send_response(302)
            self.send_header("Location", self.redirect_to)
            self.end_headers()
            return
        self.send_response(self.status)
        self.send_header("Content-Type", self.content_type)
        self.end_headers()
        with contextlib.suppress(BrokenPipeError):
            self.wfile.write(self.body)

    def log_message(self, format: str, *args: object) -> None:
        pass


@contextlib.contextmanager
def serve(
    *,
    status: int = 200,
    content_type: str = "application/json",
    body: bytes = b'{"status":"ok"}',
    delay: float = 0.0,
    redirect_to: str = "",
    expected_token: str = "",
):
    handler = type(
        "ConfiguredResponseHandler",
        (ResponseHandler,),
        {
            "status": status,
            "content_type": content_type,
            "body": body,
            "delay": delay,
            "redirect_to": redirect_to,
            "expected_token": expected_token,
        },
    )
    server = http.server.ThreadingHTTPServer(("127.0.0.1", 0), handler)
    thread = threading.Thread(target=server.serve_forever)
    thread.start()
    try:
        yield f"http://127.0.0.1:{server.server_port}"
    finally:
        server.shutdown()
        server.server_close()
        thread.join()


class ProbeTests(unittest.TestCase):
    def test_accepts_exact_health_contract(self) -> None:
        with serve() as target:
            self.assertEqual(
                uptime_probe.probe(target, timeout=1, allow_http=True),
                f"{target}/health",
            )

    def test_rejects_non_200_without_exposing_body(self) -> None:
        secret = b"do-not-log-this"
        with serve(status=503, body=secret) as target:
            with self.assertRaisesRegex(
                uptime_probe.ProbeError, "returned HTTP 503"
            ) as raised:
                uptime_probe.probe(target, timeout=1, allow_http=True)
        self.assertNotIn(secret.decode(), str(raised.exception))

    def test_rejects_wrong_json_contract_without_exposing_body(self) -> None:
        secret = b'{"status":"degraded","detail":"do-not-log-this"}'
        with serve(body=secret) as target:
            with self.assertRaisesRegex(
                uptime_probe.ProbeError, "wrong status contract"
            ) as raised:
                uptime_probe.probe(target, timeout=1, allow_http=True)
        self.assertNotIn(secret.decode(), str(raised.exception))

    def test_rejects_wrong_content_type(self) -> None:
        with serve(content_type="text/plain") as target:
            with self.assertRaisesRegex(
                uptime_probe.ProbeError, "did not return application/json"
            ):
                uptime_probe.probe(target, timeout=1, allow_http=True)

    def test_rejects_oversized_response(self) -> None:
        with serve(body=b"x" * (uptime_probe.MAX_RESPONSE_BYTES + 1)) as target:
            with self.assertRaisesRegex(
                uptime_probe.ProbeError, "response exceeded 4096 bytes"
            ):
                uptime_probe.probe(target, timeout=1, allow_http=True)

    def test_rejects_timeout(self) -> None:
        with serve(delay=0.2) as target:
            with self.assertRaisesRegex(uptime_probe.ProbeError, "timed out"):
                uptime_probe.probe(target, timeout=0.05, allow_http=True)

    def test_rejects_redirect(self) -> None:
        with serve(redirect_to="https://example.com/health") as target:
            with self.assertRaisesRegex(
                uptime_probe.ProbeError, "returned HTTP 302"
            ):
                uptime_probe.probe(target, timeout=1, allow_http=True)

    def test_deliberate_failure_requires_healthy_owned_origin(self) -> None:
        with serve() as target:
            with self.assertRaisesRegex(
                uptime_probe.ProbeError,
                "deliberate verification failure after healthy response",
            ):
                uptime_probe.probe(
                    target,
                    timeout=1,
                    allow_http=True,
                    deliberate_failure=True,
                )

    def test_ai_health_uses_authenticated_ai_path(self) -> None:
        with serve(expected_token="monitor-secret") as target:
            self.assertEqual(
                uptime_probe.probe(
                    target,
                    timeout=1,
                    allow_http=True,
                    check="ai",
                    token="monitor-secret",
                ),
                f"{target}/health/ai",
            )

    def test_ai_health_requires_token(self) -> None:
        with serve(expected_token="monitor-secret") as target:
            with self.assertRaisesRegex(
                uptime_probe.ProbeError, "AI health token is required"
            ):
                uptime_probe.probe(
                    target,
                    timeout=1,
                    allow_http=True,
                    check="ai",
                )

    def test_requires_https_outside_local_tests(self) -> None:
        with self.assertRaisesRegex(uptime_probe.ProbeError, "must use HTTPS"):
            uptime_probe.health_url("http://example.com")

    def test_rejects_credentials_and_unexpected_paths(self) -> None:
        invalid_targets = (
            "https://user:password@example.com",
            "https://example.com/admin",
            "https://example.com?token=secret",
        )
        for target in invalid_targets:
            with self.subTest(target=target):
                with self.assertRaises(uptime_probe.ProbeError):
                    uptime_probe.health_url(target)


if __name__ == "__main__":
    unittest.main()
