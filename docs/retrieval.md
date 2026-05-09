---
title: "GitHub To Postgres Curriculum Retrieval"
summary: >-
  Fetch OSS-shaped curriculum from GitHub, store it in Postgres, and retrieve it
  without the local oss submodule.
read_when:
  - You are implementing or reviewing the curriculum retrieval prototype.
  - You need the GitHub-to-Postgres data flow, schema contract, or verification steps.
---

# GitHub To Postgres Curriculum Retrieval

## Objective

Document the prototype path for using curriculum content without depending on a
local `oss/` submodule at runtime.

The prototype fetches curriculum from GitHub during ingest. Retrieval then
reads from Postgres:

```text
GitHub repo URL
-> temp zip download
-> safe extraction
-> OSS curriculum parsing
-> Postgres content rows
-> Postgres full-text search
```

After ingest, retrieval reads Postgres rows only. It does not read GitHub, temp
files, `oss/`, or `LEARN_CURRICULUM_PATH`.

## Command Contract

Ingest from GitHub:

```sh
DATABASE_URL=postgres://... \
CURRICULUM_SOURCE=https://github.com/p-n-ai/oss \
go run ./cmd/curriculum-knowledge-prototype ingest --query "linear equations"
```

Search stored content:

```sh
DATABASE_URL=postgres://... \
go run ./cmd/curriculum-knowledge-prototype search --query "linear equations"
```

Drop prototype state:

```sh
DATABASE_URL=postgres://... \
go run ./cmd/curriculum-knowledge-prototype drop
```

Inputs:

- `DATABASE_URL`: Postgres connection string.
- `CURRICULUM_SOURCE`: GitHub repository URL. Local paths and non-GitHub HTTP
  URLs are rejected.
- `--query`: smoke-test query printed after ingest or used by `search`.
- `--cache`: disposable archive workspace. Defaults to
  `tmp/curriculum-knowledge-prototype`.

## Data Flow

### 1. Resolve Source

The prototype accepts a GitHub repository URL:

```text
https://github.com/<owner>/<repo>
```

It converts that to a GitHub codeload zip URL:

```text
https://github.com/p-n-ai/oss
-> https://codeload.github.com/p-n-ai/oss/zip/refs/heads/main
```

The URL is used for ingest. Retrieval uses stored content rows.

### 2. Download Archive

The zip is downloaded to a disposable workspace:

```text
tmp/curriculum-knowledge-prototype/<temp-dir>/source.zip
```

The HTTP request has an explicit timeout. Download failures stop ingest before
database replacement.

### 3. Extract Safely

The archive is extracted under the same temp workspace.

Extraction rejects unsafe paths so archive members cannot write outside the temp
directory. After extraction, the prototype detects the archive root:

```text
oss-main/
```

### 4. Parse Curriculum Files

The parser walks the extracted repository and maps OSS-shaped curriculum files
into semantic rows.

Translation sync directories are skipped; the prototype reads canonical
curriculum files only.

| File shape | Row kind | Purpose |
| --- | --- | --- |
| `syllabus.yaml` | `syllabus` | Inventory and scope |
| `subject.yaml` | `subject` | Inventory and filtering |
| `subject-grade.yaml` | `subject_grade` | Grade/form context |
| topic YAML | `topic_card` | Topic resolution |
| `*.teaching.md` sections | `teaching_note` | Tutor explanation context |
| `*.examples.yaml` items | `worked_example` | Worked-example retrieval |
| `*.assessments.yaml` questions | `assessment_item` | Future quiz/practice retrieval |

This prototype chunks by curriculum semantics first, not blind token windows.
Malformed example or assessment files are skipped so one optional practice file
does not block topic and teaching-note ingest.

### 5. Replace Postgres Content

Ingest creates the scratch schema/table if needed, then replaces content rows in
one transaction.

The table is small:

```sql
CREATE TABLE retrieval_prototype.curriculum_content (
    id                BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    curriculum_source TEXT NOT NULL DEFAULT '',
    kind              TEXT NOT NULL,
    title             TEXT NOT NULL DEFAULT '',
    body              TEXT NOT NULL,
    metadata          JSONB NOT NULL DEFAULT '{}',
    search_text       TEXT NOT NULL,
    search_vector     TSVECTOR GENERATED ALWAYS AS (
        to_tsvector('simple', search_text)
    ) STORED,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

Indexes:

```sql
CREATE INDEX curriculum_content_kind_idx
    ON retrieval_prototype.curriculum_content (kind);

CREATE INDEX curriculum_content_metadata_gin_idx
    ON retrieval_prototype.curriculum_content USING GIN (metadata);

CREATE INDEX curriculum_content_search_vector_idx
    ON retrieval_prototype.curriculum_content USING GIN (search_vector);
```

Prototype rules:

- one table only;
- one source label only: `curriculum_source`;
- no pgvector;
- no source path, external ID, owner, tenant, or activation model;
- no persisted ingest/index status;
- re-ingest replaces rows instead of appending duplicates.

### 6. Seed And Schema Boundary

This prototype does not change the normal app seed path or production
migrations.

The CLI creates `retrieval_prototype.curriculum_content` at runtime so the
GitHub-to-Postgres flow can be tested without committing a product schema yet.
The scratch schema is disposable and removed by `drop`.

Production work should replace this with:

- a real migration;
- tenant-scoped retrieval tables;
- persisted source/run status such as `pending`, `fetching`, `ingesting`,
  `indexing`, `ready`, and `failed`;
- a real ingest or seed command;
- admin check/activate behavior.

Admin activation should only be allowed after a source reaches `ready`.
Search/fetch consumers should treat missing or non-ready status as unavailable,
not as an empty curriculum.

### 7. Search Postgres Rows

Search uses Postgres full-text search:

```sql
SELECT
    id,
    curriculum_source,
    kind,
    title,
    body,
    metadata,
    search_text,
    ts_rank_cd(search_vector, websearch_to_tsquery('simple', $1)) AS score
FROM retrieval_prototype.curriculum_content
WHERE search_vector @@ websearch_to_tsquery('simple', $1)
ORDER BY score DESC
LIMIT $2;
```

The CLI prints top hits with score, kind, and title.

## Tutor Runtime Boundary

Runtime integration should keep this boundary:

```text
child message
-> search stored curriculum content
-> resolve topic
-> fetch teaching-note rows for that topic
-> emit curriculum.topic and curriculum.teaching_notes packets
```

Render selected rows through `curriculum.topic` and `curriculum.teaching_notes`
packets.

First slice behavior:

- topic rows identify the likely curriculum topic;
- teaching-note rows provide explanation context;
- assessment rows stay out of ordinary teaching prompts;
- quiz requests stay on the quiz path;
- weak or ambiguous hits should abstain.

## Verification Checklist

- `ingest` works with `CURRICULUM_SOURCE=https://github.com/p-n-ai/oss`.
- Local paths fail.
- Non-GitHub HTTP URLs fail.
- Content rows exist in `retrieval_prototype.curriculum_content`.
- `search` returns relevant rows without re-reading the source archive.
- Temp cache can be deleted after ingest.
- `drop` removes the scratch schema.
- The prototype has no runtime retrieval dependency on `oss/`, GitHub, temp
  files, or `LEARN_CURRICULUM_PATH`.

## Local Test Run

Use this flow to test more than unit behavior. It exercises GitHub download,
archive extraction, OSS parsing, Postgres insert, search, and teardown.

Start local Postgres:

```sh
docker compose up -d postgres
```

Set the local database URL used by this repo's Docker Compose file:

```sh
export DATABASE_URL='postgres://pai:pai@127.0.0.1:5432/pai?sslmode=disable'
export CURRICULUM_SOURCE='https://github.com/p-n-ai/oss'
```

Run unit tests:

```sh
go test ./cmd/curriculum-knowledge-prototype
```

Run the full ingest path:

```sh
go run ./cmd/curriculum-knowledge-prototype ingest --query "linear equations"
```

Expected shape:

```text
postgres prototype: PASS
content_rows: 2272
query: "linear equations"
top_hits:
- score=... kind=topic_card title="Linear Equations"
```

Run search again without re-ingesting:

```sh
go run ./cmd/curriculum-knowledge-prototype search --query "linear equations"
```

Check the stored row mix:

```sh
psql "$DATABASE_URL" -Atqc "
select
  count(*),
  count(*) filter (where kind='topic_card'),
  count(*) filter (where kind='teaching_note'),
  count(*) filter (where kind='assessment_item')
from retrieval_prototype.curriculum_content;
"
```

Expected shape from the current OSS corpus:

```text
2272|154|1151|613
```

Tear down scratch data:

```sh
go run ./cmd/curriculum-knowledge-prototype drop
```

Stop the local Postgres service if this test started it:

```sh
docker compose stop postgres
```

## Not In This Prototype

Later production work can add:

- tenant-scoped active source;
- admin check/activate flow;
- ingest run history and stable error categories;
- source/content/chunk tables;
- embedding rows or pgvector as a derived index over stored content;
- Tutor Turn integration tests for `curriculum.topic` and `curriculum.teaching_notes`.
