#!/usr/bin/env python3
"""Probe pai-bot's public liveness contract from an external runner."""

from __future__ import annotations

import argparse
import json
import os
import sys
import urllib.error
import urllib.parse
import urllib.request

MAX_RESPONSE_BYTES = 4096


class ProbeError(Exception):
    """An expected uptime contract failure safe to show in monitoring logs."""


class NoRedirectHandler(urllib.request.HTTPRedirectHandler):
    """Refuse redirects so another endpoint cannot satisfy the health contract."""

    def redirect_request(self, req, fp, code, msg, headers, newurl):
        return None


def health_url(target: str, *, check: str = "app", allow_http: bool = False) -> str:
    """Return the canonical health URL for a public origin or health URL."""
    parsed = urllib.parse.urlsplit(target)
    allowed_schemes = {"https"}
    if allow_http:
        allowed_schemes.add("http")
    if parsed.scheme not in allowed_schemes:
        raise ProbeError("target must use HTTPS")
    if not parsed.hostname or parsed.username or parsed.password:
        raise ProbeError("target must be a public origin without credentials")
    if parsed.query or parsed.fragment:
        raise ProbeError("target must not contain a query or fragment")
    if parsed.path not in {"", "/", "/api", "/api/", "/health", "/health/ai"}:
        raise ProbeError("target must be an origin, API base URL, or health URL")
    path = "/health/ai" if check == "ai" else "/health"
    return urllib.parse.urlunsplit(
        (parsed.scheme, parsed.netloc, path, "", "")
    )


def probe(
    target: str,
    *,
    timeout: float,
    allow_http: bool = False,
    deliberate_failure: bool = False,
    check: str = "app",
    token: str = "",
) -> str:
    """Probe the target and return its normalized URL when healthy."""
    url = health_url(target, check=check, allow_http=allow_http)
    headers = {
        "Accept": "application/json",
        "User-Agent": "pai-uptime-monitor/1",
    }
    if check == "ai":
        if not token:
            raise ProbeError("AI health token is required")
        headers["Authorization"] = f"Bearer {token}"
    request = urllib.request.Request(
        url,
        headers=headers,
        method="GET",
    )

    try:
        opener = urllib.request.build_opener(NoRedirectHandler())
        with opener.open(request, timeout=timeout) as response:
            status = response.status
            content_type = response.headers.get_content_type()
            body = response.read(MAX_RESPONSE_BYTES + 1)
    except urllib.error.HTTPError as error:
        status = error.code
        error.close()
        raise ProbeError(f"health endpoint returned HTTP {status}") from None
    except TimeoutError:
        raise ProbeError("health endpoint timed out") from None
    except urllib.error.URLError as error:
        if isinstance(error.reason, TimeoutError):
            raise ProbeError("health endpoint timed out") from None
        raise ProbeError("health endpoint unreachable") from None

    if status != 200:
        raise ProbeError(f"health endpoint returned HTTP {status}")
    if content_type != "application/json":
        raise ProbeError("health endpoint did not return application/json")
    if len(body) > MAX_RESPONSE_BYTES:
        raise ProbeError("health response exceeded 4096 bytes")

    try:
        payload = json.loads(body)
    except (UnicodeDecodeError, json.JSONDecodeError):
        raise ProbeError("health endpoint returned invalid JSON") from None
    if payload != {"status": "ok"}:
        raise ProbeError("health endpoint returned the wrong status contract")
    if deliberate_failure:
        raise ProbeError("deliberate verification failure after healthy response")
    return url


def parse_args() -> argparse.Namespace:
    """Parse command-line input at the process boundary."""
    parser = argparse.ArgumentParser()
    parser.add_argument("target", help="Public HTTPS origin, API base, or /health URL")
    parser.add_argument("--check", choices=("app", "ai"), default="app")
    parser.add_argument("--timeout", type=float, default=10.0)
    parser.add_argument(
        "--allow-http",
        action="store_true",
        help="Permit HTTP for local testing only",
    )
    parser.add_argument(
        "--deliberate-failure",
        action="store_true",
        help="Fail after a healthy owned-origin response for rollout verification",
    )
    args = parser.parse_args()
    if args.timeout <= 0:
        parser.error("--timeout must be greater than zero")
    return args


def main() -> int:
    """Run one probe and emit only safe, bounded diagnostics."""
    args = parse_args()
    try:
        url = probe(
            args.target,
            timeout=args.timeout,
            allow_http=args.allow_http,
            deliberate_failure=args.deliberate_failure,
            check=args.check,
            token=os.environ.get("PAI_AI_HEALTH_TOKEN", ""),
        )
    except ProbeError as error:
        print(f"uptime probe failed: {error}", file=sys.stderr)
        return 1
    print(f"uptime probe passed: {url}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
