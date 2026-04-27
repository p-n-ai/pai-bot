---
summary: How Open School Syllabus (OSS) feeds pai-bot curriculum, quiz, retrieval, progress, and admin surfaces.
read_when:
  - You are changing curriculum loading, topic matching, quiz assessment flow, retrieval seeding, or OSS submodule wiring.
  - You need to explain what belongs in p-n-ai/oss versus this repo.
---

# OSS Curriculum Relationship

## Purpose

Open School Syllabus (OSS) is the curriculum source. This repo is the product runtime.

In practice:

- `p-n-ai/oss` owns structured syllabus content: syllabi, subjects, topics, teaching notes, prerequisites, and assessments.
- `pai-bot` owns how students, teachers, and admins experience that content through chat, quizzes, progress tracking, retrieval, and the admin panel.

Keep curriculum facts in OSS. Keep product behavior in this repo.

## Current Wiring

OSS is registered as a Git submodule at `oss/`:

```text
.gitmodules -> path = oss, url = https://github.com/p-n-ai/oss.git
```

The default curriculum path is `./oss`, configurable through:

```bash
LEARN_CURRICULUM_PATH=/path/to/oss
```

Current checkout note: if `git submodule status` shows a leading `-`, the submodule is not initialized and `oss/` may be empty. Initialize it before running curriculum-backed flows:

```bash
git submodule update --init oss
```

To move this repo to the latest OSS commit from the configured remote branch:

```bash
git submodule update --remote oss
```

That updates the checked-out submodule content; commit the changed submodule pointer in this repo when the product should pin that OSS revision.

## Loader Contract

`internal/curriculum/loader.go` walks the configured curriculum directory at startup and loads supported files into memory:

| OSS file pattern | pai-bot use |
| --- | --- |
| `syllabus.yaml` / `syllabus.yml` | Top-level syllabus metadata |
| `subject.yaml` / `subject.yml` | Subject metadata and topic list |
| `*.yaml` / `*.yml` | Topic cards, except known non-topic files |
| `*.teaching.md` | Teaching notes paired with a same-name topic YAML file |
| `*.assessments.yaml` / `*.assessments.yml` | Quiz assessment questions for one topic |

Loaded data maps into `internal/curriculum/types.go`:

- `Syllabus`
- `Subject`
- `Topic`
- `Assessment`
- `AssessmentQuestion`

The server currently treats curriculum load failure as non-fatal: it logs a warning and keeps the app alive. That lets non-curriculum routes work, but curriculum context, quiz assessment lookup, and curriculum retrieval will be degraded or unavailable.

## Runtime Flow

At boot, `cmd/server/main.go` creates `curriculum.NewLoader(cfg.CurriculumPath)`.

If the loader succeeds, this repo uses OSS content in four main places:

1. Chat teaching context

   `internal/agent` resolves a user message to a curriculum topic, fetches teaching notes, and injects topic context into the tutor response path.

2. Quiz mode

   `internal/agent/quiz_router.go` starts and runs quizzes from `*.assessments.yaml` questions. Current quiz grading is deterministic for OSS-backed answers.

3. Retrieval seed

   `internal/retrieval/curriculum_seed.go` turns loaded curriculum into retrieval records:

   - source: `source:curriculum`
   - collections: `curriculum:<subject-or-syllabus-id>`
   - documents: `topic:<topic-id>`, `note:<topic-id>:<section>`, `assessment:<topic-id>:<index>`

   This is why curriculum is now one retrieval source type, not the whole retrieval architecture.

4. Progress and motivation

   Progress, goals, topic unlocks, challenges, and mastery records use OSS identifiers such as `syllabus_id` and `topic_id` to stay tied to curriculum content.

## Boundary

Change OSS when the work is about curriculum truth:

- official syllabus references
- topic names and learning objectives
- prerequisite relationships
- teaching notes
- assessment questions, rubrics, hints, and distractors
- content provenance or quality level

Change this repo when the work is about product behavior:

- how topics are matched from student messages
- how prompts use teaching notes
- quiz routing, grading behavior, XP, streaks, or state handling
- retrieval indexing/search behavior
- admin APIs and UI around curriculum
- database schema for progress, goals, or challenges

If a product behavior depends on a new curriculum field, update OSS first or in the same branch/PR chain, then update `internal/curriculum/types.go`, loader tests, and the consuming feature here.

## Current Versus Planned

Current:

- OSS is wired as a submodule at `oss/`.
- `LEARN_CURRICULUM_PATH` controls where pai-bot loads curriculum from.
- The loader reads YAML topics, subjects, syllabi, teaching notes, and assessment files from the filesystem.
- Chat context, quiz mode, retrieval seeding, progress, goals, topic unlocks, and challenges can use loaded OSS identifiers/content.
- The submodule may be uninitialized in a fresh checkout; initialize it explicitly.

Planned or incomplete:

- Hot reload is described in older architecture docs but is not implemented in the current loader.
- Dragonfly-backed curriculum cache is described in older architecture docs but the current loader is in-memory.
- "Add curriculum in OSS and pai-bot automatically picks it up" is only true after the deployed repo updates its `oss` submodule pointer or points `LEARN_CURRICULUM_PATH` at the new content.
- Dynamic AI question generation from sparse OSS assessments is planned; current quiz mode primarily uses OSS-backed assessment YAML.

## Safe Change Checklist

Before changing curriculum integration:

1. Check current repo state:

   ```bash
   git status --short --branch
   git submodule status
   ```

2. Initialize OSS if needed:

   ```bash
   git submodule update --init oss
   ```

3. Inspect the relevant OSS files and this repo's loader/types before editing contracts.

4. Add or update tests for loader, topic matching, quiz assessment behavior, or retrieval seeding.

5. Run the repo gate before handoff:

   ```bash
   just test-all
   ```
