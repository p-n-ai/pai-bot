#!/usr/bin/env python3
"""Repository contract tests for production-secret deployment gates."""

from __future__ import annotations

import pathlib
import unittest


ROOT = pathlib.Path(__file__).resolve().parents[1]


def source(path: str) -> str:
    return (ROOT / path).read_text()


class ProductionSecretDeploymentTests(unittest.TestCase):
    def test_github_validates_before_copy_and_replaces_env_atomically(self) -> None:
        workflow = source(".github/workflows/deploy.yml")
        validate_at = workflow.index("go run ./cmd/validate-production-secrets")
        copy_at = workflow.index("- name: Copy files to server")
        self.assertLess(validate_at, copy_at)
        self.assertIn('umask 077', workflow)
        self.assertIn('env_tmp="$(mktemp .env.tmp.XXXXXX)"', workflow)
        self.assertIn('> "$env_tmp"', workflow)
        self.assertIn('mv "$env_tmp" .env', workflow)
        self.assertNotIn("> .env", workflow)

    def test_compose_blocks_app_on_shared_secret_check(self) -> None:
        compose = source("docker-compose.prod.yml")
        self.assertIn("config-check:", compose)
        self.assertIn("/pai-validate-production-secrets", compose)
        self.assertIn("condition: service_completed_successfully", compose)

    def test_remote_deploy_validates_before_migrations(self) -> None:
        deploy = source("scripts/deploy-remote.sh")
        validate_at = deploy.index("run --rm config-check")
        migrate_at = deploy.index('echo "--- Running migrations ---"')
        self.assertLess(validate_at, migrate_at)


if __name__ == "__main__":
    unittest.main()
