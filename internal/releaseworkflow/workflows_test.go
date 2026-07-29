// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package releaseworkflow

import (
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
		"admin-spa",
		"admin-spa-e2e",
		"go",
		"go-lint",
		"docker",
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
		"Require immutable candidate repositories",
		`[ "$mutability" != "IMMUTABLE" ]`,
		"Reuse existing SHA-addressed application images",
		"steps.existing.outputs.app_exists != 'true'",
		"steps.existing.outputs.admin_exists != 'true'",
		"candidate image pair is incomplete after construction",
		"app_image:",
		"admin_image:",
		"nightly-candidate-${{ github.event.workflow_run.head_sha }}",
		"retention-days: 30",
	)
	if strings.Contains(source, "environment: production") {
		t.Fatal("nightly workflow must not deploy to production")
	}
}

func TestStablePromotionContracts(t *testing.T) {
	source, document := repositoryWorkflow(t, ".github/workflows/stable.yml")
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
	requireContains(t, source,
		`^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`,
		".head_branch == \"main\"",
		".head_sha == $sha",
		"nightly-candidate-$sha",
		"aws ecr batch-get-image",
		"Generate server registry login",
		"environment: production",
		"TAG: ${{ needs.candidate.outputs.sha }}",
		"APP_DIGEST: ${{ needs.candidate.outputs.app_digest }}",
		"ADMIN_DIGEST: ${{ needs.candidate.outputs.admin_digest }}",
		"scripts/preflight-conversation-identities.sql",
		`if gh release view "$VERSION"`,
		`gh release create "$VERSION" --verify-tag`,
	)
	if strings.Contains(source, "docker build") ||
		strings.Contains(source, "docker/build-push-action") {
		t.Fatal("stable promotion must not rebuild candidate images")
	}
	if strings.Contains(source, "ecr-token:") {
		t.Fatal("stable promotion must not transport ECR credentials through job outputs")
	}
	if strings.Contains(source, "ECR_TOKEN: ${{ env.ECR_TOKEN }}") {
		t.Fatal("deploy must inherit the masked ECR token from GITHUB_ENV")
	}
	login := strings.Index(source, "- name: Generate server registry login")
	deploy := strings.Index(source, "- name: Deploy")
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
		"FAKE_TAG_SHA="+sha,
		"FAKE_RELEASE_EXISTS=1",
		"GITHUB_REPOSITORY=p-n-ai/pai-bot",
		"SHA="+sha,
		"VERSION=v1.2.3",
	); err != nil {
		t.Fatal(err)
	}
	log := string(mustReadFile(t, logPath))
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

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func runFakeDeploy(
	t *testing.T,
	appHealth string,
	smokeFails bool,
	gooseFails bool,
) (string, string, error) {
	t.Helper()
	deployDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(deployDir, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(deployDir, ".env"),
		[]byte("LEARN_DATABASE_URL=postgres://pai:test@postgres:5432/pai\n"),
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
  if [ "$count" -eq 1 ]; then echo old-app; else echo new-app; fi
  exit 0
fi
if [ "$1" = "compose" ] && [[ "$args" == *" ps -q admin"* ]]; then
  count_file="$FAKE_DOCKER_STATE/admin-ps"
  count=$(cat "$count_file" 2>/dev/null || echo 0)
  count=$((count + 1))
  echo "$count" > "$count_file"
  if [ "$count" -eq 1 ]; then echo old-admin; else echo new-admin; fi
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
  esac
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
if [ "$1" = "compose" ] && [[ "$args" == *"/pai-terminal-chat"* ]]; then
  if [ "$FAKE_SMOKE_FAIL" = "true" ] && [[ "$args" == *"/foobar"* ]]; then
    echo "unexpected"
    exit 0
  fi
  case "$args" in
    *"/progress"*) echo "Progress XP" ;;
    *"/create_group"*) echo "Test Deploy" ;;
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
		"REGISTRY=registry.example",
		"TAG="+strings.Repeat("a", 40),
		"APP_DIGEST=sha256:"+strings.Repeat("1", 64),
		"ADMIN_DIGEST=sha256:"+strings.Repeat("2", 64),
		"FAKE_APP_HEALTH="+appHealth,
		fmt.Sprintf("FAKE_SMOKE_FAIL=%t", smokeFails),
		fmt.Sprintf("FAKE_GOOSE_FAIL=%t", gooseFails),
		"FAKE_DOCKER_LOG="+logPath,
		"FAKE_DOCKER_STATE="+stateDir,
	)
	output, runErr := command.CombinedOutput()
	log := string(mustReadFile(t, logPath))
	return string(output), log, runErr
}

func TestRemoteDeploymentFreshInstallAndRollbackBehavior(t *testing.T) {
	output, log, err := runFakeDeploy(t, "healthy", false, false)
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

	output, log, err = runFakeDeploy(t, "unhealthy", false, false)
	if err == nil {
		t.Fatal("unhealthy candidate deployment succeeded")
	}
	requireContains(t, output, "Rollback restored app sha256:oldapp and admin sha256:oldadmin")
	requireContains(t, log,
		"tag sha256:oldapp pai-bot:latest",
		"tag sha256:oldadmin pai-admin:latest",
		"up -d --force-recreate app admin",
	)

	output, log, err = runFakeDeploy(t, "healthy", true, false)
	if err == nil {
		t.Fatal("deployment with a failed bot smoke test succeeded")
	}
	requireContains(t, output, "bot smoke test(s) failed")
	requireContains(t, log,
		"tag sha256:oldapp pai-bot:latest",
		"tag sha256:oldadmin pai-admin:latest",
	)

	output, log, err = runFakeDeploy(t, "healthy", false, true)
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
}

func TestRemoteDeploymentContracts(t *testing.T) {
	source := repositorySource(t, "scripts/deploy-remote.sh")
	requireContains(t, source,
		"PREV_APP_ID=",
		"PREV_ADMIN_ID=",
		"trap unexpected_failure ERR",
		`ROLLBACK_ARMED=false`,
		`docker image prune -f || echo "WARNING: post-deploy image cleanup failed"`,
		"--format='{{.Image}}'",
		`docker tag "$PREV_APP_ID" pai-bot:latest`,
		`docker tag "$PREV_ADMIN_ID" pai-admin:latest`,
		`SELECT to_regclass('public.users') IS NOT NULL`,
		`RUNNING_APP_ID=$(docker inspect --format '{{.Image}}' "$APP_CONTAINER")`,
		`RUNNING_ADMIN_ID=$(docker inspect --format '{{.Image}}' "$ADMIN_CONTAINER")`,
		`if [ "$RUNNING_APP_ID" != "$EXPECTED_APP_ID" ]`,
		`if [ "$RUNNING_ADMIN_ID" != "$EXPECTED_ADMIN_ID" ]`,
		`fail_release "$SMOKE_FAIL bot smoke test(s) failed"`,
	)
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
