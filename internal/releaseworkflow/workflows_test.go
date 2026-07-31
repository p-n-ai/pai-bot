// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package releaseworkflow

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

type workflowDocument map[string]any

func repositorySource(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", path))
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func parseWorkflow(source string) (workflowDocument, error) {
	var document workflowDocument
	if err := yaml.Unmarshal([]byte(source), &document); err != nil {
		return nil, err
	}
	return document, nil
}

func repositoryWorkflow(t *testing.T, path string) (string, workflowDocument) {
	t.Helper()
	source := repositorySource(t, path)
	document, err := parseWorkflow(source)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return source, document
}

func stringMap(value any) (map[string]any, bool) {
	switch typed := value.(type) {
	case map[string]any:
		return typed, true
	case workflowDocument:
		return map[string]any(typed), true
	default:
		return nil, false
	}
}

func workflowMap(document workflowDocument, key string) (map[string]any, error) {
	value, ok := stringMap(document[key])
	if !ok {
		return nil, fmt.Errorf("%s is not a map", key)
	}
	return value, nil
}

func workflowJob(document workflowDocument, name string) (map[string]any, error) {
	jobs, err := workflowMap(document, "jobs")
	if err != nil {
		return nil, err
	}
	job, ok := stringMap(jobs[name])
	if !ok {
		return nil, fmt.Errorf("job %q is missing", name)
	}
	return job, nil
}

func stringList(value any) ([]string, bool) {
	switch typed := value.(type) {
	case string:
		return []string{typed}, true
	case []any:
		values := make([]string, 0, len(typed))
		for _, item := range typed {
			text, ok := item.(string)
			if !ok {
				return nil, false
			}
			values = append(values, text)
		}
		return values, true
	default:
		return nil, false
	}
}

func exactMembers(actual, expected []string) bool {
	actual = slices.Clone(actual)
	expected = slices.Clone(expected)
	slices.Sort(actual)
	slices.Sort(expected)
	return slices.Equal(actual, expected)
}

func validGate(document workflowDocument) error {
	gate, err := workflowJob(document, "gate")
	if err != nil {
		return err
	}
	if gate["if"] != "always()" {
		return fmt.Errorf("gate must run with always()")
	}
	needs, ok := stringList(gate["needs"])
	if !ok {
		return fmt.Errorf("gate.needs is not a string list")
	}
	expected := []string{
		"changes",
		"config",
		"admin-spa",
		"react-doctor",
		"admin-spa-e2e",
		"go",
		"go-lint",
		"docker",
		"admin-docker",
		"postgres-image",
		"once",
		"api-compatibility",
	}
	if !exactMembers(needs, expected) {
		return fmt.Errorf("gate.needs = %v, want %v", needs, expected)
	}
	return nil
}

func validStableDAG(document workflowDocument) error {
	jobs, err := workflowMap(document, "jobs")
	if err != nil {
		return err
	}
	actualJobs := make([]string, 0, len(jobs))
	for name := range jobs {
		actualJobs = append(actualJobs, name)
	}
	if !exactMembers(actualJobs, []string{"candidate", "deploy", "publish"}) {
		return fmt.Errorf("stable jobs = %v", actualJobs)
	}
	deploy, err := workflowJob(document, "deploy")
	if err != nil {
		return err
	}
	deployNeeds, ok := stringList(deploy["needs"])
	if !ok || !exactMembers(deployNeeds, []string{"candidate"}) {
		return fmt.Errorf("deploy.needs = %v", deploy["needs"])
	}
	publish, err := workflowJob(document, "publish")
	if err != nil {
		return err
	}
	publishNeeds, ok := stringList(publish["needs"])
	if !ok || !exactMembers(publishNeeds, []string{"candidate", "deploy"}) {
		return fmt.Errorf("publish.needs = %v", publish["needs"])
	}
	return nil
}

func requireContains(t *testing.T, source string, values ...string) {
	t.Helper()
	for _, value := range values {
		if !strings.Contains(source, value) {
			t.Errorf("source is missing %q", value)
		}
	}
}

func requirePinnedActions(t *testing.T, document workflowDocument) {
	t.Helper()
	jobs, err := workflowMap(document, "jobs")
	if err != nil {
		t.Fatal(err)
	}
	pinned := regexp.MustCompile(`^[^@]+@[0-9a-f]{40}$`)
	for jobName, rawJob := range jobs {
		job, ok := stringMap(rawJob)
		if !ok {
			t.Fatalf("job %s is malformed", jobName)
		}
		steps, ok := job["steps"].([]any)
		if !ok {
			uses, reusable := job["uses"].(string)
			if reusable && strings.HasPrefix(uses, "./") {
				continue
			}
			t.Fatalf("job %s has malformed steps", jobName)
		}
		for _, rawStep := range steps {
			step, ok := stringMap(rawStep)
			if !ok {
				t.Fatalf("job %s has malformed step", jobName)
			}
			uses, ok := step["uses"].(string)
			if ok && !pinned.MatchString(uses) {
				t.Errorf("%s action %q is not pinned to a commit", jobName, uses)
			}
		}
	}
}

func workflowStepRun(t *testing.T, document workflowDocument, jobName, stepName string) string {
	t.Helper()
	job, err := workflowJob(document, jobName)
	if err != nil {
		t.Fatal(err)
	}
	steps, ok := job["steps"].([]any)
	if !ok {
		t.Fatalf("%s.steps is malformed", jobName)
	}
	for _, rawStep := range steps {
		step, ok := stringMap(rawStep)
		if !ok {
			t.Fatalf("%s contains a malformed step", jobName)
		}
		if step["name"] == stepName {
			run, ok := step["run"].(string)
			if !ok {
				t.Fatalf("%s/%s has no run script", jobName, stepName)
			}
			return run
		}
	}
	t.Fatalf("%s has no %q step", jobName, stepName)
	return ""
}

func writeExecutable(t *testing.T, path, source string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(source), 0o755); err != nil {
		t.Fatal(err)
	}
}

func runBash(t *testing.T, source string, environment ...string) error {
	t.Helper()
	command := exec.Command("bash", "-c", "set -euo pipefail\n"+source)
	command.Env = append(os.Environ(), environment...)
	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w\n%s", err, output)
	}
	return nil
}

func TestCIGateAndAPICompatibilityContracts(t *testing.T) {
	source, document := repositoryWorkflow(t, ".github/workflows/ci.yml")
	if err := validGate(document); err != nil {
		t.Fatal(err)
	}
	if _, err := workflowJob(document, "api-compatibility"); err != nil {
		t.Fatal(err)
	}
	requireContains(t, source,
		"github.event.pull_request.base.sha",
		"go run ./cmd/openapi",
		"go run ./cmd/apiroutes -root ../base/internal/server",
		"go run ./cmd/apiroutes -root internal/server",
		`select(.method == "POST" and .path == "/api/auth/refresh")`,
		`del(.paths["/api/auth/refresh"])`,
		"-base ../base-api-routes.json",
		"-head ../head-api-routes.json",
		"oasdiff breaking --fail-on ERR",
		`all(.[]; .result == "success" or .result == "skipped")`,
	)

	mutated := strings.Replace(source, "      - api-compatibility\n", "", 1)
	mutatedDocument, err := parseWorkflow(mutated)
	if err != nil {
		t.Fatal(err)
	}
	if err := validGate(mutatedDocument); err == nil {
		t.Fatal("gate validator accepted a gate without API compatibility")
	}
}

func TestNightlyCandidateContracts(t *testing.T) {
	source, document := repositoryWorkflow(t, ".github/workflows/nightly.yml")
	on, err := workflowMap(document, "on")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := on["workflow_run"]; !ok || len(on) != 1 {
		t.Fatalf("nightly triggers = %#v, want only workflow_run", on)
	}
	requirePinnedActions(t, document)
	requireContains(t, source,
		"github.event.workflow_run.conclusion == 'success'",
		"github.event.workflow_run.event == 'push'",
		"github.event.workflow_run.head_branch == 'main'",
		"github.event.workflow_run.head_sha",
		"Reuse completed candidate artifact",
		"this rerun is a no-op",
		"Reuse existing SHA-addressed application images",
		`docker buildx imagetools inspect "$1"`,
		"steps.existing.outputs.app_exists != 'true'",
		"steps.existing.outputs.admin_exists != 'true'",
		"candidate image pair is incomplete after construction",
		"Resolve PostgreSQL candidate",
		"app_image:",
		"admin_image:",
		"postgres_image:",
		"nightly-candidate-${{ github.event.workflow_run.head_sha }}",
		"retention-days: 30",
	)
	if strings.Contains(source, "environment: production") {
		t.Fatal("nightly workflow must not deploy to production")
	}
}

func TestNightlyCandidateDoesNotRequireECRAdministration(t *testing.T) {
	source, _ := repositoryWorkflow(t, ".github/workflows/nightly.yml")
	for _, command := range []string{
		"aws ecr describe-repositories",
		"aws ecr put-image-tag-mutability",
		"aws ecr describe-images",
	} {
		if strings.Contains(source, command) {
			t.Fatalf("nightly candidate uses ECR administration command %q", command)
		}
	}
}

func TestNightlyCandidateConstructionScripts(t *testing.T) {
	_, document := repositoryWorkflow(t, ".github/workflows/nightly.yml")
	reuse := workflowStepRun(t, document, "candidate", "Reuse existing SHA-addressed application images")
	resolveApplications := workflowStepRun(t, document, "candidate", "Resolve immutable image digests")
	resolvePostgreSQL := workflowStepRun(t, document, "candidate", "Resolve PostgreSQL candidate")
	record := workflowStepRun(t, document, "candidate", "Record candidate provenance")

	workDir := t.TempDir()
	bin := filepath.Join(workDir, "bin")
	if err := os.Mkdir(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	writeExecutable(t, filepath.Join(bin, "docker"), `#!/bin/bash
args="$*"
if [ "$1 $2 $3" = "buildx imagetools inspect" ]; then
  target=$4
  if [ "$target" = "$REGISTRY/pai-bot/app:$SHA" ]; then
    if [ "${FAKE_APP_INSPECTION:-present}" = "missing" ]; then echo "manifest unknown" >&2; exit 1; fi
    if [ "${FAKE_APP_INSPECTION:-present}" = "error" ]; then echo "unauthorized" >&2; exit 1; fi
    if [ "${FAKE_INVALID_APPLICATION:-}" = "pai-bot/app" ]; then echo "Digest: invalid"; else echo "Digest: sha256:$FAKE_APP_DIGEST"; fi
    exit 0
  fi
  if [ "$target" = "$REGISTRY/pai-bot/admin:$SHA" ]; then
    if [ "${FAKE_INVALID_APPLICATION:-}" = "pai-bot/admin" ]; then echo "Digest: invalid"; else echo "Digest: sha256:$FAKE_ADMIN_DIGEST"; fi
    exit 0
  fi
  if [ "$target" = "ghcr.io/p-n-ai/pai-postgres:$SHA" ]; then
    if [ "${FAKE_POSTGRES_MODE:-fallback}" = "existing" ] || [ -f "$FAKE_POSTGRES_STATE" ]; then
      echo "Digest: sha256:$FAKE_POSTGRES_DIGEST"
      exit 0
    fi
    exit 1
  fi
  if [ "$target" = "ghcr.io/p-n-ai/pai-postgres:deployed" ] &&
     [ "${FAKE_POSTGRES_MODE:-fallback}" = "fallback" ]; then
    echo "Digest: sha256:$FAKE_POSTGRES_DIGEST"
    exit 0
  fi
  exit 1
fi
if [ "$1 $2 $3" = "buildx imagetools create" ]; then
  echo "$args" >> "$FAKE_POSTGRES_LOG"
  touch "$FAKE_POSTGRES_STATE"
  exit 0
fi
if [ "$1 $2 $3" = "compose config --variables" ]; then
  echo "POSTGRES_IMAGE string ghcr.io/p-n-ai/pai-postgres:base"
  exit 0
fi
exit 1
`)

	sha := strings.Repeat("a", 40)
	appDigest := strings.Repeat("1", 64)
	adminDigest := strings.Repeat("2", 64)
	postgresDigest := strings.Repeat("3", 64)
	commonEnvironment := []string{
		"PATH=" + bin + ":" + os.Getenv("PATH"),
		"REGISTRY=registry.example",
		"SHA=" + sha,
		"FAKE_APP_DIGEST=" + appDigest,
		"FAKE_ADMIN_DIGEST=" + adminDigest,
		"FAKE_POSTGRES_DIGEST=" + postgresDigest,
	}

	reuseOutput := filepath.Join(workDir, "reuse-output")
	if err := runBash(t, reuse, append(commonEnvironment, "GITHUB_OUTPUT="+reuseOutput)...); err != nil {
		t.Fatal(err)
	}
	if output := string(mustReadFile(t, reuseOutput)); output != "app_exists=true\nadmin_exists=true\n" {
		t.Fatalf("existing image outputs = %q", output)
	}
	missingOutput := filepath.Join(workDir, "missing-output")
	if err := runBash(t, reuse, append(commonEnvironment,
		"GITHUB_OUTPUT="+missingOutput,
		"FAKE_APP_INSPECTION=missing",
	)...); err != nil {
		t.Fatal(err)
	}
	if output := string(mustReadFile(t, missingOutput)); output != "app_exists=false\nadmin_exists=true\n" {
		t.Fatalf("missing image outputs = %q", output)
	}
	if err := runBash(t, reuse, append(commonEnvironment,
		"GITHUB_OUTPUT="+filepath.Join(workDir, "error-output"),
		"FAKE_APP_INSPECTION=error",
	)...); err == nil {
		t.Fatal("nightly treated a registry inspection error as a missing image")
	}

	applicationEnvironment := filepath.Join(workDir, "application-environment")
	if err := runBash(
		t,
		resolveApplications,
		append(commonEnvironment, "GITHUB_ENV="+applicationEnvironment)...,
	); err != nil {
		t.Fatal(err)
	}
	requireContains(t, string(mustReadFile(t, applicationEnvironment)),
		"APP_DIGEST=sha256:"+appDigest,
		"ADMIN_DIGEST=sha256:"+adminDigest,
	)
	if err := runBash(
		t,
		resolveApplications,
		append(commonEnvironment,
			"GITHUB_ENV="+filepath.Join(workDir, "invalid-application-environment"),
			"FAKE_INVALID_APPLICATION=pai-bot/app",
		)...,
	); err == nil {
		t.Fatal("nightly accepted an invalid application digest")
	}

	postgresEnvironment := filepath.Join(workDir, "postgres-environment")
	postgresState := filepath.Join(workDir, "postgres-created")
	postgresLog := filepath.Join(workDir, "postgres.log")
	if err := runBash(
		t,
		resolvePostgreSQL,
		append(commonEnvironment,
			"GITHUB_ENV="+postgresEnvironment,
			"FAKE_POSTGRES_STATE="+postgresState,
			"FAKE_POSTGRES_LOG="+postgresLog,
		)...,
	); err != nil {
		t.Fatal(err)
	}
	if output := string(mustReadFile(t, postgresEnvironment)); output !=
		"POSTGRES_IMAGE=ghcr.io/p-n-ai/pai-postgres@sha256:"+postgresDigest+"\n" {
		t.Fatalf("PostgreSQL environment = %q", output)
	}
	requireContains(t, string(mustReadFile(t, postgresLog)),
		"imagetools create --tag ghcr.io/p-n-ai/pai-postgres:"+sha+
			" ghcr.io/p-n-ai/pai-postgres:deployed",
	)
	if err := runBash(
		t,
		resolvePostgreSQL,
		append(commonEnvironment,
			"GITHUB_ENV="+filepath.Join(workDir, "invalid-postgres-environment"),
			"FAKE_POSTGRES_STATE="+filepath.Join(workDir, "missing-postgres-state"),
			"FAKE_POSTGRES_LOG="+postgresLog,
			"FAKE_POSTGRES_MODE=missing",
		)...,
	); err == nil {
		t.Fatal("nightly accepted a missing PostgreSQL source image")
	}

	candidateDir := filepath.Join(workDir, "candidate")
	if err := os.Mkdir(candidateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := runBash(t, "cd \"$TEST_WORKDIR\"\n"+record,
		"TEST_WORKDIR="+candidateDir,
		"GITHUB_REPOSITORY=p-n-ai/pai-bot",
		"GITHUB_RUN_ID=101",
		"REGISTRY=registry.example",
		"SHA="+sha,
		"SOURCE_RUN_ID=202",
		"APP_DIGEST=sha256:"+appDigest,
		"ADMIN_DIGEST=sha256:"+adminDigest,
		"POSTGRES_IMAGE=ghcr.io/p-n-ai/pai-postgres@sha256:"+postgresDigest,
	); err != nil {
		t.Fatal(err)
	}
	var candidate struct {
		Repository    string `json:"repository"`
		SHA           string `json:"sha"`
		NightlyRunID  int64  `json:"nightly_run_id"`
		SourceRunID   int64  `json:"source_run_id"`
		AppImage      string `json:"app_image"`
		AdminImage    string `json:"admin_image"`
		PostgresImage string `json:"postgres_image"`
	}
	if err := json.Unmarshal(mustReadFile(t, filepath.Join(candidateDir, "candidate.json")), &candidate); err != nil {
		t.Fatal(err)
	}
	if candidate.Repository != "p-n-ai/pai-bot" ||
		candidate.SHA != sha ||
		candidate.NightlyRunID != 101 ||
		candidate.SourceRunID != 202 ||
		candidate.AppImage != "registry.example/pai-bot/app@sha256:"+appDigest ||
		candidate.AdminImage != "registry.example/pai-bot/admin@sha256:"+adminDigest ||
		candidate.PostgresImage != "ghcr.io/p-n-ai/pai-postgres@sha256:"+postgresDigest {
		t.Fatalf("candidate provenance = %#v", candidate)
	}
}

func TestNightlyCandidateArtifactReuseIsRetrySafe(t *testing.T) {
	_, document := repositoryWorkflow(t, ".github/workflows/nightly.yml")
	reuse := workflowStepRun(t, document, "candidate", "Reuse completed candidate artifact")
	bin := t.TempDir()
	writeExecutable(t, filepath.Join(bin, "gh"), `#!/bin/bash
expected_path="repos/$GITHUB_REPOSITORY/actions/runs/$GITHUB_RUN_ID/artifacts"
expected_filter="[.artifacts[] | select(.name == \"nightly-candidate-$SHA\" and .expired == false)] | length"
if [ "$#" -ne 4 ] ||
  [ "$1" != "api" ] ||
  [ "$2" != "$expected_path" ] ||
  [ "$3" != "--jq" ] ||
  [ "$4" != "$expected_filter" ]; then
  echo "unexpected gh invocation: $*" >&2
  exit 64
fi
echo "$FAKE_ARTIFACT_COUNT"
`)

	for _, test := range []struct {
		name       string
		count      string
		wantOutput string
		wantError  string
	}{
		{name: "new candidate", count: "0", wantOutput: "exists=false\n"},
		{name: "completed candidate", count: "1", wantOutput: "exists=true\n"},
		{
			name:      "ambiguous candidate",
			count:     "2",
			wantError: "multiple candidate artifacts already exist",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			outputPath := filepath.Join(t.TempDir(), "github-output")
			err := runBash(t, reuse,
				"PATH="+bin+":"+os.Getenv("PATH"),
				"FAKE_ARTIFACT_COUNT="+test.count,
				"GITHUB_OUTPUT="+outputPath,
				"GITHUB_REPOSITORY=p-n-ai/pai-bot",
				"GITHUB_RUN_ID=123",
				"SHA="+strings.Repeat("a", 40),
			)
			if test.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantError) {
					t.Fatalf("run error = %v, want %q", err, test.wantError)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if output := string(mustReadFile(t, outputPath)); output != test.wantOutput {
				t.Fatalf("GITHUB_OUTPUT = %q, want %q", output, test.wantOutput)
			}
		})
	}
}

func TestStablePromotionContracts(t *testing.T) {
	source, document := repositoryWorkflow(t, ".github/workflows/stable.yml")
	deploySource, deployDocument := repositoryWorkflow(t, ".github/workflows/deploy.yml")
	on, err := workflowMap(document, "on")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := on["workflow_dispatch"]; !ok || len(on) != 1 {
		t.Fatalf("stable triggers = %#v, want only workflow_dispatch", on)
	}
	if err := validStableDAG(document); err != nil {
		t.Fatal(err)
	}
	requirePinnedActions(t, document)
	requirePinnedActions(t, deployDocument)
	stableDeploy, err := workflowJob(document, "deploy")
	if err != nil {
		t.Fatal(err)
	}
	stableDeployPermissions, ok := stringMap(stableDeploy["permissions"])
	if !ok {
		t.Fatal("stable deploy job permissions are missing")
	}
	calledDeploy, err := workflowJob(deployDocument, "deploy")
	if err != nil {
		t.Fatal(err)
	}
	calledDeployPermissions, ok := stringMap(calledDeploy["permissions"])
	if !ok {
		t.Fatal("called deploy job permissions are missing")
	}
	for permission, expected := range map[string]string{
		"contents": "read",
		"id-token": "write",
		"packages": "write",
	} {
		if stableDeployPermissions[permission] != expected {
			t.Errorf("stable deploy %s permission = %v, want %s", permission, stableDeployPermissions[permission], expected)
		}
		if calledDeployPermissions[permission] != expected {
			t.Errorf("called deploy %s permission = %v, want %s", permission, calledDeployPermissions[permission], expected)
		}
	}
	requireContains(t, source,
		`^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`,
		`[ "$GITHUB_REF" != "refs/heads/main" ]`,
		".head_branch == \"main\"",
		".head_sha == $sha",
		"nightly-candidate-$sha",
		"postgres_image",
		"uses: ./.github/workflows/deploy.yml",
		"sha: ${{ needs.candidate.outputs.sha }}",
		"app_digest: ${{ needs.candidate.outputs.app_digest }}",
		"admin_digest: ${{ needs.candidate.outputs.admin_digest }}",
		"postgres_image: ${{ needs.candidate.outputs.postgres_image }}",
		`if gh release view "$VERSION"`,
		`gh release create "$VERSION" --verify-tag`,
	)
	requireContains(t, deploySource,
		"aws ecr batch-get-image",
		"environment: production",
		"Generate masked ECR token for server",
		"TAG: ${{ inputs.sha }}",
		"APP_DIGEST: ${{ inputs.app_digest }}",
		"ADMIN_DIGEST: ${{ inputs.admin_digest }}",
		"POSTGRES_IMAGE: ${{ inputs.postgres_image }}",
		"scripts/preflight-conversation-identities.sql",
	)
	if strings.Contains(source+deploySource, "docker build") ||
		strings.Contains(source+deploySource, "docker/build-push-action") {
		t.Fatal("stable promotion must not rebuild candidate images")
	}
	if strings.Contains(deploySource, "ecr-token:") {
		t.Fatal("stable promotion must not transport ECR credentials through job outputs")
	}
	if strings.Contains(deploySource, "ECR_TOKEN: ${{ env.ECR_TOKEN }}") {
		t.Fatal("deploy must inherit the masked ECR token from GITHUB_ENV")
	}
	login := strings.Index(deploySource, "- name: Generate masked ECR token for server")
	deploy := strings.Index(deploySource, "- name: Deploy")
	if login < 0 || deploy < 0 || login > deploy {
		t.Fatal("server registry login must be generated immediately before deployment")
	}

	mutated := strings.Replace(source, "    needs: [candidate, deploy]\n", "    needs: candidate\n", 1)
	mutatedDocument, err := parseWorkflow(mutated)
	if err != nil {
		t.Fatal(err)
	}
	if err := validStableDAG(mutatedDocument); err == nil {
		t.Fatal("stable DAG validator accepted publication without a deploy dependency")
	}
}

func TestStableVersionAndPublicationAreRetrySafe(t *testing.T) {
	_, document := repositoryWorkflow(t, ".github/workflows/stable.yml")
	validate := workflowStepRun(t, document, "candidate", "Validate version and candidate run")
	if err := runBash(t, validate,
		"GITHUB_REF=refs/heads/main",
		"RUN_ID=1",
		"VERSION=v01.2.3",
		"GITHUB_REPOSITORY=p-n-ai/pai-bot",
	); err == nil {
		t.Fatal("validation accepted a semantic version with a leading zero")
	}

	publish := workflowStepRun(t, document, "publish", "Publish semantic tag and GitHub Release")
	bin := t.TempDir()
	logPath := filepath.Join(bin, "gh.log")
	writeExecutable(t, filepath.Join(bin, "gh"), `#!/bin/bash
echo "$*" >> "$FAKE_GH_LOG"
if [ "$1" = "api" ] && [[ "$2" == */git/ref/tags/* ]]; then
  if [ -n "${FAKE_TAG_SHA:-}" ]; then
    echo "$FAKE_TAG_SHA"
    exit 0
  fi
  echo '{"message":"Not Found","status":"404"}'
  exit 1
fi
if [ "$1" = "release" ] && [ "$2" = "view" ]; then
  exit "${FAKE_RELEASE_EXISTS:-1}"
fi
exit 0
`)
	sha := strings.Repeat("a", 40)
	if err := runBash(t, publish,
		"PATH="+bin+":"+os.Getenv("PATH"),
		"FAKE_GH_LOG="+logPath,
		"FAKE_RELEASE_EXISTS=1",
		"GITHUB_REPOSITORY=p-n-ai/pai-bot",
		"SHA="+sha,
		"VERSION=v1.2.3",
	); err != nil {
		t.Fatal(err)
	}
	log := string(mustReadFile(t, logPath))
	requireContains(t, log,
		"api repos/p-n-ai/pai-bot/git/refs --method POST -f ref=refs/tags/v1.2.3 -f sha="+sha,
		"release create v1.2.3 --verify-tag --generate-notes",
	)

	if err := os.WriteFile(logPath, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := runBash(t, publish,
		"PATH="+bin+":"+os.Getenv("PATH"),
		"FAKE_GH_LOG="+logPath,
		"FAKE_TAG_SHA="+sha,
		"FAKE_RELEASE_EXISTS=1",
		"GITHUB_REPOSITORY=p-n-ai/pai-bot",
		"SHA="+sha,
		"VERSION=v1.2.3",
	); err != nil {
		t.Fatal(err)
	}
	log = string(mustReadFile(t, logPath))
	if strings.Contains(log, "--method POST") {
		t.Fatal("retry attempted to recreate an existing matching tag")
	}
	if !strings.Contains(log, "release create v1.2.3 --verify-tag") {
		t.Fatal("retry did not create the missing GitHub Release")
	}

	if err := runBash(t, publish,
		"PATH="+bin+":"+os.Getenv("PATH"),
		"FAKE_GH_LOG="+logPath,
		"FAKE_TAG_SHA="+strings.Repeat("b", 40),
		"GITHUB_REPOSITORY=p-n-ai/pai-bot",
		"SHA="+sha,
		"VERSION=v1.2.3",
	); err == nil {
		t.Fatal("publication accepted an existing tag for another commit")
	}
}

func TestStableCandidateArtifactIsDownloadedIntoItsNamedDirectory(t *testing.T) {
	_, document := repositoryWorkflow(t, ".github/workflows/stable.yml")
	download := workflowStepRun(t, document, "candidate", "Download candidate provenance")
	verify := workflowStepRun(t, document, "candidate", "Verify candidate provenance")
	verify = strings.ReplaceAll(
		verify,
		"${{ secrets.ECR_REGISTRY }}",
		"registry.example",
	)

	workDir := t.TempDir()
	bin := filepath.Join(workDir, "bin")
	if err := os.Mkdir(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	writeExecutable(t, filepath.Join(bin, "git"), "#!/bin/bash\nexit 0\n")
	writeExecutable(t, filepath.Join(bin, "gh"), `#!/bin/bash
if [ "$1" = "api" ] && [[ "$2" == *"/artifacts?per_page=100" ]]; then
  case "${FAKE_ARTIFACT_MODE:-single}" in
    zero) printf '{"artifacts":[]}\n' ;;
    expired) printf '{"artifacts":[{"name":"nightly-candidate-%s","expired":true}]}\n' "$FAKE_CANDIDATE_SHA" ;;
    multiple) printf '{"artifacts":[{"name":"nightly-candidate-%s","expired":false},{"name":"nightly-candidate-%s","expired":false}]}\n' "$FAKE_CANDIDATE_SHA" "$FAKE_SECOND_SHA" ;;
    *) printf '{"artifacts":[{"name":"nightly-candidate-%s","expired":false}]}\n' "$FAKE_CANDIDATE_SHA" ;;
  esac
  exit 0
fi
if [ "$1" = "api" ] && [[ "$2" == *"/actions/runs/22" ]]; then
  printf '{"name":"CI","event":"push","head_branch":"main","head_sha":"%s","conclusion":"success"}\n' "$FAKE_CANDIDATE_SHA"
  exit 0
fi
if [ "$1" = "run" ] && [ "$2" = "download" ]; then
  while [ "$#" -gt 0 ]; do
    case "$1" in
      --dir)
        artifact_dir=$2
        shift 2
        ;;
      *)
        shift
        ;;
    esac
  done
  mkdir -p "$artifact_dir"
  printf '{"repository":"p-n-ai/pai-bot","nightly_run_id":11,"source_run_id":22,"sha":"%s","app_image":"registry.example/pai-bot/app@sha256:%s","admin_image":"registry.example/pai-bot/admin@sha256:%s","postgres_image":"ghcr.io/p-n-ai/pai-postgres@sha256:%s"}\n' \
    "$FAKE_CANDIDATE_SHA" "$FAKE_APP_DIGEST" "$FAKE_ADMIN_DIGEST" "$FAKE_POSTGRES_DIGEST" \
    > "$artifact_dir/candidate.json"
  exit 0
fi
exit 1
`)

	sha := strings.Repeat("a", 40)
	appDigest := strings.Repeat("1", 64)
	adminDigest := strings.Repeat("2", 64)
	postgresDigest := strings.Repeat("3", 64)
	downloadOutput := filepath.Join(workDir, "download-output")
	commonEnvironment := []string{
		"PATH=" + bin + ":" + os.Getenv("PATH"),
		"FAKE_CANDIDATE_SHA=" + sha,
		"FAKE_SECOND_SHA=" + strings.Repeat("b", 40),
		"FAKE_APP_DIGEST=" + appDigest,
		"FAKE_ADMIN_DIGEST=" + adminDigest,
		"FAKE_POSTGRES_DIGEST=" + postgresDigest,
		"GITHUB_REPOSITORY=p-n-ai/pai-bot",
		"RUN_ID=11",
	}
	if err := runBash(
		t,
		"cd \"$TEST_WORKDIR\"\n"+download,
		append(commonEnvironment,
			"TEST_WORKDIR="+workDir,
			"GITHUB_OUTPUT="+downloadOutput,
		)...,
	); err != nil {
		t.Fatal(err)
	}
	artifactName := "nightly-candidate-" + sha
	if output := string(mustReadFile(t, downloadOutput)); output != "artifact_name="+artifactName+"\n" {
		t.Fatalf("download GITHUB_OUTPUT = %q", output)
	}
	for _, mode := range []string{"zero", "expired", "multiple"} {
		t.Run("reject_"+mode+"_artifacts", func(t *testing.T) {
			if err := runBash(
				t,
				"cd \"$TEST_WORKDIR\"\n"+download,
				append(commonEnvironment,
					"TEST_WORKDIR="+workDir,
					"GITHUB_OUTPUT="+filepath.Join(workDir, mode+"-output"),
					"FAKE_ARTIFACT_MODE="+mode,
				)...,
			); err == nil {
				t.Fatalf("download accepted %s candidate artifacts", mode)
			}
		})
	}

	verifyOutput := filepath.Join(workDir, "verify-output")
	if err := runBash(
		t,
		"cd \"$TEST_WORKDIR\"\n"+verify,
		append(commonEnvironment,
			"TEST_WORKDIR="+workDir,
			"ARTIFACT_NAME="+artifactName,
			"VERSION=v1.2.3",
			"GITHUB_OUTPUT="+verifyOutput,
		)...,
	); err != nil {
		t.Fatal(err)
	}
	requireContains(t, string(mustReadFile(t, verifyOutput)),
		"sha="+sha,
		"app_digest=sha256:"+appDigest,
		"admin_digest=sha256:"+adminDigest,
		"postgres_image=ghcr.io/p-n-ai/pai-postgres@sha256:"+postgresDigest,
	)

	manifestPath := filepath.Join(workDir, "candidate", artifactName, "candidate.json")
	validManifest := string(mustReadFile(t, manifestPath))
	for _, test := range []struct {
		name string
		from string
		to   string
	}{
		{
			name: "repository",
			from: `"repository":"p-n-ai/pai-bot"`,
			to:   `"repository":"other/repository"`,
		},
		{
			name: "nightly run",
			from: `"nightly_run_id":11`,
			to:   `"nightly_run_id":12`,
		},
		{
			name: "artifact SHA",
			from: `"sha":"` + sha + `"`,
			to:   `"sha":"` + strings.Repeat("b", 40) + `"`,
		},
		{
			name: "source CI",
			from: `"source_run_id":22`,
			to:   `"source_run_id":23`,
		},
		{
			name: "app digest",
			from: `"app_image":"registry.example/pai-bot/app@sha256:` + appDigest + `"`,
			to:   `"app_image":"registry.example/pai-bot/app@latest"`,
		},
		{
			name: "admin digest",
			from: `"admin_image":"registry.example/pai-bot/admin@sha256:` + adminDigest + `"`,
			to:   `"admin_image":"registry.example/pai-bot/admin@latest"`,
		},
		{
			name: "PostgreSQL digest",
			from: `"postgres_image":"ghcr.io/p-n-ai/pai-postgres@sha256:` + postgresDigest + `"`,
			to:   `"postgres_image":"ghcr.io/p-n-ai/pai-postgres:latest"`,
		},
	} {
		t.Run("reject_"+test.name, func(t *testing.T) {
			invalidManifest := strings.Replace(validManifest, test.from, test.to, 1)
			if invalidManifest == validManifest {
				t.Fatalf("test mutation %q did not change manifest", test.name)
			}
			if err := os.WriteFile(manifestPath, []byte(invalidManifest), 0o600); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := os.WriteFile(manifestPath, []byte(validManifest), 0o600); err != nil {
					t.Error(err)
				}
			})
			if err := runBash(
				t,
				"cd \"$TEST_WORKDIR\"\n"+verify,
				append(commonEnvironment,
					"TEST_WORKDIR="+workDir,
					"ARTIFACT_NAME="+artifactName,
					"VERSION=v1.2.3",
					"GITHUB_OUTPUT="+filepath.Join(workDir, "invalid-"+strings.ReplaceAll(test.name, " ", "-")),
				)...,
			); err == nil {
				t.Fatalf("verification accepted invalid %s", test.name)
			}
			if err := os.WriteFile(manifestPath, []byte(validManifest), 0o600); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

type fakeDeployOptions struct {
	appHealth            string
	smokeFails           bool
	gooseFails           bool
	adminHTTPFails       bool
	adminHTTPFailures    int
	aiStatusFails        bool
	appLookupFails       bool
	missingPreviousAdmin bool
}

func runFakeDeploy(
	t *testing.T,
	options fakeDeployOptions,
) (string, string, error) {
	t.Helper()
	deployDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(deployDir, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(deployDir, ".env"),
		[]byte("LEARN_DATABASE_URL=postgres://pai:test@postgres:5432/pai\nPAI_TEST_RELEASE=candidate\n"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(deployDir, ".env.rollback"),
		[]byte("LEARN_DATABASE_URL=postgres://pai:test@postgres:5432/pai\nPAI_TEST_RELEASE=previous\n"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(deployDir, "scripts", "preflight-conversation-identities.sql"),
		[]byte("SELECT 1;\n"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}

	bin := filepath.Join(deployDir, "bin")
	if err := os.Mkdir(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(deployDir, "docker.log")
	stateDir := filepath.Join(deployDir, "state")
	if err := os.Mkdir(stateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeExecutable(t, filepath.Join(bin, "sudo"), "#!/bin/bash\nexit 0\n")
	writeExecutable(t, filepath.Join(bin, "sleep"), "#!/bin/bash\nexit 0\n")
	writeExecutable(t, filepath.Join(bin, "curl"), "#!/bin/bash\nexit 0\n")
	writeExecutable(t, filepath.Join(bin, "docker"), `#!/bin/bash
echo "$*" >> "$FAKE_DOCKER_LOG"
args="$*"
if [ "$1" = "compose" ] && [[ "$args" == *" ps -q app"* ]]; then
  count_file="$FAKE_DOCKER_STATE/app-ps"
  count=$(cat "$count_file" 2>/dev/null || echo 0)
  count=$((count + 1))
  echo "$count" > "$count_file"
  if [ "$count" -gt 1 ] && [ "$FAKE_APP_LOOKUP_FAIL" = "true" ]; then exit 9; fi
  if [ "$count" -eq 1 ]; then echo old-app; else echo new-app; fi
  exit 0
fi
if [ "$1" = "compose" ] && [[ "$args" == *" ps -q admin"* ]]; then
  count_file="$FAKE_DOCKER_STATE/admin-ps"
  count=$(cat "$count_file" 2>/dev/null || echo 0)
  count=$((count + 1))
  echo "$count" > "$count_file"
  if [ "$count" -eq 1 ]; then
    if [ "$FAKE_MISSING_PREVIOUS_ADMIN" != "true" ]; then echo old-admin; fi
  else
    echo new-admin
  fi
  exit 0
fi
if [ "$1" = "inspect" ]; then
  target="${@: -1}"
  if [[ "$args" == *".State.Health"* ]]; then echo "$FAKE_APP_HEALTH"; exit 0; fi
  if [[ "$args" == *".State.Status"* ]]; then echo running; exit 0; fi
  case "$target" in
    old-app) echo sha256:oldapp ;;
    old-admin) echo sha256:oldadmin ;;
    new-app) echo sha256:newapp ;;
    new-admin) echo sha256:newadmin ;;
  esac
  exit 0
fi
if [ "$1" = "image" ] && [ "$2" = "inspect" ]; then
  case "${@: -1}" in
    *"/app@"*) echo sha256:newapp ;;
    *"/admin@"*) echo sha256:newadmin ;;
    ghcr.io/p-n-ai/pai-postgres@*) echo sha256:newpostgres ;;
  esac
  exit 0
fi
if [ "$1" = "compose" ] && [[ "$args" == *"pg_dump"* ]]; then
  echo fake-backup
  exit 0
fi
if [ "$1" = "compose" ] && [[ "$args" == *"pg_restore --list"* ]]; then
  exit 0
fi
if [ "$1" = "compose" ] && [[ "$args" == *"config --variables"* ]]; then
  echo "POSTGRES_IMAGE string ghcr.io/p-n-ai/pai-postgres:pggraph-1.0.0-pgvector-0.8.5"
  exit 0
fi
if [ "$1" = "compose" ] && [[ "$args" == *"exec -T postgres psql"*"-Atqc"* ]]; then
  echo f
  exit 0
fi
if [ "$1" = "compose" ] && [[ "$args" == *"--profile tools run --rm goose"* ]]; then
  if [ "$FAKE_GOOSE_FAIL" = "true" ]; then exit 7; fi
  exit 0
fi
if [ "$1" = "compose" ] && [[ "$args" == *"exec -T admin wget"* ]]; then
  count_file="$FAKE_DOCKER_STATE/admin-http"
  count=$(cat "$count_file" 2>/dev/null || echo 0)
  count=$((count + 1))
  echo "$count" > "$count_file"
  if [ "$FAKE_ADMIN_HTTP_FAIL" = "true" ] || [ "$count" -le "$FAKE_ADMIN_HTTP_FAILURES" ]; then exit 8; fi
  exit 0
fi
if [ "$1" = "compose" ] && [[ "$args" == *"http://localhost:8080/health/status"* ]]; then
  if [ "$FAKE_AI_STATUS_FAIL" = "true" ]; then
    echo '{"status":"degraded","components":[{"id":"application","status":"operational"},{"id":"ai_provider","status":"unavailable"}]}'
  else
    echo '{"status":"ok","components":[{"id":"application","status":"operational"},{"id":"ai_provider","status":"operational"}]}'
  fi
  exit 0
fi
if [ "$1" = "compose" ] && [[ "$args" == *"up -d --force-recreate app admin"* ]]; then
  grep '^PAI_TEST_RELEASE=' .env >> "$FAKE_DOCKER_LOG"
  exit 0
fi
if [ "$1" = "compose" ] && [[ "$args" == *"/pai-terminal-chat"* ]]; then
  if [ "$FAKE_SMOKE_FAIL" = "true" ] && [[ "$args" == *"/foobar"* ]]; then
    echo "unexpected"
    exit 0
  fi
  case "$args" in
    *"/progress"*) echo "Progress XP" ;;
    *"/help"*) echo "available commands" ;;
    *"/foobar"*) echo "Unknown" ;;
    *) echo "/learn" ;;
  esac
  exit 0
fi
exit 0
`)

	script, err := filepath.Abs(filepath.Join("..", "..", "scripts", "deploy-remote.sh"))
	if err != nil {
		t.Fatal(err)
	}
	command := exec.Command("bash", script)
	command.Env = append(os.Environ(),
		"PATH="+bin+":"+os.Getenv("PATH"),
		"DEPLOY_DIR="+deployDir,
		"ECR_TOKEN=test-token",
		"GHCR_TOKEN=test-ghcr-token",
		"GHCR_USER=test-user",
		"POSTGRES_IMAGE=ghcr.io/p-n-ai/pai-postgres@sha256:"+strings.Repeat("3", 64),
		"REGISTRY=registry.example",
		"TAG="+strings.Repeat("a", 40),
		"APP_DIGEST=sha256:"+strings.Repeat("1", 64),
		"ADMIN_DIGEST=sha256:"+strings.Repeat("2", 64),
		"FAKE_APP_HEALTH="+options.appHealth,
		fmt.Sprintf("FAKE_SMOKE_FAIL=%t", options.smokeFails),
		fmt.Sprintf("FAKE_GOOSE_FAIL=%t", options.gooseFails),
		fmt.Sprintf("FAKE_ADMIN_HTTP_FAIL=%t", options.adminHTTPFails),
		fmt.Sprintf("FAKE_ADMIN_HTTP_FAILURES=%d", options.adminHTTPFailures),
		fmt.Sprintf("FAKE_AI_STATUS_FAIL=%t", options.aiStatusFails),
		fmt.Sprintf("FAKE_APP_LOOKUP_FAIL=%t", options.appLookupFails),
		fmt.Sprintf("FAKE_MISSING_PREVIOUS_ADMIN=%t", options.missingPreviousAdmin),
		"FAKE_DOCKER_LOG="+logPath,
		"FAKE_DOCKER_STATE="+stateDir,
	)
	output, runErr := command.CombinedOutput()
	log := string(mustReadFile(t, logPath))
	return string(output), log, runErr
}

func TestRemoteDeploymentFreshInstallAndRollbackBehavior(t *testing.T) {
	output, log, err := runFakeDeploy(t, fakeDeployOptions{appHealth: "healthy"})
	if err != nil {
		t.Fatalf("healthy deploy failed: %v\n%s", err, output)
	}
	if strings.Contains(log, "psql postgres://pai:test@postgres:5432/pai -f -") {
		t.Fatal("fresh install ran the standalone identity preflight before migrations")
	}
	requireContains(t, output,
		"Fresh database: identity preflight will be enforced by the migration itself",
		"Admin container is running the candidate image",
		"Deploy successful",
	)

	output, log, err = runFakeDeploy(t, fakeDeployOptions{appHealth: "unhealthy"})
	if err == nil {
		t.Fatal("unhealthy candidate deployment succeeded")
	}
	requireContains(t, output, "Rollback restored app sha256:oldapp and admin sha256:oldadmin")
	requireContains(t, log,
		"tag sha256:oldapp pai-bot:latest",
		"tag sha256:oldadmin pai-admin:latest",
		"up -d --force-recreate app admin",
		"PAI_TEST_RELEASE=previous",
	)

	output, log, err = runFakeDeploy(t, fakeDeployOptions{
		appHealth:  "healthy",
		smokeFails: true,
	})
	if err == nil {
		t.Fatal("deployment with a failed bot smoke test succeeded")
	}
	requireContains(t, output, "bot smoke test(s) failed")
	requireContains(t, log,
		"tag sha256:oldapp pai-bot:latest",
		"tag sha256:oldadmin pai-admin:latest",
	)

	output, log, err = runFakeDeploy(t, fakeDeployOptions{
		appHealth:     "healthy",
		aiStatusFails: true,
	})
	if err == nil {
		t.Fatal("deployment with an unavailable AI provider succeeded")
	}
	requireContains(t, output,
		"AI response health check failed",
		"Rollback restored app sha256:oldapp and admin sha256:oldadmin",
	)
	requireContains(t, log,
		"tag sha256:oldapp pai-bot:latest",
		"tag sha256:oldadmin pai-admin:latest",
	)

	output, log, err = runFakeDeploy(t, fakeDeployOptions{
		appHealth:      "healthy",
		appLookupFails: true,
	})
	if err == nil {
		t.Fatal("deployment succeeded after an app container lookup failed")
	}
	requireContains(t, output,
		"deployment command failed with status 9",
		"Rollback restored app sha256:oldapp and admin sha256:oldadmin",
	)
	if rollbackCount := strings.Count(log, "up -d --force-recreate app admin"); rollbackCount != 1 {
		t.Fatalf("rollback restarted app/admin %d times, want exactly once", rollbackCount)
	}

	output, log, err = runFakeDeploy(t, fakeDeployOptions{
		appHealth:  "healthy",
		gooseFails: true,
	})
	if err == nil {
		t.Fatal("deployment succeeded after a migration command failed")
	}
	requireContains(t, output,
		"deployment command failed with status 7",
		"Rollback restored app sha256:oldapp and admin sha256:oldadmin",
	)
	requireContains(t, log,
		"tag sha256:oldapp pai-bot:latest",
		"tag sha256:oldadmin pai-admin:latest",
	)

	output, log, err = runFakeDeploy(t, fakeDeployOptions{
		appHealth:      "healthy",
		adminHTTPFails: true,
	})
	if err == nil {
		t.Fatal("deployment with a failed admin HTTP check succeeded")
	}
	requireContains(t, output,
		"admin HTTP check failed",
		"Rollback restored app sha256:oldapp and admin sha256:oldadmin",
	)
	requireContains(t, log,
		"tag sha256:oldapp pai-bot:latest",
		"tag sha256:oldadmin pai-admin:latest",
	)

	output, log, err = runFakeDeploy(t, fakeDeployOptions{
		appHealth:            "healthy",
		adminHTTPFails:       true,
		missingPreviousAdmin: true,
	})
	if err == nil {
		t.Fatal("deployment with no complete rollback pair succeeded")
	}
	requireContains(t, output,
		"admin HTTP check failed",
		"Rollback unavailable: no complete previous application image pair",
	)
	if strings.Contains(log, "up -d --force-recreate") {
		t.Fatal("rollback restarted services without a complete previous image pair")
	}
}

func TestRemoteDeploymentRetriesAdminHTTPReadiness(t *testing.T) {
	output, log, err := runFakeDeploy(t, fakeDeployOptions{
		appHealth:         "healthy",
		adminHTTPFailures: 2,
	})
	if err != nil {
		t.Fatalf("deployment after transient admin HTTP failures: %v\n%s", err, output)
	}
	requireContains(t, output, "Admin HTTP ready after attempt 3", "Deploy successful")
	if attempts := strings.Count(log, "exec -T admin wget"); attempts != 3 {
		t.Fatalf("admin HTTP attempts = %d, want 3", attempts)
	}
}

func TestRemoteDeploymentContracts(t *testing.T) {
	source := repositorySource(t, "scripts/deploy-remote.sh")
	requireContains(t, source,
		"PREV_APP_ID=",
		"PREV_ADMIN_ID=",
		"trap unexpected_failure ERR",
		`ROLLBACK_ARMED=false`,
		`run --rm config-check`,
		`pg_dump "$DB_URL" --format=custom`,
		`docker compose -f docker-compose.yml -f docker-compose.prod.yml pull postgres dragonfly`,
		`docker image prune -f || echo "WARNING: post-deploy image cleanup failed"`,
		"--format='{{.Image}}'",
		`docker tag "$PREV_APP_ID" pai-bot:latest`,
		`docker tag "$PREV_ADMIN_ID" pai-admin:latest`,
		`SELECT to_regclass('public.users') IS NOT NULL`,
		`RUNNING_APP_ID=$(docker inspect --format '{{.Image}}' "$APP_CONTAINER")`,
		`RUNNING_ADMIN_ID=$(docker inspect --format '{{.Image}}' "$ADMIN_CONTAINER")`,
		`http://127.0.0.1:3000/`,
		`if [ "$RUNNING_APP_ID" != "$EXPECTED_APP_ID" ]`,
		`if [ "$RUNNING_ADMIN_ID" != "$EXPECTED_ADMIN_ID" ]`,
		`fail_release "$SMOKE_FAIL bot smoke test(s) failed"`,
		`smoke "/help"`,
	)
	if strings.Contains(source, `smoke "/create_group"`) {
		t.Fatal("production smoke tests must not mutate application state")
	}
	if strings.Contains(source, ".Config.Image") {
		t.Fatal("rollback must not capture a mutable image reference")
	}
	preflight := strings.Index(source, "scripts/preflight-conversation-identities.sql")
	migration := strings.Index(source, "go run github.com/pressly/goose")
	if preflight < 0 || migration < 0 || preflight > migration {
		t.Fatal("identity preflight must guard the conversation migration")
	}
}

func TestRiskTemplatesCaptureReleaseBoundaries(t *testing.T) {
	pullRequest := repositorySource(t, ".github/PULL_REQUEST_TEMPLATE.md")
	issue := repositorySource(t, ".github/ISSUE_TEMPLATE/change.yml")
	for _, source := range []string{pullRequest, issue} {
		requireContains(t, strings.ToLower(source), "api", "migration", "release", "rollback")
	}
}
