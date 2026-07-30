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
    def test_deploy_accepts_only_verified_candidate_inputs(self) -> None:
        workflow = source(".github/workflows/deploy.yml")
        stable = source(".github/workflows/stable.yml")

        self.assertIn("  workflow_call:", workflow)
        self.assertNotIn("  push:\n", workflow)
        self.assertNotIn("  workflow_dispatch:\n", workflow)
        self.assertNotIn("\n  test:\n", workflow)
        self.assertNotIn("pnpm test:e2e", workflow)
        self.assertNotIn("go test ./...", workflow)
        self.assertNotIn("docker/build-push-action", workflow)
        for candidate_input in (
            "sha",
            "app_digest",
            "admin_digest",
            "postgres_image",
        ):
            with self.subTest(candidate_input=candidate_input):
                self.assertIn(f"      {candidate_input}:\n", workflow)
        stable_deploy = stable.split("\n  deploy:\n", maxsplit=1)[1].split(
            "\n  publish:\n", maxsplit=1
        )[0]
        for permission in (
            "      contents: read\n",
            "      id-token: write\n",
            "      packages: write\n",
        ):
            with self.subTest(permission=permission.strip()):
                self.assertIn(permission, stable_deploy)

    def test_main_candidate_waits_for_one_exact_sha_ci_gate(self) -> None:
        workflow = source(".github/workflows/ci.yml")
        nightly = source(".github/workflows/nightly.yml")

        self.assertNotIn("  production-deploy:", workflow)
        self.assertIn("  gate:\n", workflow)
        self.assertIn("    name: Required CI gate", workflow)
        self.assertIn("    workflows: [CI]", nightly)
        self.assertIn(
            "github.event.workflow_run.conclusion == 'success'",
            nightly,
        )
        self.assertIn(
            "github.event.workflow_run.head_branch == 'main'",
            nightly,
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
        ci_workflow = source(".github/workflows/ci.yml")
        nightly = source(".github/workflows/nightly.yml")
        publish_job = ci_workflow.split(
            "\n  postgres-image:\n", maxsplit=1
        )[1].split("\n  admin-spa-e2e:\n", maxsplit=1)[0]

        self.assertIn(
            "needs.changes.outputs.postgres_image == 'true'",
            publish_job,
        )
        self.assertIn("push: true", publish_job)
        self.assertIn(
            "tags: ghcr.io/p-n-ai/pai-postgres:${{ github.sha }}",
            publish_job,
        )
        self.assertIn(
            'deployment_tag="ghcr.io/p-n-ai/pai-postgres:$SHA"',
            nightly,
        )
        self.assertNotIn("Dockerfile.postgres", nightly)

    def test_postgres_deployment_uses_an_immutable_digest(self) -> None:
        nightly = source(".github/workflows/nightly.yml")
        stable = source(".github/workflows/stable.yml")
        deploy_workflow = source(".github/workflows/deploy.yml")

        self.assertIn(
            'echo "POSTGRES_IMAGE=ghcr.io/p-n-ai/pai-postgres@$digest"',
            nightly,
        )
        self.assertIn(
            "--arg postgres_image \"$POSTGRES_IMAGE\"",
            nightly,
        )
        self.assertIn(
            "postgres_image: $postgres_image",
            nightly,
        )
        self.assertIn(
            "postgres_image: ${{ needs.candidate.outputs.postgres_image }}",
            stable,
        )
        self.assertIn(
            "POSTGRES_IMAGE: ${{ inputs.postgres_image }}",
            deploy_workflow,
        )
        self.assertIn(
            'run: docker manifest inspect "$POSTGRES_IMAGE"',
            deploy_workflow,
        )

        compose = source("docker-compose.yml")
        self.assertIn("image: ${POSTGRES_IMAGE:-", compose)

    def test_repository_contracts_run_in_exact_sha_ci_gate(self) -> None:
        workflow = source(".github/workflows/ci.yml")

        self.assertIn("  config:\n", workflow)
        self.assertIn("      - 'scripts/*.py'", workflow)
        self.assertIn("python3 -m unittest discover", workflow)
        gate = workflow.split("\n  gate:\n", maxsplit=1)[1]
        self.assertIn("      - config\n", gate)
        self.assertIn("RESULTS: ${{ toJSON(needs) }}", gate)
        self.assertIn(
            'all(.[]; .result == "success" or .result == "skipped")',
            gate,
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
            "POSTGRES_IMAGE,REGISTRY,TAG,APP_DIGEST,ADMIN_DIGEST",
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
            "      - admin-spa-e2e\n",
            workflow.split("\n  gate:\n", maxsplit=1)[1],
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

    def test_manual_stable_releases_require_main(self) -> None:
        workflow = source(".github/workflows/stable.yml")
        guard = named_step(workflow, "Validate version and candidate run")

        self.assertIn('[ "$GITHUB_REF" != "refs/heads/main" ]', guard)
        self.assertIn("exit 1", guard)
        self.assertLess(
            workflow.index("Validate version and candidate run"),
            workflow.index("uses: ./.github/workflows/deploy.yml"),
        )

    def test_react_doctor_is_observed_by_required_gate(self) -> None:
        workflow = source(".github/workflows/ci.yml")
        doctor = workflow.split("\n  react-doctor:\n", maxsplit=1)[1]
        doctor = doctor.split("\n  postgres-image:\n", maxsplit=1)[0]
        gate = workflow.split("\n  gate:\n", maxsplit=1)[1]

        self.assertIn("needs.changes.outputs.admin_spa == 'true'", doctor)
        self.assertIn("uses: millionco/react-doctor@v2", doctor)
        self.assertIn("      - react-doctor\n", gate)
        self.assertFalse((ROOT / ".github/workflows/react-doctor.yml").exists())

    def test_app_and_admin_candidate_builds_resume_existing_sha_images(
        self,
    ) -> None:
        workflow = source(".github/workflows/nightly.yml")

        for image in ("app", "admin"):
            with self.subTest(image=image):
                build = named_step(
                    workflow,
                    f"Build and push SHA-addressed {image} image",
                )

                self.assertIn(
                    "steps.artifact.outputs.exists != 'true'",
                    build,
                )
                self.assertIn(
                    f"steps.existing.outputs.{image}_exists != 'true'",
                    build,
                )
                self.assertIn("uses: docker/build-push-action@", build)
                self.assertIn("push: true", build)
                self.assertIn(
                    "${{ github.event.workflow_run.head_sha }}",
                    build,
                )
                self.assertIn(f"scope=nightly-{image}", build)
                self.assertNotIn(f"/pai-bot/{image}:latest", build)

    def test_postgres_reuse_and_rebuild_paths_fail_closed(self) -> None:
        workflow = source(".github/workflows/nightly.yml")
        resolve = named_step(workflow, "Resolve PostgreSQL candidate")

        recorded = resolve.index('resolve_digest "$deployment_tag"')
        semantic_fallback = resolve.index(
            "ghcr.io/p-n-ai/pai-postgres:deployed"
        )
        immutable_tag = resolve.index("docker buildx imagetools create")
        validation = resolve.index(
            'if [[ "$digest" != sha256:* ]]; then',
            immutable_tag,
        )
        output = resolve.index(
            'echo "POSTGRES_IMAGE=ghcr.io/p-n-ai/pai-postgres@$digest"'
        )
        self.assertLess(recorded, semantic_fallback)
        self.assertLess(semantic_fallback, immutable_tag)
        self.assertLess(immutable_tag, validation)
        self.assertLess(validation, output)
        self.assertIn("no immutable PostgreSQL source image is available", resolve)
        self.assertIn("docker compose config --variables", resolve)

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
            "POSTGRES_IMAGE,REGISTRY,TAG,APP_DIGEST,ADMIN_DIGEST",
            deploy,
        )
        self.assertIn(
            "POSTGRES_IMAGE: ${{ inputs.postgres_image }}",
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

    def test_successful_deploy_records_only_postgres_compatibility_aliases(
        self,
    ) -> None:
        script = source("scripts/deploy-remote.sh")
        smoke_result = script.index('if [ "$SMOKE_FAIL" -gt 0 ]; then')
        record = script.index(
            'echo "--- Recording successfully deployed PostgreSQL aliases ---"'
        )
        success = script.index('echo "Deploy successful (image: $TAG)"')

        self.assertLess(smoke_result, record)
        self.assertLess(record, success)
        self.assertNotIn('"/pai-bot/app:deployed"', script)
        self.assertNotIn('"/pai-bot/admin:deployed"', script)
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

    def test_candidate_image_builds_use_shared_caches(self) -> None:
        workflow = source(".github/workflows/nightly.yml")
        ci_workflow = source(".github/workflows/ci.yml")

        for step_name in (
            "Build and publish PostgreSQL candidate",
            "Build changed PostgreSQL retrieval image",
            "Build Docker image",
            "Build admin Docker image",
            "Build ONCE image",
            "Build and push SHA-addressed app image",
            "Build and push SHA-addressed admin image",
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


if __name__ == "__main__":
    unittest.main()
