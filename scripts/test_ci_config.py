#!/usr/bin/env python3
"""Repository contracts for CI-to-deployment orchestration."""

from __future__ import annotations

import pathlib
import unittest


ROOT = pathlib.Path(__file__).resolve().parents[1]


def source(path: str) -> str:
    return (ROOT / path).read_text()


class CIWorkflowTests(unittest.TestCase):
    def test_deploy_is_reusable_and_does_not_repeat_ci_tests(self) -> None:
        workflow = source(".github/workflows/deploy.yml")

        self.assertIn("  workflow_call:", workflow)
        self.assertNotIn("  push:\n", workflow)
        self.assertNotIn("\n  test:\n", workflow)
        self.assertNotIn("pnpm test:e2e", workflow)
        self.assertNotIn("go test ./...", workflow)

    def test_main_deploy_waits_for_exact_sha_ci_jobs(self) -> None:
        workflow = source(".github/workflows/ci.yml")

        self.assertIn("  production-deploy:", workflow)
        self.assertIn("      always() &&", workflow)
        self.assertIn("    uses: ./.github/workflows/deploy.yml", workflow)
        self.assertIn(
            "admin_changed: ${{ needs.changes.outputs.admin_image == 'true' }}",
            workflow,
        )
        self.assertIn(
            "app_changed: ${{ needs.changes.outputs.app_image == 'true' }}",
            workflow,
        )
        for job in (
            "changes",
            "config",
            "admin-spa",
            "admin-spa-e2e",
            "go",
            "go-lint",
            "docker",
            "admin-docker",
            "postgres",
            "once",
        ):
            with self.subTest(job=job):
                self.assertIn(f"      - {job}\n", workflow)

    def test_routine_postgres_publish_is_change_gated(self) -> None:
        workflow = source(".github/workflows/deploy.yml")

        self.assertIn("inputs.postgres_changed", workflow)
        self.assertIn(
            "          POSTGRES_ACTION: ${{ steps.record-postgres.outputs.action }}",
            workflow,
        )
        self.assertNotIn("needs.changes.outputs.postgres", workflow)

        ci_workflow = source(".github/workflows/ci.yml")
        self.assertIn(
            "postgres_changed: ${{ needs.changes.outputs.postgres_image == 'true' }}",
            ci_workflow,
        )

    def test_postgres_deployment_uses_an_immutable_digest(self) -> None:
        deploy_workflow = source(".github/workflows/deploy.yml")
        self.assertIn(
            "inputs.postgres_changed ||",
            deploy_workflow,
        )
        self.assertIn(
            "inputs.rebuild_postgres",
            deploy_workflow,
        )
        self.assertIn(
            "DEPLOYMENT_TAG: ghcr.io/p-n-ai/pai-postgres:${{ github.sha }}",
            deploy_workflow,
        )
        self.assertIn(
            "steps.recorded-postgres.outputs.exists != 'true'",
            deploy_workflow,
        )
        self.assertIn(
            "      - name: Resolve immutable PostgreSQL image",
            deploy_workflow,
        )
        self.assertIn(
            "      - name: Record PostgreSQL image for deployment SHA",
            deploy_workflow,
        )
        self.assertIn(
            "POSTGRES_IMAGE: ${{ needs.push-images.outputs.postgres-image }}",
            deploy_workflow,
        )
        self.assertIn(
            "image=ghcr.io/p-n-ai/pai-postgres@$digest",
            deploy_workflow,
        )

        compose = source("docker-compose.prod.yml")
        self.assertIn("image: ${POSTGRES_IMAGE:-", compose)

    def test_repository_contracts_run_in_exact_sha_ci_gate(self) -> None:
        workflow = source(".github/workflows/ci.yml")

        self.assertIn("  config:\n", workflow)
        self.assertIn("      - 'scripts/*.py'", workflow)
        self.assertIn("python3 -m unittest discover", workflow)
        self.assertIn(
            "(needs.config.result == 'success' || needs.config.result == 'skipped')",
            workflow,
        )

    def test_private_postgres_image_pulls_are_authenticated(self) -> None:
        ci_workflow = source(".github/workflows/ci.yml")
        ci_login = ci_workflow.index("      - name: Log in to GHCR")
        ci_pull = ci_workflow.index("          docker compose pull postgres dragonfly")

        self.assertLess(ci_login, ci_pull)
        self.assertIn("      packages: read", ci_workflow)

        deploy_workflow = source(".github/workflows/deploy.yml")
        self.assertIn(
            "envs: DEPLOY_DIR,ECR_TOKEN,GHCR_TOKEN,GHCR_USER,"
            "POSTGRES_IMAGE,REGISTRY,TAG",
            deploy_workflow,
        )
        self.assertIn("GHCR_TOKEN: ${{ secrets.GITHUB_TOKEN }}", deploy_workflow)
        deploy_job = deploy_workflow.split("\n  deploy:\n", maxsplit=1)[1]
        self.assertIn("      packages: read", deploy_job)
        self.assertNotIn("      packages: write", deploy_job)

        deploy_script = source("scripts/deploy-remote.sh")
        server_login = deploy_script.index(
            'docker login --username "$GHCR_USER" --password-stdin ghcr.io'
        )
        server_pull = deploy_script.index(
            "docker compose -f docker-compose.yml -f docker-compose.prod.yml "
            "pull postgres dragonfly"
        )
        self.assertLess(server_login, server_pull)

    def test_postgres_smoke_test_provides_compose_env_file(self) -> None:
        workflow = source(".github/workflows/ci.yml")
        postgres_job = workflow.split("\n  postgres:\n", maxsplit=1)[1]
        postgres_job = postgres_job.split("\n  once:\n", maxsplit=1)[0]

        create_env = postgres_job.index("      - name: Create Compose environment file")
        verify = postgres_job.index("      - name: Verify PostgreSQL retrieval image")
        cleanup = postgres_job.index("      - name: Stop PostgreSQL retrieval image")

        self.assertIn("        run: touch .env", postgres_job)
        self.assertLess(create_env, verify)
        self.assertLess(create_env, cleanup)

    def test_react_doctor_pr_filter_cannot_leave_required_check_pending(self) -> None:
        workflow = source(".github/workflows/react-doctor.yml")
        pull_request_trigger = workflow.split("  pull_request:\n", maxsplit=1)[1]
        pull_request_trigger = pull_request_trigger.split("\n  push:\n", maxsplit=1)[0]

        self.assertNotIn("paths:", pull_request_trigger)
        self.assertIn("  changes:\n", workflow)
        self.assertIn("uses: dorny/paths-filter@v3", workflow)
        self.assertIn("  react-doctor:\n", workflow)
        self.assertIn("      always() &&", workflow)
        self.assertIn("needs.changes.outputs.admin_spa == 'true'", workflow)


if __name__ == "__main__":
    unittest.main()
