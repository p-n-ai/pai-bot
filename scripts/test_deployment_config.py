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
        self.assertEqual(
            workflow.count(
                "${{ secrets.PAI_AUTH_SECRET || "
                "secrets.LEARN_AUTH_JWT_SECRET }}"
            ),
            2,
        )
        self.assertIn('umask 077', workflow)
        self.assertIn('env_tmp="$(mktemp .env.tmp.XXXXXX)"', workflow)
        self.assertIn('> "$env_tmp"', workflow)
        self.assertIn('mv "$rollback_tmp" .env.rollback', workflow)
        self.assertIn('mv "$env_tmp" .env', workflow)
        self.assertNotIn("> .env", workflow)
        self.assertIn('"LEARN_AI_DEFAULT_PROVIDER=codex"', workflow)
        self.assertIn('"LEARN_AI_OPENAI_API_KEY=${OPENAI_KEY}"', workflow)

    def test_github_masks_ecr_token_before_same_job_handoff(self) -> None:
        workflow = source(".github/workflows/deploy.yml")
        deploy = workflow.split("\n  deploy:\n", maxsplit=1)[1]

        self.assertNotIn("ecr-token:", workflow)
        self.assertNotIn("outputs.ecr-token", workflow)
        self.assertIn("      id-token: write", deploy)
        self.assertEqual(workflow.count("Configure AWS credentials"), 1)
        mask_at = deploy.index('echo "::add-mask::$ecr_token"')
        output_at = deploy.index(
            'echo "ECR_TOKEN=$ecr_token" >> "$GITHUB_ENV"'
        )
        handoff_at = deploy.index(
            "envs: DEPLOY_DIR,ECR_TOKEN,GHCR_TOKEN"
        )
        self.assertLess(mask_at, output_at)
        self.assertLess(output_at, handoff_at)

    def test_compose_blocks_app_on_shared_secret_check(self) -> None:
        compose = source("docker-compose.prod.yml")
        self.assertIn("config-check:", compose)
        self.assertIn("/pai-validate-production-secrets", compose)
        self.assertIn("condition: service_completed_successfully", compose)

    def test_remote_deploy_validates_before_migrations(self) -> None:
        deploy = source("scripts/deploy-remote.sh")
        validate_at = deploy.index("run --rm config-check")
        backup_at = deploy.index('echo "--- Creating pre-migration database backup ---"')
        migrate_at = deploy.index('echo "--- Running migrations ---"')
        self.assertLess(validate_at, migrate_at)
        self.assertGreater(backup_at, migrate_at)
        self.assertIn('pg_dump "$DB_URL" --format=custom', deploy)
        self.assertIn("pg_restore --list", deploy)
        self.assertLess(backup_at, deploy.index("goose@v3.26.0"))

    def test_remote_rollback_restores_matching_environment(self) -> None:
        deploy = source("scripts/deploy-remote.sh")
        rollback_at = deploy.index("rollback_release()")
        restore_at = deploy.index("Restored previous environment")
        restart_at = deploy.index("up -d --force-recreate app admin")
        self.assertLess(rollback_at, restore_at)
        self.assertLess(restore_at, restart_at)
        self.assertIn("--format='{{.Image}}'", deploy)
        self.assertNotIn("--format='{{.Config.Image}}'", deploy)
        health_failure = deploy.split(
            'if [ "$APP_HEALTH" != "healthy" ]; then', maxsplit=1
        )[1]
        self.assertIn('fail_release "app did not become healthy"', health_failure)
        fail_release = deploy.split("fail_release() {", maxsplit=1)[1].split(
            "\n}", maxsplit=1
        )[0]
        self.assertIn("rollback_release", fail_release)

    def test_remote_smoke_checks_are_read_only_and_blocking(self) -> None:
        deploy = source("scripts/deploy-remote.sh")
        self.assertNotIn('smoke "/create_group"', deploy)
        self.assertIn('smoke "/help"', deploy)
        smoke_failure = deploy.split(
            'if [ "$SMOKE_FAIL" -gt 0 ]; then', maxsplit=1
        )[1]
        self.assertIn("fail_release", smoke_failure)

    def test_remote_deploy_checks_ai_response_before_success(self) -> None:
        deploy = source("scripts/deploy-remote.sh")
        app_health_at = deploy.index('echo "--- Health check: app endpoint ---"')
        ai_health_at = deploy.index(
            'echo "--- Health check: application and AI provider status ---"'
        )
        success_at = deploy.index(
            'echo "--- Recording successfully deployed PostgreSQL aliases ---"'
        )
        self.assertLess(app_health_at, ai_health_at)
        self.assertLess(ai_health_at, success_at)
        self.assertIn("http://localhost:8080/health/status", deploy)
        self.assertIn('"id":"ai_provider","status":"operational"', deploy)
        self.assertNotIn("PAI_AI_HEALTH_TOKEN", deploy)
        self.assertIn("--max-time 15", deploy)

    def test_remote_deploy_rolls_back_when_ai_response_is_unhealthy(self) -> None:
        deploy = source("scripts/deploy-remote.sh")
        ai_failure = deploy.split(
            'if [ "$STATUS_RESPONSE" != "$STATUS_EXPECTED" ]; then',
            maxsplit=1,
        )[1]
        self.assertIn('fail_release "AI response health check failed"', ai_failure)


if __name__ == "__main__":
    unittest.main()
