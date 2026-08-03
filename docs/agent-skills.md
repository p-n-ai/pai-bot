# Agent Skills

Pai-bot can load trusted, local [Agent Skills](https://agentskills.io/specification) bundles for native tool-capable teaching turns.

Pai-bot loads built-in system skills from `./skills` by default. Set `LEARN_SKILLS_PATH` only to override that trusted directory. It contains one child directory per skill:

```text
skills/
└── maths-coach/
    ├── SKILL.md
    ├── references/
    ├── scripts/
    └── assets/
```

At startup, pai-bot validates each `SKILL.md` and loads only its `name` and `description` into the tutor prompt. When a learner request matches, the model uses `load_skill` to load the complete Markdown instructions. It can then use `read_skill_resource` for referenced UTF-8 text files. Reads are limited to the selected skill directory and files up to 1 MiB.

The configured directory is an operator-controlled trust boundary. Pai-bot does not download skills or execute bundled scripts. The experimental `allowed-tools` field is parsed for format compatibility but does not grant tools or override pai-bot's runtime policy.

Set `metadata.activation: always` for trusted skills that must be added to every model-facing system prompt. This works with managed Codex and providers without native tool continuation. Other skills automatically use metadata-first, tool-driven activation when a native provider is available.

In `just chat-codex`, `/reload` re-parses `LEARN_SKILLS_PATH` and starts a fresh memory session. An invalid skill rejects the reload and preserves the active session. The long-running server loads skills at startup and currently requires a restart to refresh them.

Invalid system skills fail startup so pai-bot never silently runs without its configured behavior.
