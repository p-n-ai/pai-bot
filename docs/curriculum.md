# Curriculum

P&AI Bot loads curriculum content from structured YAML files. The default curriculum is **KSSM Matematik** (Malaysian national syllabus, Forms 1–3), with Algebra as the primary validation target.

## Source

Curriculum data lives in the [p-n-ai/oss](https://github.com/p-n-ai/oss) repository (Open School Syllabus) and is consumed as a Git submodule at `./oss`.

```bash
git submodule update --init    # Pull curriculum data
```

Set the path via `LEARN_CURRICULUM_PATH` (default: `./oss`).

## Directory Structure

```
oss/
├── curricula/
│   └── malaysia/
│       └── malaysia-kssm/
│           ├── syllabus.yaml                          # Syllabus metadata
│           └── malaysia-kssm-matematik/
│               ├── subject.yaml                       # Subject metadata
│               ├── malaysia-kssm-matematik-tingkatan-1/
│               │   ├── subject-grade.yaml
│               │   └── topics/
│               │       ├── MT1-01.yaml                # Topic definition
│               │       ├── MT1-01.teaching.md         # Teaching notes
│               │       ├── MT1-01.assessments.yaml    # Quiz questions
│               │       └── MT1-01.examples.yaml       # Worked examples (skipped by loader)
│               ├── malaysia-kssm-matematik-tingkatan-2/
│               └── malaysia-kssm-matematik-tingkatan-3/
├── concepts/
│   └── mathematics/                                   # Cross-curriculum concept definitions
└── schema/                                            # JSON Schema validation rules
```

Topic files use **topic codes** (e.g., `MT1-01`, `MT2-05`, `MT3-12`), not descriptive names. The code after the hyphen is a sequential number within the form.

## File Types

### Topic YAML (`*.yaml`)

Defines a topic with learning objectives, prerequisites, and metadata:

```yaml
id: MT1-01
name: "Nombor Nisbah"
name_en: "Rational Numbers"
subject_grade_id: malaysia-kssm-matematik-tingkatan-1
subject_id: malaysia-kssm-matematik
syllabus_id: malaysia-kssm
country_id: malaysia
language: ms
quality_level: 3
provenance: ai-generated
difficulty: beginner        # string: beginner, intermediate, advanced
tier: core                  # string: core, enrichment, extension

prerequisites:
  required: []

bloom_levels:
  - remember
  - understand
  - apply

learning_objectives:
  - id: "1.1.1"
    text: "Mengenal nombor positif dan nombor negatif berdasarkan situasi sebenar."
    text_en: "Recognize positive and negative numbers based on real situations."
    bloom: remember
  - id: "1.1.2"
    text: "Mengenal dan memerihalkan integer."
    text_en: "Recognize and describe integers."
    bloom: understand
```

Key fields consumed by the loader (`internal/curriculum/types.go`):
- `id`, `name`, `subject_id`, `syllabus_id` — identity and hierarchy
- `difficulty`, `tier` — string values (not numbers)
- `learning_objectives` — each with `id`, `text`, and `bloom` level
- `prerequisites` — `required[]` and `recommended[]` topic IDs
- `quality_level`, `provenance` — data quality tracking

### Teaching Notes (`*.teaching.md`)

Markdown files with instructor guidance. Loaded into the AI system prompt to provide curriculum-aligned context. Retrieved via `Loader.GetTeachingNotes(topicID)`.

Teaching notes are matched to topics by filename: `MT1-01.teaching.md` pairs with `MT1-01.yaml`.

### Assessment YAML (`*.assessments.yaml`)

Quiz questions with answers, hints, and difficulty levels:

```yaml
topic_id: MT1-01
provenance: ai-generated

questions:
  - id: q1
    text: "Selesaikan: 2x + 5 = 13"
    difficulty: easy
    learning_objective: "1.1.1"
    answer:
      type: numeric
      value: "4"
      working: "2x = 13 - 5 = 8, x = 8/2 = 4"
    hints:
      - level: 0
        text: "Alihkan pemalar ke sebelah kanan dahulu"
      - level: 1
        text: "2x = 13 - 5. Berapakah 13 - 5?"
    distractors:
      - value: "9"
        feedback: "Anda menambah 5 dan bukannya menolak"
      - value: "6.5"
        feedback: "Anda membahagi 13 dengan 2 tanpa menolak 5"
```

### Subject YAML (`subject.yaml`)

Metadata for a subject within a syllabus:

```yaml
id: malaysia-kssm-matematik
name: "Matematik"
name_en: "Mathematics"
syllabus_id: malaysia-kssm
country_id: malaysia
language: ms
provenance: ai-generated
```

### Syllabus YAML (`syllabus.yaml`)

Top-level syllabus definition:

```yaml
id: malaysia-kssm
name: "Kurikulum Standard Sekolah Menengah"
name_en: "Standard Curriculum for Secondary Schools"
board: malaysia
level: kssm
version: "2017"
country_id: malaysia
language: ms
subjects:
  - malaysia-kssm-matematik-tingkatan-1
  - malaysia-kssm-matematik-tingkatan-2
  - malaysia-kssm-matematik-tingkatan-3
```

### Skipped Files

The loader explicitly skips `*.examples.yaml` files — these contain worked examples used by the OSS repository but are not consumed by the bot's runtime.

## How Curriculum Is Used

1. **Topic Detection:** When a student asks a question, the agent engine uses BM25 retrieval (`internal/retrieval/` and `internal/agent/curriculum_retriever.go`) to match it to the most relevant topic, factoring in the student's form level and conversation context.

2. **System Prompt Injection:** The matched topic's teaching notes are injected into the AI system prompt, along with the curriculum citation (e.g., "KSSM Form 1 > Algebra > Linear Equations").

3. **Adaptive Depth:** The system prompt adjusts explanation depth based on the student's mastery level for that topic (`internal/agent/adaptive_depth.go`):
   - Mastery < 0.3: Simple language, more examples, smaller steps
   - Mastery 0.3–0.6: Standard explanations, introduce formal notation
   - Mastery > 0.6: Concise, edge cases, cross-topic connections

4. **Quiz Engine:** Assessment questions are loaded for quizzes. When a student completes all loaded questions and slots remain (up to `QuizMaxQuestions = 10` per session), the AI generates additional questions using `CompleteJSON`, styled after real exam exemplars.

5. **Progress Tracking:** Mastery scores per topic drive spaced repetition scheduling (SM-2 algorithm in `internal/progress/spaced_rep.go`) and topic unlocking.

## How the Loader Works

The curriculum loader (`internal/curriculum/loader.go`) walks the filesystem under `LEARN_CURRICULUM_PATH` and matches files by pattern:

| Pattern | What It Loads |
|---------|---------------|
| `syllabus.yaml` / `syllabus.yml` | Syllabus metadata |
| `subject.yaml` / `subject.yml` | Subject metadata |
| `*.assessments.yaml` / `*.assessments.yml` | Assessment questions for a topic |
| `*.teaching.md` | Teaching notes (linked to matching topic by filename) |
| `*.examples.yaml` | **Skipped** |
| Other `*.yaml` files with `id` field | Topic definitions |

All data is held in memory with `sync.RWMutex` for thread-safe concurrent reads. There is no external cache dependency for curriculum data.

## Adding New Curriculum

To add content for a new syllabus or subject:

1. Create the directory structure under `oss/curricula/` following the `{country}/{syllabus}/{subject}/` pattern
2. Add `syllabus.yaml` and `subject.yaml` files with unique IDs
3. Add topic YAML, teaching notes, and assessment files per topic using topic code naming (e.g., `MT1-01.yaml`)
4. Ensure each topic has a unique `id` field and `quality_level >= 3` for AI-ready content
5. Run the bot — the curriculum loader picks up new files automatically on startup

See [p-n-ai/oss](https://github.com/p-n-ai/oss) for contribution guidelines, ID conventions, and the full JSON Schema reference.
