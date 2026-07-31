#!/usr/bin/env python3
"""Codex lifecycle guard for bounded validation and focused test authoring.

PreToolUse blocks raw validation, guardrail self-editing, direct shell writes to
test paths, and a single oversized test patch before it runs. PostToolUse and
Stop check the complete Git diff so many individually small edits cannot bypass
the policy.

This is a workflow guardrail, not a security boundary.
"""

from __future__ import annotations

import json
import os
import re
import shlex
import sys
from dataclasses import dataclass
from pathlib import Path
from typing import Iterable, Sequence

sys.dont_write_bytecode = True
sys.path.insert(0, str(Path(__file__).resolve().parent))

from test_policy import (  # noqa: E402
    BYPASS_ENV as TEST_EDIT_OVERRIDE_ENV,
    POLICY_PATH as TEST_POLICY_PATH,
    TestPolicyError,
    classify_path,
    evaluate_patch,
    evaluate_repository,
    load_policy,
    normalize_path as normalize_policy_path,
    repository_root,
)

VALIDATION_OVERRIDE_ENV = "CODEX_ALLOW_FULL_VALIDATION"
EDIT_OVERRIDE_ENV = "CODEX_ALLOW_GUARDRAIL_EDITS"
ALLOWED_WRAPPER = "./scripts/agent-check"
PROTECTED_PATHS = {
    "AGENTS.md",
    ".codex/agent-check.json",
    ".codex/test-policy.json",
    ".codex/config.toml",
    ".codex/hooks.json",
    ".codex/hooks/test_guard.py",
    ".codex/hooks/test_policy.py",
    ".github/CODEOWNERS",
    ".github/workflows/codex-test-policy.yml",
    "scripts/agent-check",
    "scripts/doctor",
    "scripts/test-policy",
    "scripts/test-policy-ci",
}
PATCH_FILE_HEADER = re.compile(
    r"^\*\*\*\s+(?:Update|Add|Delete)\s+File:\s*(.+?)\s*$",
    re.MULTILINE,
)

# Direct executables whose primary purpose is validation.
DIRECT_VALIDATION_RUNNERS = {
    "ava",
    "cargo-nextest",
    "clippy-driver",
    "compileall",
    "cypress",
    "detox",
    "eslint",
    "golangci-lint",
    "jest",
    "junit",
    "mocha",
    "mypy",
    "nextest",
    "nose",
    "nose2",
    "phpunit",
    "playwright",
    "pytest",
    "rspec",
    "ruff",
    "shellcheck",
    "swiftlint",
    "tox",
    "tsc",
    "unittest",
    "vitest",
}

PACKAGE_MANAGERS = {"npm", "pnpm", "yarn", "bun", "npx", "bunx"}
PYTHON_LAUNCHERS = {"python", "python3", "py", "uv", "poetry", "pipenv"}
SHELLS = {"bash", "dash", "fish", "sh", "zsh"}

# Script/task names. Suffixes such as test:unit or lint-fix are matched.
TASK_NAME = re.compile(
    r"^(?:pre|post)?(?:"
    r"test|tests|lint|typecheck|type-check|check|build|verify|validation|"
    r"e2e|integration|coverage|ci|bench|benchmark|format|fmt"
    r")(?:[:_.-].*)?$",
    re.IGNORECASE,
)

ASSIGNMENT = re.compile(r"^[A-Za-z_][A-Za-z0-9_]*=.*$", re.DOTALL)
SHELL_OPERATOR_CHARS = frozenset(";&|()")


@dataclass(frozen=True)
class Segment:
    argv: tuple[str, ...]
    operator_after: str | None = None


@dataclass(frozen=True)
class Inspection:
    blocked: bool
    reason: str = ""
    is_agent_check: bool = False


def _basename(value: str) -> str:
    name = Path(value).name.lower()
    for suffix in (".exe", ".cmd", ".bat"):
        if name.endswith(suffix):
            name = name[: -len(suffix)]
    return name


def _is_operator(token: str) -> bool:
    return bool(token) and all(char in SHELL_OPERATOR_CHARS for char in token)


def split_shell(command: str) -> list[Segment]:
    """Split a shell command into simple argv segments and joining operators."""

    lexer = shlex.shlex(command, posix=True, punctuation_chars=";&|()")
    lexer.whitespace_split = True
    lexer.commenters = ""

    segments: list[Segment] = []
    current: list[str] = []

    for token in lexer:
        if _is_operator(token):
            if current:
                segments.append(Segment(tuple(current), token))
                current = []
            elif segments:
                previous = segments[-1]
                segments[-1] = Segment(previous.argv, (previous.operator_after or "") + token)
        else:
            current.append(token)

    if current:
        segments.append(Segment(tuple(current), None))

    return segments


def _strip_leading_options(argv: list[str]) -> list[str]:
    while argv and argv[0].startswith("-") and argv[0] != "-":
        argv.pop(0)
    if argv and argv[0] == "--":
        argv.pop(0)
    return argv


def normalize_argv(argv: Sequence[str]) -> tuple[list[str], bool]:
    """Remove common launch wrappers while preserving the actual command."""

    items = list(argv)
    backgroundish = False

    while items and ASSIGNMENT.match(items[0]):
        items.pop(0)

    while items:
        executable = _basename(items[0])

        if executable == "env":
            items.pop(0)
            while items and (items[0].startswith("-") or ASSIGNMENT.match(items[0])):
                items.pop(0)
            continue

        if executable in {"command", "builtin", "exec"}:
            items.pop(0)
            items = _strip_leading_options(items)
            continue

        if executable in {"nohup", "setsid"}:
            backgroundish = True
            items.pop(0)
            items = _strip_leading_options(items)
            continue

        if executable == "sudo":
            items.pop(0)
            # Good-enough parsing for common sudo options. The hook is a guardrail,
            # not a shell parser or security boundary.
            while items and items[0].startswith("-"):
                option = items.pop(0)
                if option in {"-u", "-g", "-h", "-p", "-C", "-T", "-R", "-D"} and items:
                    items.pop(0)
            continue

        if executable in {"time", "nice"}:
            items.pop(0)
            items = _strip_leading_options(items)
            continue

        if executable in {"timeout", "gtimeout"}:
            items.pop(0)
            while items and items[0].startswith("-"):
                option = items.pop(0)
                if option in {"-k", "--kill-after", "-s", "--signal"} and items:
                    items.pop(0)
            if items:
                items.pop(0)  # duration
            continue

        break

    return items, backgroundish


def _task_name(value: str) -> bool:
    normalized = value.strip().lower().lstrip("-")
    if "=" in normalized:
        normalized = normalized.split("=", 1)[-1]
    normalized = Path(normalized).name
    return bool(TASK_NAME.match(normalized))


def _contains_validation_runner(values: Iterable[str]) -> bool:
    return any(_basename(value) in DIRECT_VALIDATION_RUNNERS for value in values)


def _is_allowed_wrapper(executable: str) -> bool:
    return executable == ALLOWED_WRAPPER


def _nested_shell_command(argv: Sequence[str]) -> str | None:
    for index, value in enumerate(argv[1:], start=1):
        if value.startswith("-") and "c" in value[1:]:
            return argv[index + 1] if index + 1 < len(argv) else ""
    return None


def inspect_argv(argv: Sequence[str], operator_after: str | None = None) -> Inspection:
    normalized, backgroundish = normalize_argv(argv)
    if not normalized:
        return Inspection(False)

    executable_raw = normalized[0]
    executable = _basename(executable_raw)
    args = normalized[1:]

    if _is_allowed_wrapper(executable_raw):
        is_background = backgroundish or bool(operator_after and "&" in operator_after and "&&" not in operator_after)
        if is_background:
            return Inspection(
                True,
                "The bounded agent-check must run in the foreground; background polling recreates the wait-loop problem.",
                True,
            )
        if args != ["changed"]:
            return Inspection(
                True,
                "Only the exact command './scripts/agent-check changed' is allowed inside Codex.",
                True,
            )
        return Inspection(False, is_agent_check=True)

    if executable in SHELLS:
        nested = _nested_shell_command(normalized)
        if nested is None:
            return Inspection(False)
        try:
            nested_segments = split_shell(nested)
        except ValueError:
            return Inspection(True, "Could not safely parse a nested shell command.")
        for segment in nested_segments:
            result = inspect_argv(segment.argv, segment.operator_after)
            if result.blocked:
                return result
        return Inspection(False)

    if executable in DIRECT_VALIDATION_RUNNERS:
        return Inspection(True, f"Direct validation runner '{executable}' is blocked.")

    if executable in PACKAGE_MANAGERS:
        lowered = [value.lower() for value in args]
        if executable in {"npx", "bunx"} and _contains_validation_runner(args):
            return Inspection(True, f"Direct validation through '{executable}' is blocked.")

        # npm/pnpm/yarn/bun test, run test:unit, exec vitest, dlx eslint, etc.
        meaningful = [value for value in lowered if value not in {"run", "run-script", "exec", "dlx", "--"}]
        if any(_task_name(value) for value in meaningful) or _contains_validation_runner(meaningful):
            return Inspection(True, f"Package-manager validation through '{executable}' is blocked.")
        return Inspection(False)

    if executable in PYTHON_LAUNCHERS:
        lowered = [value.lower() for value in args]
        if _contains_validation_runner(lowered):
            return Inspection(True, f"Python validation through '{executable}' is blocked.")
        if "-m" in lowered:
            index = lowered.index("-m")
            if index + 1 < len(lowered) and (
                _basename(lowered[index + 1]) in DIRECT_VALIDATION_RUNNERS
                or _task_name(lowered[index + 1])
            ):
                return Inspection(True, f"Python module validation through '{executable}' is blocked.")
        return Inspection(False)

    if executable == "cargo":
        if any(value.lower() in {"test", "check", "clippy", "build", "bench", "fmt"} for value in args):
            return Inspection(True, "Cargo validation/build commands are blocked.")
        return Inspection(False)

    if executable == "go":
        if any(value.lower() in {"test", "vet", "build"} for value in args):
            return Inspection(True, "Go validation/build commands are blocked.")
        return Inspection(False)

    if executable in {"make", "gmake"}:
        targets = [value for value in args if not value.startswith("-") and "=" not in value]
        if not targets or any(_task_name(value) for value in targets):
            return Inspection(True, "Make validation/default build is blocked.")
        return Inspection(False)

    if executable in {"just", "task", "nx", "turbo", "bazel", "bazelisk", "buck2"}:
        if any(_task_name(value) for value in args) or _contains_validation_runner(args):
            return Inspection(True, f"Validation task through '{executable}' is blocked.")
        return Inspection(False)

    if executable in {"mvn", "mvnw", "gradle", "gradlew", "dotnet", "swift"}:
        if any(_task_name(value) for value in args):
            return Inspection(True, f"Validation/build through '{executable}' is blocked.")
        return Inspection(False)

    return Inspection(False)

def normalize_patch_path(value: str) -> str:
    path = value.strip().strip("\"'").replace("\\", "/")
    while path.startswith("./"):
        path = path[2:]
    if path.startswith("a/") or path.startswith("b/"):
        path = path[2:]
    return path


def protected_patch_paths(command: str) -> list[str]:
    found = {
        path
        for raw in PATCH_FILE_HEADER.findall(command)
        if (path := normalize_patch_path(raw)) in PROTECTED_PATHS
    }
    return sorted(found)


def deny_pre(reason: str, guidance: str) -> None:
    payload = {
        "hookSpecificOutput": {
            "hookEventName": "PreToolUse",
            "permissionDecision": "deny",
            "permissionDecisionReason": f"{reason} {guidance}",
        }
    }
    print(json.dumps(payload, separators=(",", ":")))


def block_post(reason: str) -> None:
    payload = {
        "decision": "block",
        "reason": reason,
        "hookSpecificOutput": {
            "hookEventName": "PostToolUse",
            "additionalContext": reason,
        },
    }
    print(json.dumps(payload, separators=(",", ":")))


def block_stop(reason: str, *, already_continued: bool) -> None:
    if already_continued:
        payload = {
            "continue": False,
            "stopReason": "test-authoring-policy-remains-violated",
            "systemMessage": reason,
        }
    else:
        payload = {"decision": "block", "reason": reason}
    print(json.dumps(payload, separators=(",", ":")))


def _root_from_payload(payload: dict) -> Path:
    cwd = payload.get("cwd")
    return repository_root(Path(cwd) if isinstance(cwd, str) and cwd else None)


def _candidate_paths_from_shell(command: str) -> list[str]:
    """Extract literal path-like shell tokens for a best-effort write preflight."""

    try:
        segments = split_shell(command)
    except ValueError:
        return []
    candidates: list[str] = []
    for segment in segments:
        for value in segment.argv[1:]:
            if value.startswith("-") or value == "-":
                continue
            normalized = normalize_policy_path(value)
            if "/" in normalized or "." in Path(normalized).name:
                candidates.append(normalized)
    return candidates


def _looks_like_shell_write(command: str) -> bool:
    lowered = command.lower()
    if re.search(r"(?:^|[^<])>{1,2}(?!=)", command):
        return True
    write_words = (
        " tee ",
        " cp ",
        " mv ",
        " install ",
        " touch ",
        " truncate ",
        " rm ",
        " unlink ",
        "sed -i",
        "perl -pi",
    )
    padded = f" {lowered} "
    return any(word in padded for word in write_words)


READ_ONLY_EXECUTABLES = {
    "[",
    "basename",
    "cat",
    "cd",
    "cut",
    "date",
    "df",
    "dirname",
    "du",
    "echo",
    "egrep",
    "false",
    "fgrep",
    "file",
    "grep",
    "head",
    "id",
    "jq",
    "ls",
    "popd",
    "printf",
    "pushd",
    "pwd",
    "readlink",
    "realpath",
    "rg",
    "sort",
    "stat",
    "tail",
    "test",
    "tr",
    "tree",
    "true",
    "uname",
    "uniq",
    "wc",
    "whoami",
}
READ_ONLY_GIT_SUBCOMMANDS = {
    "blame",
    "cat-file",
    "describe",
    "diff",
    "grep",
    "log",
    "ls-files",
    "ls-tree",
    "merge-base",
    "name-rev",
    "rev-parse",
    "shortlog",
    "show",
    "status",
}


def _git_subcommand(args: Sequence[str]) -> tuple[str | None, list[str]]:
    values = list(args)
    index = 0
    options_with_value = {
        "-C",
        "-c",
        "--config-env",
        "--exec-path",
        "--git-dir",
        "--namespace",
        "--super-prefix",
        "--work-tree",
    }
    while index < len(values):
        value = values[index]
        if value == "--":
            index += 1
            break
        if value in options_with_value:
            index += 2
            continue
        if any(value.startswith(prefix + "=") for prefix in options_with_value if prefix.startswith("--")):
            index += 1
            continue
        if value.startswith("-"):
            index += 1
            continue
        return value.lower(), values[index + 1 :]
    return None, values[index:]


def _read_only_argv(argv: Sequence[str]) -> bool:
    normalized, backgroundish = normalize_argv(argv)
    if not normalized or backgroundish:
        return False
    executable = _basename(normalized[0])
    args = normalized[1:]

    if executable in SHELLS:
        nested = _nested_shell_command(normalized)
        return nested is not None and _is_known_read_only_shell(nested)

    if executable in READ_ONLY_EXECUTABLES:
        return True

    if executable == "find":
        mutating = {"-delete", "-exec", "-execdir", "-ok", "-okdir"}
        return not any(value in mutating for value in args)

    if executable == "fd":
        return not any(
            value in {"-x", "-X", "--exec", "--exec-batch"}
            or value.startswith("--exec=")
            or value.startswith("--exec-batch=")
            for value in args
        )

    if executable == "git":
        subcommand, subargs = _git_subcommand(args)
        if subcommand in READ_ONLY_GIT_SUBCOMMANDS:
            return True
        if subcommand == "remote":
            return bool(subargs) and subargs[0] in {"get-url", "show"}
        if subcommand == "branch":
            return any(
                value in {"--list", "-l", "--show-current", "--contains", "--merged", "--no-merged"}
                for value in subargs
            )
        return False

    return False


def _is_known_read_only_shell(command: str) -> bool:
    """Return true only when every parsed segment is a known inspection command."""

    if _looks_like_shell_write(command):
        return False
    try:
        segments = split_shell(command)
    except ValueError:
        return False
    return bool(segments) and all(_read_only_argv(segment.argv) for segment in segments)


def risky_shell_write_paths(root: Path, command: str) -> tuple[list[str], list[str]]:
    """Find obvious shell writes to protected or test-bearing paths.

    Arbitrary shell programs can still generate paths dynamically. The cumulative
    PostToolUse check catches those after the command; this preflight mainly keeps
    ordinary redirects, tee/cp/mv, and generated Codex commands on apply_patch.
    """

    if not _looks_like_shell_write(command):
        return [], []
    config, _, _ = load_policy(root)
    protected: set[str] = set()
    test_paths: set[str] = set()
    for candidate in _candidate_paths_from_shell(command):
        if candidate in PROTECTED_PATHS:
            protected.add(candidate)
        if classify_path(candidate, config) in {
            "test",
            "test_infrastructure",
            "expensive_test",
            "snapshot",
            "fixture",
        }:
            test_paths.add(candidate)
    return sorted(protected), sorted(test_paths)


def _parse_payload() -> tuple[dict, str, str, str]:
    payload = json.load(sys.stdin)
    event = payload.get("hook_event_name")
    tool_name = payload.get("tool_name")
    if event not in {"PreToolUse", "PostToolUse", "Stop"}:
        raise ValueError("expected PreToolUse, PostToolUse, or Stop")
    if event == "Stop":
        return payload, event, "", ""
    if tool_name not in {"Bash", "apply_patch"}:
        return payload, event, str(tool_name or ""), ""
    command = payload.get("tool_input", {}).get("command")
    if not isinstance(command, str) or not command.strip():
        raise ValueError("missing tool_input.command")
    return payload, event, tool_name, command


def handle_pre(payload: dict, tool_name: str, command: str) -> None:
    if tool_name not in {"Bash", "apply_patch"}:
        return

    if tool_name == "apply_patch":
        if os.environ.get(EDIT_OVERRIDE_ENV) != "1":
            protected = protected_patch_paths(command)
            if protected:
                deny_pre(
                    f"Codex may not edit active guardrail files: {', '.join(protected)}.",
                    f"For deliberate maintenance, start the Codex parent process with {EDIT_OVERRIDE_ENV}=1.",
                )
                return

        if os.environ.get(TEST_EDIT_OVERRIDE_ENV) != "1":
            try:
                report = evaluate_patch(_root_from_payload(payload), command)
            except (TestPolicyError, OSError) as exc:
                deny_pre(
                    f"Test-authoring guard could not inspect the patch ({exc}).",
                    f"Review {TEST_POLICY_PATH} instead of bypassing it from the child command.",
                )
                return
            if not report.ok:
                deny_pre(report.concise(), report.guidance())
        return

    try:
        root = _root_from_payload(payload)
        protected, test_paths = risky_shell_write_paths(root, command)
    except (TestPolicyError, OSError) as exc:
        deny_pre(
            f"Test-authoring guard could not inspect a shell write ({exc}).",
            f"Review {TEST_POLICY_PATH} instead of bypassing it from the child command.",
        )
        return

    if protected and os.environ.get(EDIT_OVERRIDE_ENV) != "1":
        deny_pre(
            f"Shell write to protected guardrail path(s) is blocked: {', '.join(protected)}.",
            f"Use {EDIT_OVERRIDE_ENV}=1 only for deliberate guardrail maintenance.",
        )
        return
    if test_paths and os.environ.get(TEST_EDIT_OVERRIDE_ENV) != "1":
        deny_pre(
            f"Direct shell write to test path(s) is blocked: {', '.join(test_paths)}.",
            "Use apply_patch so the focused test budget is checked before the edit.",
        )
        return

    if os.environ.get(VALIDATION_OVERRIDE_ENV) == "1":
        return

    try:
        segments = split_shell(command)
    except ValueError as exc:
        deny_pre(
            f"Validation guard could not safely parse the shell command ({exc}).",
            "Use ./scripts/agent-check changed instead.",
        )
        return

    for segment in segments:
        result = inspect_argv(segment.argv, segment.operator_after)
        if result.blocked:
            deny_pre(
                result.reason,
                f"Use ./scripts/agent-check changed instead. For an intentional full run, "
                f"start Codex with {VALIDATION_OVERRIDE_ENV}=1 or run it outside Codex.",
            )
            return


def handle_post(payload: dict, tool_name: str, command: str) -> None:
    if tool_name not in {"Bash", "apply_patch"}:
        return
    if os.environ.get(TEST_EDIT_OVERRIDE_ENV) == "1":
        return
    # apply_patch is always a write. Skip the complete diff rescan only for a
    # conservative allowlist of known read-only shell commands. Unknown commands
    # are rescanned after execution, which catches interpreter-based or custom
    # writes without penalizing normal rg/git/cat inspection loops.
    if tool_name == "Bash" and _is_known_read_only_shell(command):
        return
    try:
        report = evaluate_repository(_root_from_payload(payload))
    except (TestPolicyError, OSError) as exc:
        block_post(f"Test-authoring guard could not inspect the cumulative Git diff: {exc}.")
        return
    if not report.ok:
        block_post(report.guidance())



def handle_stop(payload: dict) -> None:
    if os.environ.get(TEST_EDIT_OVERRIDE_ENV) == "1":
        print("{}")
        return
    try:
        report = evaluate_repository(_root_from_payload(payload))
    except (TestPolicyError, OSError) as exc:
        block_stop(
            f"Test-authoring guard could not inspect the final Git diff: {exc}.",
            already_continued=bool(payload.get("stop_hook_active")),
        )
        return
    if report.ok:
        print("{}")
        return
    # One correction pass is enough. A second continuation would recreate the
    # unbounded agent loop this repository is intended to prevent.
    block_stop(
        report.guidance(),
        already_continued=bool(payload.get("stop_hook_active")),
    )

def main() -> int:
    try:
        payload, event, tool_name, command = _parse_payload()
    except Exception as exc:
        # Fail closed with the shape appropriate for the event when possible.
        try:
            raw_event = payload.get("hook_event_name")  # type: ignore[name-defined]
        except Exception:
            raw_event = "PreToolUse"
        if raw_event == "PostToolUse":
            block_post(f"Guard could not parse the PostToolUse request: {exc}.")
        elif raw_event == "Stop":
            block_stop(
                f"Guard could not parse the Stop request: {exc}.",
                already_continued=False,
            )
        else:
            deny_pre(
                f"Guard could not parse the tool request ({exc}).",
                "Review the hook input instead of bypassing it.",
            )
        return 0

    if event == "PreToolUse":
        handle_pre(payload, tool_name, command)
    elif event == "PostToolUse":
        handle_post(payload, tool_name, command)
    else:
        handle_stop(payload)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
