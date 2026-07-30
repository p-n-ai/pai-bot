#!/usr/bin/env python3
"""Repository contract tests for production-secret deployment gates."""

from __future__ import annotations

import json
import os
import pathlib
import shutil
import subprocess
import tempfile
import unittest


ROOT = pathlib.Path(__file__).resolve().parents[1]


def source(path: str) -> str:
    return (ROOT / path).read_text()


class ProductionSecretDeploymentTests(unittest.TestCase):
    def test_compose_resolves_secret_gate_and_runtime_environment(self) -> None:
        docker = shutil.which("docker")
        if docker is None:
            self.skipTest("docker is not installed")
        if subprocess.run(
            [docker, "compose", "version"],
            capture_output=True,
            check=False,
            text=True,
        ).returncode != 0:
            self.skipTest("docker compose is not installed")

        environment = os.environ.copy()
        environment.update(
            {
                "POSTGRES_PASSWORD": "test-postgres-password",
                "PAI_AUTH_SECRET": "test-auth-secret-with-enough-variety",
                "PAI_CONFIG_ENCRYPTION_KEY": "test-config-encryption-key-12345",
                "PAI_CONFIG_PREVIOUS_ENCRYPTION_KEYS": "[]",
                "PAI_AUTH_BOOTSTRAP_ADMIN_EMAIL": "admin@example.com",
                "PAI_AUTH_BOOTSTRAP_ADMIN_PASSWORD": "test-bootstrap-password",
            }
        )
        with tempfile.TemporaryDirectory() as project_directory:
            pathlib.Path(project_directory, ".env").write_text("")
            result = subprocess.run(
                [
                    docker,
                    "compose",
                    "--project-directory",
                    project_directory,
                    "-f",
                    str(ROOT / "docker-compose.yml"),
                    "-f",
                    str(ROOT / "docker-compose.prod.yml"),
                    "config",
                    "--format",
                    "json",
                ],
                capture_output=True,
                check=False,
                env=environment,
                text=True,
            )
        self.assertEqual(result.returncode, 0, result.stderr)
        compose = json.loads(result.stdout)
        services = compose["services"]
        self.assertEqual(
            services["config-check"]["entrypoint"],
            ["/pai-validate-production-secrets"],
        )
        self.assertEqual(
            services["app"]["depends_on"]["config-check"]["condition"],
            "service_completed_successfully",
        )
        self.assertEqual(
            services["app"]["environment"]["PAI_CONFIG_ENCRYPTION_KEY"],
            environment["PAI_CONFIG_ENCRYPTION_KEY"],
        )
        self.assertEqual(
            services["app"]["environment"]["PAI_CONFIG_PREVIOUS_ENCRYPTION_KEYS"],
            "[]",
        )

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
