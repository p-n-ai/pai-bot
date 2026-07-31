# PaiBot Conversation Tester

Dual-host plugin for running the shared `pai-chat-codex` skill against PaiBot's local, memory-only conversation harness.

The skill launches `just chat-codex`, which pins PaiBot to the managed Codex provider with no fallback. The CLI accepts messages while a reply is running, exposes FIFO queue and interruption controls, and watches `.codex/chat-codex-candidate/` for prompt or character changes.

The candidate file is optional and should be saved atomically:

```text
.codex/chat-codex-candidate/
└── candidate.yaml
```

`prompt` is appended as a local conversation-test instruction. The same snapshot defines selectable learner identities:

```yaml
prompt: |
  Keep replies short and conversational.
default: aina
characters:
  - id: aina
    first_name: Aina
    username: aina
    language: en
```

The CLI reports only the candidate's SHA-256 identity. Apply a detected change with `/reload`; use `/new` to reset while retaining the selected character.

## Claude Code

From the PaiBot repository root:

```bash
claude --plugin-dir ./plugins/pai-conversation-tester
```

After editing the plugin, run `/reload-plugins` in the active Claude Code session.

## Codex

Codex installs local plugins through a marketplace, not `--plugin-dir`. Point a configured local marketplace entry named `pai-conversation-tester` at this directory, then install it with:

```bash
codex plugin add pai-conversation-tester@<local-marketplace-name>
```

For subsequent edits, update the manifest cachebuster and reinstall through the same marketplace:

```bash
python3 ~/.codex/skills/.system/plugin-creator/scripts/update_plugin_cachebuster.py \
  ./plugins/pai-conversation-tester
codex plugin add pai-conversation-tester@<local-marketplace-name>
```

Start a new Codex task after reinstalling. Existing tasks do not pick up updated plugin skills.
