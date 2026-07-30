#!/usr/bin/env python3
"""Repository contracts for CI-to-deployment orchestration."""

from __future__ import annotations

import json
import os
import pathlib
import subprocess
import unittest


ROOT = pathlib.Path(__file__).resolve().parents[1]


def source(path: str) -> str:
    return (ROOT / path).read_text()


def named_step(workflow: str, name: str) -> str:
    marker = f"      - name: {name}\n"
    start = workflow.index(marker)
    end = workflow.find("\n      - name:", start + len(marker))
    return workflow[start:] if end == -1 else workflow[start:end]


class CIWorkflowTests(unittest.TestCase):
    def test_deploy_is_reusable_and_does_not_repeat_ci_tests(self) -> None:
        workflow = source(".github/workflows/deploy.yml")

        self.assertIn("  workflow_call:", workflow)
        self.assertNotIn("  push:\n", workflow)
        self.assertNotIn("\n  test:\n", workflow)
        self.assertNotIn("pnpm test:e2e", workflow)
        self.assertNotIn("go test ./...", workflow)
        self.assertNotIn("inputs.deployable", workflow)

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
            "postgres-image",
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

        compose = source("docker-compose.yml")
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
        ci_pull = ci_workflow.index(
            "      - name: Pull published PostgreSQL retrieval image"
        )

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
        self.assertIn("      packages: write", deploy_job)

        deploy_script = source("scripts/deploy-remote.sh")
        server_login = deploy_script.index(
            'docker login --username "$GHCR_USER" --password-stdin ghcr.io'
        )
        server_pull = deploy_script.index(
            "docker compose -f docker-compose.yml -f docker-compose.prod.yml "
            "pull postgres dragonfly"
        )
        self.assertLess(server_login, server_pull)

    def test_changed_postgres_image_runs_inside_backend_e2e(self) -> None:
        workflow = source(".github/workflows/ci.yml")
        publish_job = workflow.split("\n  postgres-image:\n", maxsplit=1)[1]
        publish_job = publish_job.split("\n  admin-spa-e2e:\n", maxsplit=1)[0]
        e2e = workflow.split("\n  admin-spa-e2e:\n", maxsplit=1)[1]
        e2e = e2e.split("\n  go:\n", maxsplit=1)[0]

        build = named_step(workflow, "Build changed PostgreSQL retrieval image")
        pull_candidate = named_step(workflow, "Pull PostgreSQL candidate")
        pull = named_step(workflow, "Pull published PostgreSQL retrieval image")
        select = named_step(workflow, "Select changed PostgreSQL retrieval image")
        start = named_step(workflow, "Start PostgreSQL and Dragonfly")

        self.assertIn("github.event_name == 'push' &&", publish_job)
        self.assertIn("uses: docker/build-push-action@v6", publish_job)
        self.assertIn("push: true", publish_job)
        self.assertIn(
            "tags: ghcr.io/p-n-ai/pai-postgres:${{ github.sha }}",
            publish_job,
        )
        self.assertIn("      - postgres-image\n", e2e)
        self.assertIn("needs.postgres-image.result == 'success'", e2e)
        self.assertIn(
            "github.event_name == 'pull_request' &&",
            build,
        )
        self.assertIn(
            "if: needs.changes.outputs.postgres_image == 'true'",
            select,
        )
        self.assertIn("uses: docker/build-push-action@v6", build)
        self.assertIn("load: true", build)
        self.assertIn("scope=ci-postgres", build)
        self.assertIn("github.event_name == 'push' &&", pull_candidate)
        self.assertIn(
            "if: needs.changes.outputs.postgres_image != 'true'",
            pull,
        )
        self.assertLess(e2e.index(build), e2e.index(start))
        self.assertLess(e2e.index(pull_candidate), e2e.index(start))
        self.assertLess(e2e.index(pull), e2e.index(start))
        self.assertLess(e2e.index(select), e2e.index(start))
        self.assertNotIn("\n  postgres:\n", workflow)

    def test_compose_runtime_changes_run_backend_e2e(self) -> None:
        workflow = source(".github/workflows/ci.yml")
        e2e = workflow.split("\n  admin-spa-e2e:\n", maxsplit=1)[1]
        e2e = e2e.split("\n  go:\n", maxsplit=1)[0]

        self.assertIn("compose_runtime:", workflow)
        self.assertIn("              - '.github/compose.e2e.yml'", workflow)
        self.assertIn("              - 'docker-compose.yml'", workflow)
        self.assertIn("              - 'docker-compose.prod.yml'", workflow)
        self.assertIn(
            "needs.changes.outputs.compose_runtime == 'true'",
            e2e,
        )
        self.assertIn(
            "(needs.admin-spa-e2e.result == 'success' || "
            "needs.admin-spa-e2e.result == 'skipped')",
            workflow,
        )
        start = named_step(workflow, "Start PostgreSQL and Dragonfly")
        self.assertIn("-f docker-compose.prod.yml", start)
        self.assertIn("-f .github/compose.e2e.yml", start)
        self.assertIn("POSTGRES_PASSWORD: pai", start)
        for name in (
            "PAI_AUTH_SECRET",
            "PAI_CONFIG_ENCRYPTION_KEY",
            "PAI_CONFIG_PREVIOUS_ENCRYPTION_KEYS",
            "PAI_AUTH_BOOTSTRAP_ADMIN_EMAIL",
            "PAI_AUTH_BOOTSTRAP_ADMIN_PASSWORD",
        ):
            with self.subTest(environment=name):
                self.assertIn(f"{name}:", start)

    def test_e2e_compose_merge_preserves_production_runtime_and_test_ports(
        self,
    ) -> None:
        env = os.environ.copy()
        env.update(
            {
                "POSTGRES_IMAGE": (
                    "ghcr.io/p-n-ai/pai-postgres@sha256:"
                    + ("a" * 64)
                ),
                "POSTGRES_PASSWORD": "pai",
                "PAI_AUTH_SECRET": "test-auth-secret-with-enough-variety",
                "PAI_CONFIG_ENCRYPTION_KEY": "test-config-encryption-key-12345",
                "PAI_CONFIG_PREVIOUS_ENCRYPTION_KEYS": "[]",
                "PAI_AUTH_BOOTSTRAP_ADMIN_EMAIL": "admin@example.com",
                "PAI_AUTH_BOOTSTRAP_ADMIN_PASSWORD": "test-bootstrap-password",
            }
        )
        env_path = ROOT / ".env"
        created_env = not env_path.exists()
        if created_env:
            env_path.touch(mode=0o600)
        try:
            result = subprocess.run(
                [
                    "docker",
                    "compose",
                    "-f",
                    "docker-compose.yml",
                    "-f",
                    "docker-compose.prod.yml",
                    "-f",
                    ".github/compose.e2e.yml",
                    "config",
                    "--format",
                    "json",
                ],
                cwd=ROOT,
                env=env,
                capture_output=True,
                text=True,
                timeout=30,
            )
        finally:
            if created_env:
                env_path.unlink()
        self.assertEqual(result.returncode, 0, result.stderr)
        services = json.loads(result.stdout)["services"]

        self.assertNotIn("build", services["postgres"])
        self.assertEqual(
            services["postgres"]["image"],
            env["POSTGRES_IMAGE"],
        )
        self.assertEqual(
            {port["target"] for port in services["postgres"]["ports"]},
            {5432},
        )
        self.assertEqual(
            {port["target"] for port in services["dragonfly"]["ports"]},
            {6379},
        )

    def test_manual_production_deploys_require_main(self) -> None:
        workflow = source(".github/workflows/deploy.yml")
        guard = named_step(workflow, "Require main for manual production deploy")
        first_build = workflow.index("      - name: Build and push app image")

        self.assertIn("github.event_name == 'workflow_dispatch' &&", guard)
        self.assertIn("github.ref != 'refs/heads/main'", guard)
        self.assertIn("exit 1", guard)
        self.assertLess(workflow.index(guard), first_build)

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

    def test_app_and_admin_builds_are_mutually_exclusive_with_promotion(
        self,
    ) -> None:
        workflow = source(".github/workflows/deploy.yml")

        for image in ("app", "admin"):
            with self.subTest(image=image):
                build = named_step(workflow, f"Build and push {image} image")
                promote = named_step(
                    workflow,
                    f"Promote deployed {image} image to deployment SHA",
                )

                self.assertIn(
                    "github.event_name == 'workflow_dispatch' ||",
                    build,
                )
                self.assertIn(f"inputs.{image}_changed", build)
                self.assertIn("uses: docker/build-push-action@v6", build)
                self.assertIn("push: true", build)
                self.assertIn(f"scope=deploy-{image}", build)
                self.assertNotIn(f"/pai-bot/{image}:latest", build)

                self.assertIn(
                    "github.event_name != 'workflow_dispatch' &&",
                    promote,
                )
                self.assertIn(f"inputs.{image}_changed == false", promote)
                self.assertIn("docker buildx imagetools create", promote)
                self.assertIn(
                    f"/pai-bot/{image}:${{{{ github.sha }}}}",
                    promote,
                )
                self.assertIn(f"/pai-bot/{image}:deployed", promote)
                self.assertIn(f"/pai-bot/{image}:latest", promote)
                self.assertIn('source="$DEPLOYED_IMAGE"', promote)
                self.assertIn('source="$LEGACY_IMAGE"', promote)

    def test_postgres_reuse_and_rebuild_paths_fail_closed(self) -> None:
        workflow = source(".github/workflows/deploy.yml")
        recorded = named_step(workflow, "Resolve recorded PostgreSQL image")
        build = named_step(workflow, "Build and push PostgreSQL retrieval image")
        resolve = named_step(workflow, "Resolve immutable PostgreSQL image")

        self.assertIn('case "$digest" in', recorded)
        self.assertIn("Invalid recorded PostgreSQL image digest", recorded)
        self.assertIn("exit 1", recorded)

        self.assertIn(
            "steps.recorded-postgres.outputs.exists != 'true'",
            build,
        )
        self.assertIn("inputs.postgres_changed ||", build)
        self.assertIn(
            "github.event_name == 'workflow_dispatch' &&",
            build,
        )
        self.assertIn("inputs.rebuild_postgres", build)
        self.assertIn("scope=deploy-postgres", build)

        built = resolve.index('digest="$BUILT_DIGEST"')
        recorded_fallback = resolve.index('digest="$RECORDED_DIGEST"')
        semantic_fallback = resolve.index("docker buildx imagetools inspect")
        validation = resolve.index('case "$digest" in')
        output = resolve.index(
            'echo "image=ghcr.io/p-n-ai/pai-postgres@$digest"'
        )
        self.assertLess(built, recorded_fallback)
        self.assertLess(recorded_fallback, semantic_fallback)
        self.assertLess(semantic_fallback, validation)
        self.assertLess(validation, output)
        self.assertIn("Invalid PostgreSQL image digest", resolve)
        self.assertIn("docker compose config --variables", resolve)
        self.assertIn("ghcr.io/p-n-ai/pai-postgres:deployed", resolve)
        self.assertNotIn("pggraph-1.0.0-pgvector-0.8.5", workflow)

    def test_production_postgres_is_pulled_by_digest_without_a_local_build(
        self,
    ) -> None:
        compose = source("docker-compose.yml")
        production_compose = source("docker-compose.prod.yml")
        self.assertIn("image: ${POSTGRES_IMAGE:-", compose)
        self.assertIn("build: !reset null", production_compose)
        self.assertNotIn("image: ${POSTGRES_IMAGE:-", production_compose)

        workflow = source(".github/workflows/deploy.yml")
        deploy = named_step(workflow, "Deploy")
        self.assertIn(
            "envs: DEPLOY_DIR,ECR_TOKEN,GHCR_TOKEN,GHCR_USER,"
            "POSTGRES_IMAGE,REGISTRY,TAG",
            deploy,
        )
        self.assertIn(
            "POSTGRES_IMAGE: ${{ needs.push-images.outputs.postgres-image }}",
            deploy,
        )

        script = source("scripts/deploy-remote.sh")
        login = script.index(
            'docker login --username "$GHCR_USER" --password-stdin ghcr.io'
        )
        pull = script.index(
            "docker compose -f docker-compose.yml -f docker-compose.prod.yml "
            "pull postgres dragonfly"
        )
        up = script.index(
            "docker compose -f docker-compose.yml -f docker-compose.prod.yml "
            "up -d postgres dragonfly"
        )
        self.assertLess(login, pull)
        self.assertLess(pull, up)
        self.assertNotIn(
            "docker compose -f docker-compose.yml -f docker-compose.prod.yml "
            "build postgres",
            script,
        )

    def test_successful_deploy_records_stable_image_aliases(self) -> None:
        script = source("scripts/deploy-remote.sh")
        smoke_result = script.index('if [ "$SMOKE_FAIL" -gt 0 ]; then')
        record = script.index(
            'echo "--- Recording successfully deployed image aliases ---"'
        )
        success = script.index('echo "Deploy successful (image: $TAG)"')

        self.assertLess(smoke_result, record)
        self.assertLess(record, success)
        self.assertIn("for component in app admin; do", script)
        self.assertIn("for alias in deployed latest; do", script)
        self.assertIn(
            'source_image="$REGISTRY/pai-bot/$component:$TAG"',
            script,
        )
        self.assertIn('docker push "$alias_image"', script)
        self.assertIn('postgres_repository=${POSTGRES_IMAGE%@*}', script)
        self.assertIn('"$postgres_repository:deployed"', script)
        self.assertIn('"$postgres_release_image"', script)
        self.assertIn(
            "config --variables",
            script,
        )

    def test_app_dockerfile_keeps_dependency_cache_and_narrow_source_copies(
        self,
    ) -> None:
        dockerfile = source("deploy/docker/Dockerfile")

        dependency_copy = dockerfile.index("COPY go.mod go.sum ./")
        dependency_download = dockerfile.index("RUN go mod download")
        command_copy = dockerfile.index("COPY cmd ./cmd")
        internal_copy = dockerfile.index("COPY internal ./internal")
        build = dockerfile.index("RUN CGO_ENABLED=0 go build")

        self.assertLess(dependency_copy, dependency_download)
        self.assertLess(dependency_download, command_copy)
        self.assertLess(command_copy, build)
        self.assertLess(internal_copy, build)
        self.assertNotIn("COPY . .", dockerfile)
        self.assertIn("COPY oss /oss", dockerfile)
        self.assertIn("COPY migrations /migrations", dockerfile)

        dockerignore = source(".dockerignore")
        for ignored in (
            ".git",
            ".github",
            ".agents",
            ".codex",
            ".env",
            "docs",
            "site",
            "terraform",
        ):
            with self.subTest(ignored=ignored):
                self.assertIn(f"{ignored}\n", dockerignore)

    def test_image_metrics_distinguish_rebuild_promotion_and_reuse(self) -> None:
        workflow = source(".github/workflows/deploy.yml")
        ci_workflow = source(".github/workflows/ci.yml")
        metrics = named_step(workflow, "Record image-build metrics")
        postgres_record = named_step(
            workflow,
            "Record PostgreSQL image for deployment SHA",
        )

        self.assertIn("admin_action=rebuilt", metrics)
        self.assertIn("admin_action=promoted", metrics)
        self.assertIn("app_action=rebuilt", metrics)
        self.assertIn("app_action=promoted", metrics)
        self.assertIn('echo "| App | $app_action |"', metrics)
        self.assertIn('echo "| Admin | $admin_action |"', metrics)
        self.assertIn('echo "| PostgreSQL | $POSTGRES_ACTION |"', metrics)
        self.assertIn("### Orchestration optimization baseline", metrics)
        self.assertIn(
            "Duplicate deployment test suites | 1 | 0 | 100%",
            metrics,
        )
        self.assertIn(
            "Routine unconditional production image builds | 4 | 0 | 100%",
            metrics,
        )
        self.assertIn(
            "Uncached heavy image-build paths | 6/8 | 0/8 | 100%",
            metrics,
        )
        self.assertIn(
            "Fixed structural score for this change: 100% "
            "(target: at least 80%)",
            metrics,
        )
        self.assertIn("without a 30-day measurement window", metrics)

        for step_name in (
            "Build and publish PostgreSQL candidate",
            "Build changed PostgreSQL retrieval image",
            "Build Docker image",
            "Build admin Docker image",
            "Build ONCE image",
            "Build and push app image",
            "Build and push admin image",
            "Build and push PostgreSQL retrieval image",
        ):
            with self.subTest(step=step_name):
                source_workflow = (
                    ci_workflow
                    if step_name in {
                        "Build and publish PostgreSQL candidate",
                        "Build changed PostgreSQL retrieval image",
                        "Build Docker image",
                        "Build admin Docker image",
                        "Build ONCE image",
                    }
                    else workflow
                )
                build = named_step(source_workflow, step_name)
                self.assertIn("cache-from: type=gha", build)
                self.assertIn("cache-to: type=gha,mode=max", build)

        self.assertIn("action=reused", postgres_record)
        self.assertIn("action=rebuilt", postgres_record)
        self.assertIn("action=promoted", postgres_record)


if __name__ == "__main__":
    unittest.main()
