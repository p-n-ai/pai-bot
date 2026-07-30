# Nightly candidates and stable releases

P&AI separates building a release from deploying it. A successful `CI` run for
a push to `main` produces a digest-pinned nightly candidate, but it never
deploys production. A maintainer must explicitly promote one successful
candidate through the `Stable release` workflow.

This runbook is for maintainers who create candidates, promote releases, or
diagnose release failures.

## Release guarantees

The release workflows enforce these invariants:

- Pull requests and `main` must pass the aggregate `Required CI gate`.
- Only a successful `CI` push run on `main` can produce a nightly candidate.
- A candidate records immutable app, admin, and PostgreSQL image digests.
- Stable promotion downloads an existing candidate and never rebuilds it.
- Production receives the candidate commit and image digests before a semantic
  tag or GitHub Release is published.
- Production deployment is manual. A merge to `main` does not deploy.
- A failed deployment does not publish a new semantic tag or GitHub Release.

The workflow sequence is:

```text
pull request
  -> Required CI gate
  -> merge to main
  -> successful main CI
  -> Nightly candidate artifact
  -> manual Stable release
  -> production deployment
  -> vMAJOR.MINOR.PATCH tag and GitHub Release
```

## Operator quick start

1. Complete [release access configuration](#configure-release-access) once.
2. [Find a successful candidate](#find-successful-candidates) and record its
   Nightly run ID.
3. [Inspect its provenance](#inspect-candidate-provenance) when the change or
   deployment risk warrants a manual audit.
4. [Dispatch `Stable release`](#promote-a-stable-release) from `main` with the
   candidate run ID and a new semantic version.
5. Follow the run through any `production` environment approval, then
   [verify the published release](#verify-a-release).

Stop before promotion if candidate provenance, migration compatibility, or
production recovery is unclear.

## Configure release access

Candidate creation and stable deployment use GitHub Actions secrets. Keep
secret values in repository or environment settings; never add them to the
repository or paste them into workflow logs.

| Scope | Configuration | Where it must be available |
|---|---|---|
| Candidate registry | `AWS_ROLE_ARN`, `AWS_REGION`, `ECR_REGISTRY` | Repository or organization secrets, because Nightly does not use a GitHub environment |
| Deployment host | `DEPLOY_HOST`, `DEPLOY_USER`, `DEPLOY_KEY`, `DEPLOY_DIR` | Repository secrets or the `production` environment |
| Required production values | `POSTGRES_PASSWORD`, `PAI_AUTH_SECRET` (or the legacy `LEARN_AUTH_JWT_SECRET`), `PAI_CONFIG_ENCRYPTION_KEY`, `PAI_AUTH_BOOTSTRAP_ADMIN_PASSWORD` | Repository secrets or the `production` environment |
| Conditional runtime values | Chat adapter, AI provider, Google login, email, domain, previous encryption keys, and feature configuration used by the deployment | Repository secrets, variables, or the `production` environment as referenced by the deploy workflow |

The AWS role must support OIDC authentication, ECR login, app/admin image
push and pull, and `ecr:BatchGetImage`. Nightly candidate creation uses the
authenticated registry interface to inspect image manifests; it does not
require ECR repository-administration permissions.

The reusable deploy workflow targets the GitHub `production` environment.
Configure required reviewers and deployment-branch restrictions in repository
settings if production must have an approval gate beyond the manual workflow
dispatch. The workflow YAML references the environment but does not create
those protection rules.

See [Runtime AI settings](operations/runtime-ai-settings.md) for the production
AI and secret-rotation contract. The exact values consumed during deployment
are defined in [the deploy workflow](../.github/workflows/deploy.yml).

## Create a nightly candidate

No operator action is normally required. The `Nightly candidate` workflow
starts after `CI` completes successfully for a push to `main`.

The workflow:

1. Checks out the exact successful `main` SHA.
2. Resolves or records a SHA-addressed PostgreSQL image.
3. Reuses app and admin images already present for that SHA.
4. Builds only confirmed-missing app or admin images.
5. Resolves all three registry digests.
6. Uploads `candidate.json` as
   `nightly-candidate-<40-character-sha>` with 30-day retention.

Registry inspection is fail-closed. Only a confirmed `manifest unknown` or
not-found response marks an app/admin image as missing. Authentication,
network, and other registry errors stop the workflow instead of rebuilding
over an existing SHA tag.

### Find successful candidates

List recent candidates from a checkout authenticated with `gh`:

```bash
gh run list \
  --workflow nightly.yml \
  --branch main \
  --status success \
  --limit 10 \
  --json databaseId,headSha,createdAt,url
```

Keep both the Nightly workflow run ID and its full commit SHA. Stable promotion
requires the run ID and verifies the SHA from the artifact.

### Inspect candidate provenance

Download the artifact before promotion when you need to audit its contents:

```bash
candidate_run_id="REPLACE_WITH_NIGHTLY_RUN_ID"
candidate_sha="REPLACE_WITH_40_CHARACTER_MAIN_SHA"
artifact_dir=$(mktemp -d)

gh run download "$candidate_run_id" \
  --name "nightly-candidate-$candidate_sha" \
  --dir "$artifact_dir"
jq . "$artifact_dir/candidate.json"
```

`candidate.json` has this contract:

```json
{
  "repository": "p-n-ai/pai-bot",
  "sha": "<40-character-main-sha>",
  "nightly_run_id": 123456789,
  "source_run_id": 123456700,
  "app_image": "<ecr-registry>/pai-bot/app@sha256:<64-hex-digest>",
  "admin_image": "<ecr-registry>/pai-bot/admin@sha256:<64-hex-digest>",
  "postgres_image": "ghcr.io/p-n-ai/pai-postgres@sha256:<64-hex-digest>"
}
```

| Field | Meaning |
|---|---|
| `repository` | Repository that produced the candidate |
| `sha` | Successful `main` commit to deploy and tag |
| `nightly_run_id` | Nightly workflow run that owns the artifact |
| `source_run_id` | Successful `CI` push run that authorized candidate creation |
| `app_image` | Immutable backend image reference |
| `admin_image` | Immutable admin SPA image reference |
| `postgres_image` | Immutable PostgreSQL image reference |

Do not edit or repack this artifact. Stable promotion validates the repository,
run IDs, artifact name, source CI run, SHA, registry paths, and digest formats.

### Retry a nightly run

Rerunning the same Nightly workflow is safe:

- If its complete candidate artifact already exists, the rerun is a no-op.
- If no artifact exists, app and admin images are checked independently.
- Confirmed-missing images are built; existing images are reused.
- Multiple matching candidate artifacts fail as ambiguous.

If a candidate has expired, rerun its successful source CI or use the next
green `main` commit to produce a fresh Nightly run. Promote the new run ID.

## Promote a stable release

Promotion is a deliberate production action. Choose a successful, unexpired
nightly candidate and a new version matching `vMAJOR.MINOR.PATCH`, with no
leading zeroes in numeric components.

Run:

```bash
candidate_run_id="REPLACE_WITH_SUCCESSFUL_NIGHTLY_RUN_ID"
version=v1.2.3

gh workflow run stable.yml \
  --ref main \
  -f candidate_run_id="$candidate_run_id" \
  -f version="$version"
```

Then follow the dispatched run:

```bash
gh run list \
  --workflow stable.yml \
  --limit 5 \
  --json databaseId,status,conclusion,createdAt,url
```

The candidate validation job rejects promotion unless:

- the workflow was dispatched from `main`;
- the run ID is a positive integer;
- the version exactly matches `vMAJOR.MINOR.PATCH`;
- the referenced run is a successful `Nightly candidate` run for `main`;
- exactly one unexpired candidate artifact exists;
- the artifact and manifest SHA agree;
- the recorded source is a successful `CI` push run for the same `main` SHA;
- all image references use the expected repositories and valid SHA-256 digests;
- an existing version tag, if present, already points to the candidate SHA.

### What production deployment does

After provenance validation, the reusable deploy workflow:

1. Checks out the recorded candidate SHA.
2. Verifies the app, admin, and PostgreSQL digests still exist.
3. Validates required production secrets.
4. Copies Compose files, migrations, Caddy files, the deployment script, and
   migration preflight files to the server.
5. Writes the server `.env` atomically while preserving `.env.rollback`.
6. Pulls the exact candidate digests.
7. Creates and verifies a pre-migration PostgreSQL backup.
8. Runs identity preflight checks and migrations.
9. Starts the candidate and verifies:
   - app container health;
   - app and admin image identities;
   - `/healthz`;
   - application and primary AI-provider status;
   - admin HTTP availability;
   - non-mutating bot command smoke tests.
10. Records the successfully deployed PostgreSQL aliases.

Only after deployment succeeds does the publish job create the semantic tag
and GitHub Release.

### Approval and publication order

`Stable release` is manual and serialized by the `deploy-production`
concurrency group. If the `production` environment has required reviewers, the
deploy job waits for approval there.

Never create or move the requested semantic tag by hand while a stable run is
in progress. The workflow owns publication and guarantees this order:

```text
candidate validation -> production deployment -> tag -> GitHub Release
```

### Retry a stable release

Stable promotion is retry-safe when the candidate and version do not change:

- A deployment failure publishes no tag or GitHub Release.
- A tag already pointing to the candidate SHA is accepted.
- A tag pointing to another SHA is rejected and is never moved.
- An existing GitHub Release is treated as complete.
- If tag creation succeeds but Release creation fails, rerunning the same
  candidate and version creates only the missing Release.

After an operational failure is corrected, rerun with the same candidate run
ID and version. Use a new candidate if the application, migration, or image
contents must change.

## Rollback and migration safety

The remote deploy script records the currently running app/admin image pair
before rollout. After candidate images are pulled and rollback is armed, a
failed command, health check, image-identity check, or smoke test attempts to
restore:

- the previous app and admin images;
- the previous `.env`;
- the previous app/admin containers.

Rollback requires a complete previous app/admin image pair. It may be
unavailable on a first deployment.

Database rollback is intentionally not automatic. The workflow creates a
verified pre-migration backup, but application rollback does not reverse
migrations or restore that backup. Every stable candidate migration must be
backward-compatible with the previously deployed application. For an
incompatible migration, document and review a coordinated rollout and database
recovery procedure before promotion.

## Troubleshoot failures

| Symptom | Meaning and action |
|---|---|
| Nightly did not start | Confirm the source run is named `CI`, was triggered by a push to `main`, and completed successfully. Pull request and manually dispatched CI runs do not create candidates. |
| `registry inspection failed` | Check AWS OIDC authentication, ECR login permissions, registry availability, and network health. The workflow intentionally refuses to treat this as a missing image. |
| No immutable PostgreSQL source image | Confirm source CI published the SHA-addressed PostgreSQL image, or that the deployed/base PostgreSQL image is available in GHCR. |
| Candidate artifact is missing, expired, or ambiguous | Produce a fresh successful Nightly run and use its run ID. Do not reconstruct candidate metadata manually. |
| Candidate provenance is rejected | Verify the run ID belongs to this repository, the run succeeded on `main`, and the artifact was not edited or repacked. |
| Version is rejected | Use a new exact `vMAJOR.MINOR.PATCH` version. Never retarget an existing stable tag. |
| Deployment waits for approval | Review the pending deployment in the GitHub `production` environment. |
| Production secret validation fails | Correct the named GitHub secret or environment configuration; do not weaken the validation command. |
| Deployment or health checks fail | Inspect the deploy job and rollback output. Correct the operational issue and rerun the same candidate, or create a new candidate for code/image changes. |
| Tag exists but GitHub Release is missing | Rerun the same stable candidate and version; publication completes idempotently. |

## Verify a release

A stable run is complete only when:

- the `Stable release` workflow is successful;
- production is running the recorded app/admin image IDs;
- the semantic tag resolves to the candidate SHA;
- the GitHub Release exists for that tag.

Inspect publication with:

```bash
version=v1.2.3
git ls-remote --tags origin "refs/tags/$version"
gh release view "$version" --json tagName,url
```

## Validate release changes locally

Run the focused contract suite whenever changing release workflows,
`candidate.json`, deployment behavior, or this runbook:

```bash
GOCACHE=/tmp/pai-bot-go-cache \
  go test ./internal/releaseworkflow -count=1

python3 -m unittest \
  scripts.test_ci_config \
  scripts.test_deployment_config \
  scripts.test_uptime_config

GOCACHE=/tmp/pai-bot-go-cache \
  go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.7 \
  .github/workflows/ci.yml \
  .github/workflows/nightly.yml \
  .github/workflows/stable.yml \
  .github/workflows/deploy.yml

bash -n scripts/deploy-remote.sh
git diff --check
```

For a release-affecting code change, also run the smallest behavior-level tests
for the changed boundary and the full repository suite required by CI.

## Keep this runbook current

Treat these files as the release source of truth:

- `.github/workflows/ci.yml`
- `.github/workflows/nightly.yml`
- `.github/workflows/stable.yml`
- `.github/workflows/deploy.yml`
- `scripts/deploy-remote.sh`
- `internal/releaseworkflow/workflows_test.go`

Update this runbook when workflow inputs, candidate fields, secret names,
artifact retention, validation rules, publication order, health checks, or
rollback behavior changes.
