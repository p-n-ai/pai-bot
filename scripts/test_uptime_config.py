#!/usr/bin/env python3
"""Repository contract tests for external uptime routing and workflows."""

from __future__ import annotations

import pathlib
import unittest


ROOT = pathlib.Path(__file__).resolve().parents[1]


def source(path: str) -> str:
    return (ROOT / path).read_text()


class UptimeRoutingTests(unittest.TestCase):
    def test_caddy_routes_app_and_provider_status_to_backend(self) -> None:
        for path in (
            "deploy/caddy/Caddyfile",
            "deploy/caddy/Caddyfile.http",
            "deploy/caddy/Caddyfile.https",
        ):
            with self.subTest(path=path):
                config = source(path)
                self.assertIn("handle /health/api {", config)
                self.assertIn("handle /health/status {", config)
                self.assertNotIn("handle /health/ai {", config)
                self.assertNotIn("handle /health {", config)
                self.assertGreaterEqual(config.count("reverse_proxy app:8080"), 3)

    def test_helm_routes_public_health_paths(self) -> None:
        ingress = source("deploy/helm/pai/templates/ingress.yaml")
        self.assertIn("- path: /health/api\n            pathType: Exact", ingress)
        self.assertIn("- path: /health/status\n            pathType: Exact", ingress)
        self.assertNotIn("- path: /health/ai", ingress)

    def test_nginx_routes_public_health_paths(self) -> None:
        self.assertNotIn("location = /health/ai", source("deploy/nginx/pai-bot.conf"))
        self.assertIn(
            "location = /health/status",
            source("deploy/nginx/pai-bot.conf"),
        )
        self.assertIn(
            "health/(api|status)",
            source("deploy/once/nginx.conf"),
        )

    def test_deploy_has_no_separate_ai_health_configuration(self) -> None:
        workflow = source(".github/workflows/deploy.yml")
        self.assertIn('"PAI_FEATURES=${FEATURES}"', workflow)
        self.assertIn("FEATURES: ${{ vars.PAI_FEATURES }}", workflow)
        self.assertNotIn("PAI_AI_HEALTH_TOKEN", workflow)
        self.assertNotIn("AI_HEALTH_TOKEN", workflow)

    def test_uptime_checks_application_and_provider_together(self) -> None:
        workflow = source(".github/workflows/uptime.yml")
        self.assertIn("vars.PAI_UPTIME_ENABLED == 'true'", workflow)
        self.assertIn("cancel-in-progress: false", workflow)
        self.assertIn("--deliberate-failure", workflow)
        self.assertIn("--check status", workflow)
        self.assertNotIn("target_url:", workflow)
        self.assertNotIn("PAI_AI_UPTIME_ENABLED", workflow)
        self.assertNotIn("PAI_AI_HEALTH_TOKEN", workflow)
        self.assertNotIn("ai-probe:", workflow)


if __name__ == "__main__":
    unittest.main()
