#!/usr/bin/env python3
"""Dependency-free policy for keeping agent-authored tests focused and reviewable.

The policy does not try to score test quality and deliberately avoids a universal
"test lines per product line" ratio. It enforces configurable review tripwires
around the failure modes that can be measured reliably from a Git diff:

* too many test files or cases in one task;
* new test frameworks, runner configuration, or helper subsystems in routine work;
* new high-cost integration/E2E tests in a normal implementation session;
* large generated snapshots, fixtures, or golden files;
* deleting whole test files;
* tests-only changes hidden inside a normal feature/fix task.

Semantic guidance such as testing public behavior and avoiding duplicate coverage
lives in AGENTS.md. This module supplies the deterministic layer beneath it.
"""

from __future__ import annotations

import difflib
import fnmatch
import json
import os
import re
import subprocess
from dataclasses import asdict, dataclass, field
from pathlib import Path, PurePosixPath
from typing import Any, Iterable, Sequence

POLICY_PATH = ".codex/test-policy.json"
PROFILE_ENV = "CODEX_TEST_PROFILE"
BYPASS_ENV = "CODEX_ALLOW_BROAD_TEST_EDITS"
BASE_ENV = "CODEX_TEST_BASE"

TEST_KINDS = {"test", "expensive_test"}
TEST_INFRASTRUCTURE_KIND = "test_infrastructure"
ARTIFACT_KINDS = {"snapshot", "fixture"}
TEST_BEARING_KINDS = TEST_KINDS | ARTIFACT_KINDS | {TEST_INFRASTRUCTURE_KIND}

PATCH_FILE_HEADER = re.compile(
    r"^\*\*\*\s+(Add|Update|Delete)\s+File:\s*(.+?)\s*$",
    re.MULTILINE,
)
PATCH_MOVE_TO = re.compile(r"^\*\*\*\s+Move to:\s*(.+?)\s*$", re.MULTILINE)

# Count concrete test cases, not suites/describes. These patterns are intentionally
# conservative and cover common JS/TS, Python, Rust, Go, JVM, .NET, Ruby, PHP,
# Swift, C/C++, Dart, and Elixir styles.
TEST_CASE_PATTERNS = (
    re.compile(r"^\s*(?:async\s+)?def\s+test_[A-Za-z0-9_]*\s*\("),
    re.compile(r"^\s*(?:it|test|specify)\s*(?:\.(?:only|skip|todo|concurrent|fails|each))?\s*\("),
    re.compile(r"^\s*#\s*\[(?:(?:tokio|async_std)::)?test(?:\([^]]*\))?\]"),
    re.compile(r"^\s*#\s*\[(?:rstest|case)(?:\([^]]*\))?\]"),
    re.compile(r"^\s*func\s+Test[A-Z0-9_][A-Za-z0-9_]*\s*\("),
    re.compile(r"^\s*@(?:Test|ParameterizedTest|RepeatedTest)\b"),
    re.compile(r"^\s*\[(?:Fact|Theory|Test|TestCase)(?:\([^]]*\))?\]"),
    re.compile(r"^\s*(?:it|specify)\s+[\"']"),
    re.compile(r"^\s*(?:public\s+)?function\s+test[A-Za-z0-9_]*\s*\("),
    re.compile(r"^\s*func\s+test[A-Z0-9_][A-Za-z0-9_]*\s*\("),
    re.compile(r"^\s*(?:TEST|TEST_F|TEST_P|TYPED_TEST)\s*\("),
    re.compile(r"^\s*test\s+[\"']"),
)

PROFILE_INT_KEYS = (
    "max_test_files_touched",
    "max_new_test_files",
    "max_deleted_test_files",
    "max_added_test_lines",
    "max_added_test_cases",
    "max_inline_test_files_touched",
    "max_test_infrastructure_files_touched",
    "max_new_test_infrastructure_files",
    "max_added_test_infrastructure_lines",
    "max_expensive_test_files_touched",
    "max_new_expensive_test_files",
    "max_added_expensive_test_lines",
    "max_snapshot_files_touched",
    "max_new_snapshot_files",
    "max_added_snapshot_lines",
    "max_snapshot_added_lines_per_file",
    "max_fixture_files_touched",
    "max_new_fixture_files",
    "max_added_fixture_lines",
)

PROFILE_BOOL_KEYS = (
    "allow_binary_test_artifacts",
    "require_product_change",
    "allow_new_test_suite",
)

PATTERN_KEYS = (
    "test_file_patterns",
    "test_infrastructure_patterns",
    "expensive_test_patterns",
    "snapshot_patterns",
    "fixture_patterns",
    "support_only_patterns",
    "ignore_patterns",
)


class TestPolicyError(RuntimeError):
    """Raised for invalid policy configuration or repository state."""


@dataclass(frozen=True)
class Change:
    status: str
    path: str
    old_path: str | None = None

    @property
    def display_path(self) -> str:
        if self.old_path and self.old_path != self.path:
            return f"{self.old_path} -> {self.path}"
        return self.path


@dataclass(frozen=True)
class FileMetrics:
    path: str
    old_path: str | None
    status: str
    kind: str
    is_new: bool
    is_deleted: bool
    added_lines: int
    deleted_lines: int
    added_test_cases: int
    binary: bool = False
    oversized: bool = False

    @property
    def display_path(self) -> str:
        if self.old_path and self.old_path != self.path:
            return f"{self.old_path} -> {self.path}"
        return self.path


@dataclass(frozen=True)
class DiffSummary:
    profile: str
    base: str | None
    baseline_has_test_suite: bool
    changed_paths: tuple[str, ...]
    product_files: tuple[str, ...]
    test_files: tuple[FileMetrics, ...]
    inline_test_files: tuple[FileMetrics, ...]
    test_infrastructure_files: tuple[FileMetrics, ...]
    snapshot_files: tuple[FileMetrics, ...]
    fixture_files: tuple[FileMetrics, ...]

    @property
    def test_files_touched(self) -> int:
        return len(self.test_files) + len(self.inline_test_files)

    @property
    def new_test_files(self) -> int:
        return sum(item.is_new for item in self.test_files)

    @property
    def deleted_test_files(self) -> int:
        return sum(item.is_deleted for item in self.test_files)

    @property
    def added_test_lines(self) -> int:
        return sum(item.added_lines for item in self.test_files)

    @property
    def added_test_cases(self) -> int:
        return sum(item.added_test_cases for item in self.test_files + self.inline_test_files)

    @property
    def expensive_test_files(self) -> tuple[FileMetrics, ...]:
        return tuple(item for item in self.test_files if item.kind == "expensive_test")

    @property
    def new_test_infrastructure_files(self) -> int:
        return sum(item.is_new for item in self.test_infrastructure_files)

    @property
    def added_test_infrastructure_lines(self) -> int:
        return sum(item.added_lines for item in self.test_infrastructure_files)

    @property
    def new_expensive_test_files(self) -> int:
        return sum(item.is_new for item in self.expensive_test_files)

    @property
    def added_expensive_test_lines(self) -> int:
        return sum(item.added_lines for item in self.expensive_test_files)

    @property
    def new_snapshot_files(self) -> int:
        return sum(item.is_new for item in self.snapshot_files)

    @property
    def added_snapshot_lines(self) -> int:
        return sum(item.added_lines for item in self.snapshot_files)

    @property
    def new_fixture_files(self) -> int:
        return sum(item.is_new for item in self.fixture_files)

    @property
    def added_fixture_lines(self) -> int:
        return sum(item.added_lines for item in self.fixture_files)

    @property
    def has_test_change(self) -> bool:
        return bool(
            self.test_files
            or self.inline_test_files
            or self.test_infrastructure_files
            or self.snapshot_files
            or self.fixture_files
        )

    @property
    def all_test_metrics(self) -> tuple[FileMetrics, ...]:
        return (
            self.test_files
            + self.inline_test_files
            + self.test_infrastructure_files
            + self.snapshot_files
            + self.fixture_files
        )


@dataclass(frozen=True)
class PolicyReport:
    ok: bool
    profile: str
    summary: DiffSummary
    violations: tuple[str, ...] = field(default_factory=tuple)
    overridden: bool = False
    would_violate: tuple[str, ...] = field(default_factory=tuple)

    def concise(self) -> str:
        summary = self.summary
        counts = (
            f"{summary.test_files_touched} test file(s), "
            f"{summary.added_test_lines} added test line(s), "
            f"{summary.added_test_cases} added test case(s), "
            f"{len(summary.test_infrastructure_files)} test-infrastructure file(s), "
            f"{len(summary.snapshot_files)} snapshot file(s), "
            f"{len(summary.fixture_files)} fixture/golden file(s)"
        )
        if self.overridden:
            return f"test-policy: {self.profile} profile bypassed ({counts})"
        if self.ok:
            return f"test-policy: {self.profile} profile passed ({counts})"
        details = "\n".join(f"- {item}" for item in self.violations)
        return f"test-policy: {self.profile} profile failed ({counts})\n{details}"

    def guidance(self) -> str:
        return (
            self.concise()
            + "\nKeep only tests that prove changed, externally observable behavior. "
            "Prefer the nearest existing test file and the cheapest deterministic layer. "
            "Remove duplicate coverage, speculative input matrices, implementation-detail assertions, "
            "and unrelated snapshots or fixtures. Do not split the same test expansion across smaller "
            "patches because the cumulative Git diff is checked again. "
            f"For a deliberately broader task, a human can start Codex with {PROFILE_ENV}=expanded, "
            f"use {PROFILE_ENV}=tests-only for test maintenance, or set {BYPASS_ENV}=1 for an explicit exception."
        )

    def as_dict(self) -> dict[str, Any]:
        summary = self.summary
        return {
            "ok": self.ok,
            "profile": self.profile,
            "overridden": self.overridden,
            "violations": list(self.violations),
            "would_violate": list(self.would_violate),
            "summary": {
                "base": summary.base,
                "baseline_has_test_suite": summary.baseline_has_test_suite,
                "changed_paths": list(summary.changed_paths),
                "product_files": list(summary.product_files),
                "test_files_touched": summary.test_files_touched,
                "new_test_files": summary.new_test_files,
                "deleted_test_files": summary.deleted_test_files,
                "added_test_lines": summary.added_test_lines,
                "added_test_cases": summary.added_test_cases,
                "inline_test_files_touched": len(summary.inline_test_files),
                "test_infrastructure_files_touched": len(summary.test_infrastructure_files),
                "new_test_infrastructure_files": summary.new_test_infrastructure_files,
                "added_test_infrastructure_lines": summary.added_test_infrastructure_lines,
                "expensive_test_files_touched": len(summary.expensive_test_files),
                "new_expensive_test_files": summary.new_expensive_test_files,
                "added_expensive_test_lines": summary.added_expensive_test_lines,
                "snapshot_files_touched": len(summary.snapshot_files),
                "new_snapshot_files": summary.new_snapshot_files,
                "added_snapshot_lines": summary.added_snapshot_lines,
                "fixture_files_touched": len(summary.fixture_files),
                "new_fixture_files": summary.new_fixture_files,
                "added_fixture_lines": summary.added_fixture_lines,
                "test_files": [asdict(item) for item in summary.test_files],
                "inline_test_files": [asdict(item) for item in summary.inline_test_files],
                "test_infrastructure_files": [
                    asdict(item) for item in summary.test_infrastructure_files
                ],
                "snapshot_files": [asdict(item) for item in summary.snapshot_files],
                "fixture_files": [asdict(item) for item in summary.fixture_files],
            },
        }


def normalize_path(value: str) -> str:
    path = value.strip().strip("\"'").replace("\\", "/")
    while path.startswith("./"):
        path = path[2:]
    if path.startswith("a/") or path.startswith("b/"):
        path = path[2:]
    return str(PurePosixPath(path))


def glob_matches(path: str, pattern: str) -> bool:
    """Match repository paths while making zero-directory uses of **/ work."""

    normalized_path = normalize_path(path)
    normalized_pattern = pattern.replace("\\", "/")
    while normalized_pattern.startswith("./"):
        normalized_pattern = normalized_pattern[2:]
    candidates = {normalized_pattern}
    pending = [normalized_pattern]
    while pending:
        current = pending.pop()
        marker = "**/"
        index = current.find(marker)
        if index >= 0:
            collapsed = current[:index] + current[index + len(marker) :]
            if collapsed not in candidates:
                candidates.add(collapsed)
                pending.append(collapsed)
    return any(fnmatch.fnmatchcase(normalized_path, candidate) for candidate in candidates)


def matches_any(path: str, patterns: Iterable[str]) -> bool:
    return any(glob_matches(path, pattern) for pattern in patterns)


def classify_path(path: str, config: dict[str, Any]) -> str:
    if matches_any(path, config["ignore_patterns"]):
        return "ignored"
    if matches_any(path, config["support_only_patterns"]):
        return "support"
    if matches_any(path, config["test_infrastructure_patterns"]):
        return TEST_INFRASTRUCTURE_KIND
    if matches_any(path, config["snapshot_patterns"]):
        return "snapshot"
    if matches_any(path, config["fixture_patterns"]):
        return "fixture"
    if matches_any(path, config["expensive_test_patterns"]):
        return "expensive_test"
    if matches_any(path, config["test_file_patterns"]):
        return "test"
    return "product"


def count_test_cases(lines: Sequence[str]) -> int:
    return sum(any(pattern.search(line) for pattern in TEST_CASE_PATTERNS) for line in lines)


def _non_negative_int(value: Any, name: str) -> int:
    if not isinstance(value, int) or isinstance(value, bool) or value < 0:
        raise TestPolicyError(f"{name} must be a non-negative integer")
    return value


def load_policy(root: Path, profile_name: str | None = None) -> tuple[dict[str, Any], str, dict[str, Any]]:
    path = root / POLICY_PATH
    try:
        config = json.loads(path.read_text(encoding="utf-8"))
    except FileNotFoundError as exc:
        raise TestPolicyError(f"missing {POLICY_PATH}") from exc
    except json.JSONDecodeError as exc:
        raise TestPolicyError(f"invalid JSON in {POLICY_PATH}: {exc}") from exc

    if config.get("version") != 1:
        raise TestPolicyError(f"{POLICY_PATH} version must be 1")

    _non_negative_int(config.get("max_changed_files"), "max_changed_files")
    max_bytes = _non_negative_int(config.get("max_inspected_file_bytes"), "max_inspected_file_bytes")
    if max_bytes < 1024:
        raise TestPolicyError("max_inspected_file_bytes must be at least 1024")

    profiles = config.get("profiles")
    if not isinstance(profiles, dict) or not profiles:
        raise TestPolicyError(f"{POLICY_PATH} must define profiles")

    requested = profile_name or os.environ.get(PROFILE_ENV) or config.get("default_profile", "focused")
    if not isinstance(requested, str) or requested not in profiles:
        available = ", ".join(sorted(str(name) for name in profiles))
        raise TestPolicyError(f"unknown test profile {requested!r}; available: {available}")

    profile = profiles[requested]
    if not isinstance(profile, dict):
        raise TestPolicyError(f"profile {requested!r} must be an object")

    for key in PROFILE_INT_KEYS:
        _non_negative_int(profile.get(key), f"profiles.{requested}.{key}")
    for key in PROFILE_BOOL_KEYS:
        if not isinstance(profile.get(key), bool):
            raise TestPolicyError(f"profiles.{requested}.{key} must be boolean")

    for key in PATTERN_KEYS:
        values = config.get(key)
        if not isinstance(values, list) or not all(isinstance(value, str) and value for value in values):
            raise TestPolicyError(f"{key} must be an array of non-empty strings")

    return config, requested, profile


def repository_root(cwd: Path | None = None) -> Path:
    current = cwd or Path.cwd()
    completed = subprocess.run(
        ["git", "-C", str(current), "rev-parse", "--show-toplevel"],
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
        check=False,
        timeout=4,
    )
    if completed.returncode != 0:
        raise TestPolicyError("run test-policy inside a Git repository")
    return Path(completed.stdout.strip()).resolve()


def _git_bytes(
    root: Path,
    *args: str,
    check: bool = True,
    timeout: float = 8,
) -> subprocess.CompletedProcess[bytes]:
    try:
        completed = subprocess.run(
            ["git", "-C", str(root), *args],
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            check=False,
            timeout=timeout,
        )
    except subprocess.TimeoutExpired as exc:
        raise TestPolicyError(f"git {' '.join(args)} timed out") from exc
    if check and completed.returncode != 0:
        message = completed.stderr.decode("utf-8", errors="replace").strip()
        if not message:
            message = completed.stdout.decode("utf-8", errors="replace").strip() or "unknown Git error"
        raise TestPolicyError(f"git {' '.join(args)} failed: {message}")
    return completed


def _git_text(root: Path, *args: str, check: bool = True, timeout: float = 8) -> str:
    return _git_bytes(root, *args, check=check, timeout=timeout).stdout.decode(
        "utf-8", errors="surrogateescape"
    )


def _head_exists(root: Path) -> bool:
    return _git_bytes(root, "rev-parse", "--verify", "HEAD", check=False).returncode == 0


def resolve_base(root: Path, requested_base: str | None = None) -> tuple[str | None, str | None]:
    selected = requested_base or os.environ.get(BASE_ENV)
    if selected:
        if not _head_exists(root):
            raise TestPolicyError("--base requires a repository with at least one commit")
        merge_base = _git_text(root, "merge-base", selected, "HEAD").strip()
        if not merge_base:
            raise TestPolicyError(f"could not resolve merge-base for {selected!r}")
        return merge_base, selected
    if _head_exists(root):
        return "HEAD", "HEAD"
    return None, None


def _parse_name_status(raw: bytes) -> list[Change]:
    values = raw.decode("utf-8", errors="surrogateescape").split("\0")
    changes: list[Change] = []
    index = 0
    while index < len(values):
        status = values[index]
        index += 1
        if not status:
            continue
        code = status[0]
        if code in {"R", "C"}:
            if index + 1 >= len(values):
                raise TestPolicyError("could not parse Git rename/copy output")
            old_path = normalize_path(values[index])
            new_path = normalize_path(values[index + 1])
            index += 2
            changes.append(Change(status=status, path=new_path, old_path=old_path))
        else:
            if index >= len(values):
                raise TestPolicyError("could not parse Git name-status output")
            path = normalize_path(values[index])
            index += 1
            changes.append(Change(status=status, path=path))
    return changes


def collect_changes(root: Path, base_commit: str | None) -> list[Change]:
    by_path: dict[str, Change] = {}
    if base_commit:
        raw = _git_bytes(
            root,
            "diff",
            "--name-status",
            "-z",
            "--find-renames",
            "--diff-filter=ACDMRTUXB",
            base_commit,
            "--",
        ).stdout
        for change in _parse_name_status(raw):
            by_path[change.path] = change
    else:
        tracked = _git_text(root, "ls-files", "-z").split("\0")
        for path in tracked:
            if path:
                normalized = normalize_path(path)
                by_path[normalized] = Change(status="A", path=normalized)

    untracked = _git_text(root, "ls-files", "--others", "--exclude-standard", "-z").split("\0")
    for path in untracked:
        if path:
            normalized = normalize_path(path)
            by_path.setdefault(normalized, Change(status="A", path=normalized))

    return sorted(by_path.values(), key=lambda item: (item.path, item.old_path or ""))


def _paths_at_ref(root: Path, ref: str) -> tuple[str, ...]:
    raw = _git_text(root, "ls-tree", "-r", "-z", "--name-only", ref)
    return tuple(normalize_path(path) for path in raw.split("\0") if path)


def _working_paths(root: Path) -> tuple[str, ...]:
    raw = _git_text(root, "ls-files", "-z", "--cached", "--others", "--exclude-standard")
    return tuple(normalize_path(path) for path in raw.split("\0") if path)


def has_test_suite(root: Path, config: dict[str, Any], ref: str | None = None) -> bool:
    """Return whether a committed ref or the current worktree already has tests.

    Snapshots and fixtures do not count as a test suite by themselves. The check
    intentionally follows repository path conventions instead of guessing from
    file contents.
    """

    paths = _paths_at_ref(root, ref) if ref else _working_paths(root)
    return any(classify_path(path, config) in TEST_KINDS for path in paths)


def _base_blob(root: Path, base_commit: str | None, path: str | None) -> bytes | None:
    if not base_commit or not path:
        return None
    completed = _git_bytes(root, "show", f"{base_commit}:{path}", check=False)
    if completed.returncode != 0:
        return None
    return completed.stdout


def _working_blob(root: Path, path: str) -> bytes | None:
    absolute = root / path
    try:
        if absolute.is_symlink():
            return os.readlink(absolute).encode("utf-8", errors="surrogateescape")
        if not absolute.is_file():
            return None
        return absolute.read_bytes()
    except OSError as exc:
        raise TestPolicyError(f"could not read {path}: {exc}") from exc


def _diff_lines(
    before: bytes | None,
    after: bytes | None,
    max_bytes: int,
) -> tuple[list[str], int, bool, bool]:
    before_data = before or b""
    after_data = after or b""
    oversized = len(before_data) > max_bytes or len(after_data) > max_bytes
    binary = b"\0" in before_data or b"\0" in after_data
    if oversized or binary:
        return [], 0, binary, oversized

    before_lines = before_data.decode("utf-8", errors="replace").splitlines()
    after_lines = after_data.decode("utf-8", errors="replace").splitlines()
    matcher = difflib.SequenceMatcher(None, before_lines, after_lines, autojunk=True)
    added: list[str] = []
    deleted = 0
    for operation, before_start, before_end, after_start, after_end in matcher.get_opcodes():
        if operation in {"replace", "insert"}:
            added.extend(after_lines[after_start:after_end])
        if operation in {"replace", "delete"}:
            deleted += before_end - before_start
    return added, deleted, binary, oversized


def _kind_is_test(kind: str) -> bool:
    return kind in TEST_BEARING_KINDS


def _category_new(status: str, old_kind: str, new_kind: str, category: set[str]) -> bool:
    return new_kind in category and (status.startswith("A") or old_kind not in category)


def _category_deleted(status: str, old_kind: str, new_kind: str, category: set[str]) -> bool:
    return old_kind in category and (status.startswith("D") or new_kind not in category)


def _metric_for_change(
    root: Path,
    change: Change,
    base_commit: str | None,
    config: dict[str, Any],
) -> tuple[FileMetrics | None, FileMetrics | None, str | None]:
    old_path = change.old_path or change.path
    old_kind = classify_path(old_path, config)
    new_kind = "deleted" if change.status.startswith("D") else classify_path(change.path, config)

    before = _base_blob(root, base_commit, old_path)
    after = None if change.status.startswith("D") else _working_blob(root, change.path)
    added, deleted, binary, oversized = _diff_lines(
        before,
        after,
        config["max_inspected_file_bytes"],
    )
    cases = count_test_cases(added)

    category_kind: str | None = None
    if new_kind in TEST_BEARING_KINDS:
        category_kind = new_kind
    elif old_kind in TEST_BEARING_KINDS:
        category_kind = old_kind

    test_metric: FileMetrics | None = None
    inline_metric: FileMetrics | None = None

    if category_kind:
        if category_kind in TEST_KINDS:
            category = TEST_KINDS
        else:
            category = {category_kind}
        test_metric = FileMetrics(
            path=change.path,
            old_path=change.old_path,
            status=change.status,
            kind=category_kind,
            is_new=_category_new(change.status, old_kind, new_kind, category),
            is_deleted=_category_deleted(change.status, old_kind, new_kind, category),
            added_lines=len(added),
            deleted_lines=deleted,
            added_test_cases=cases if category_kind in TEST_KINDS else 0,
            binary=binary,
            oversized=oversized,
        )
    elif new_kind == "product" and cases:
        inline_metric = FileMetrics(
            path=change.path,
            old_path=change.old_path,
            status=change.status,
            kind="inline_test",
            is_new=False,
            is_deleted=False,
            added_lines=0,
            deleted_lines=deleted,
            added_test_cases=cases,
            binary=binary,
            oversized=oversized,
        )

    product_path: str | None = None
    if old_kind == "product" or new_kind == "product":
        product_path = change.display_path

    return test_metric, inline_metric, product_path


def summarize_diff(
    root: Path,
    *,
    base: str | None = None,
    profile_name: str | None = None,
) -> DiffSummary:
    config, selected_profile, _ = load_policy(root, profile_name)
    base_commit, base_label = resolve_base(root, base)
    baseline_has_test_suite = bool(base_commit) and has_test_suite(root, config, base_commit)
    changes = collect_changes(root, base_commit)
    if len(changes) > config["max_changed_files"]:
        raise TestPolicyError(
            f"{len(changes)} changed files exceed max_changed_files={config['max_changed_files']}; "
            "review this as an explicit broad change"
        )

    product_files: list[str] = []
    test_files: list[FileMetrics] = []
    inline_test_files: list[FileMetrics] = []
    test_infrastructure_files: list[FileMetrics] = []
    snapshot_files: list[FileMetrics] = []
    fixture_files: list[FileMetrics] = []

    for change in changes:
        old_kind = classify_path(change.old_path or change.path, config)
        new_kind = "deleted" if change.status.startswith("D") else classify_path(change.path, config)
        if old_kind in {"ignored", "support"} and new_kind in {"ignored", "support", "deleted"}:
            continue

        metric, inline_metric, product_path = _metric_for_change(root, change, base_commit, config)
        if product_path:
            product_files.append(product_path)
        if inline_metric:
            inline_test_files.append(inline_metric)
        if not metric:
            continue
        if metric.kind in TEST_KINDS:
            test_files.append(metric)
        elif metric.kind == TEST_INFRASTRUCTURE_KIND:
            test_infrastructure_files.append(metric)
        elif metric.kind == "snapshot":
            snapshot_files.append(metric)
        elif metric.kind == "fixture":
            fixture_files.append(metric)

    return DiffSummary(
        profile=selected_profile,
        base=base_label,
        baseline_has_test_suite=baseline_has_test_suite,
        changed_paths=tuple(change.display_path for change in changes),
        product_files=tuple(sorted(set(product_files))),
        test_files=tuple(test_files),
        inline_test_files=tuple(inline_test_files),
        test_infrastructure_files=tuple(test_infrastructure_files),
        snapshot_files=tuple(snapshot_files),
        fixture_files=tuple(fixture_files),
    )


def _limit(violations: list[str], actual: int, allowed: int, label: str) -> None:
    if actual > allowed:
        violations.append(f"{label}: {actual} exceeds the {allowed} limit")


def _violations_for_summary(
    summary: DiffSummary,
    profile: dict[str, Any],
    *,
    patch_local: bool,
) -> list[str]:
    violations: list[str] = []
    _limit(
        violations,
        summary.test_files_touched,
        profile["max_test_files_touched"],
        "test files touched",
    )
    _limit(violations, summary.new_test_files, profile["max_new_test_files"], "new test files")
    _limit(
        violations,
        summary.deleted_test_files,
        profile["max_deleted_test_files"],
        "deleted test files",
    )
    _limit(
        violations,
        summary.added_test_lines,
        profile["max_added_test_lines"],
        "added test lines",
    )
    _limit(
        violations,
        summary.added_test_cases,
        profile["max_added_test_cases"],
        "added test cases",
    )
    _limit(
        violations,
        len(summary.inline_test_files),
        profile["max_inline_test_files_touched"],
        "production files receiving inline tests",
    )
    _limit(
        violations,
        len(summary.test_infrastructure_files),
        profile["max_test_infrastructure_files_touched"],
        "test-infrastructure files touched",
    )
    _limit(
        violations,
        summary.new_test_infrastructure_files,
        profile["max_new_test_infrastructure_files"],
        "new test-infrastructure files",
    )
    _limit(
        violations,
        summary.added_test_infrastructure_lines,
        profile["max_added_test_infrastructure_lines"],
        "added test-infrastructure lines",
    )
    _limit(
        violations,
        len(summary.expensive_test_files),
        profile["max_expensive_test_files_touched"],
        "integration/E2E/system test files touched",
    )
    _limit(
        violations,
        summary.new_expensive_test_files,
        profile["max_new_expensive_test_files"],
        "new integration/E2E/system test files",
    )
    _limit(
        violations,
        summary.added_expensive_test_lines,
        profile["max_added_expensive_test_lines"],
        "added integration/E2E/system test lines",
    )
    _limit(
        violations,
        len(summary.snapshot_files),
        profile["max_snapshot_files_touched"],
        "snapshot files touched",
    )
    _limit(
        violations,
        summary.new_snapshot_files,
        profile["max_new_snapshot_files"],
        "new snapshot files",
    )
    _limit(
        violations,
        summary.added_snapshot_lines,
        profile["max_added_snapshot_lines"],
        "added snapshot lines",
    )
    for item in summary.snapshot_files:
        _limit(
            violations,
            item.added_lines,
            profile["max_snapshot_added_lines_per_file"],
            f"added snapshot lines in {item.display_path}",
        )
    _limit(
        violations,
        len(summary.fixture_files),
        profile["max_fixture_files_touched"],
        "fixture/golden files touched",
    )
    _limit(
        violations,
        summary.new_fixture_files,
        profile["max_new_fixture_files"],
        "new fixture/golden files",
    )
    _limit(
        violations,
        summary.added_fixture_lines,
        profile["max_added_fixture_lines"],
        "added fixture/golden lines",
    )

    for item in summary.all_test_metrics:
        if item.oversized:
            violations.append(
                f"{item.display_path} exceeds max_inspected_file_bytes and needs explicit review"
            )
        if item.binary and item.kind in ARTIFACT_KINDS and not profile["allow_binary_test_artifacts"]:
            violations.append(
                f"binary {item.kind} change {item.display_path} requires an explicit exception"
            )

    if (
        not patch_local
        and summary.has_test_change
        and profile["require_product_change"]
        and not summary.product_files
    ):
        violations.append(
            f"test changes have no product-code change; use {PROFILE_ENV}=tests-only for intentional test maintenance"
        )

    if (
        summary.has_test_change
        and not summary.baseline_has_test_suite
        and (summary.new_test_files or summary.added_test_cases)
        and not profile["allow_new_test_suite"]
    ):
        violations.append(
            "the baseline has no test suite; bootstrapping one requires explicit human ownership "
            f"or {BYPASS_ENV}=1"
        )

    # Stable, non-duplicated messages make hook output and CI easier to read.
    return list(dict.fromkeys(violations))


def evaluate_summary(
    root: Path,
    summary: DiffSummary,
    *,
    patch_local: bool = False,
    profile_name: str | None = None,
) -> PolicyReport:
    _, selected_profile, profile = load_policy(root, profile_name or summary.profile)
    violations = tuple(_violations_for_summary(summary, profile, patch_local=patch_local))
    if os.environ.get(BYPASS_ENV) == "1":
        return PolicyReport(
            ok=True,
            profile=selected_profile,
            summary=summary,
            overridden=True,
            would_violate=violations,
        )
    return PolicyReport(
        ok=not violations,
        profile=selected_profile,
        summary=summary,
        violations=violations,
    )


def evaluate_repository(
    root: Path,
    *,
    base: str | None = None,
    profile_name: str | None = None,
) -> PolicyReport:
    summary = summarize_diff(root, base=base, profile_name=profile_name)
    return evaluate_summary(root, summary, profile_name=profile_name)


def _added_lines_from_patch(body: str) -> list[str]:
    return [
        line[1:]
        for line in body.splitlines()
        if line.startswith("+") and not line.startswith("+++")
    ]


def _deleted_lines_from_patch(body: str) -> int:
    return sum(
        1
        for line in body.splitlines()
        if line.startswith("-") and not line.startswith("---")
    )


def _patch_blocks(command: str) -> list[tuple[str, str, str | None, str]]:
    matches = list(PATCH_FILE_HEADER.finditer(command))
    blocks: list[tuple[str, str, str | None, str]] = []
    for index, match in enumerate(matches):
        start = match.end()
        end = matches[index + 1].start() if index + 1 < len(matches) else len(command)
        body = command[start:end]
        old_path = normalize_path(match.group(2))
        move = PATCH_MOVE_TO.search(body)
        new_path = normalize_path(move.group(1)) if move else old_path
        blocks.append((match.group(1), new_path, old_path if move else None, body))
    return blocks


def _coalesced_patch_blocks(command: str) -> list[tuple[str, str, str | None, str]]:
    coalesced: list[tuple[str, str, str | None, str]] = []
    indexes: dict[str, int] = {}
    for action, path, moved_from, body in _patch_blocks(command):
        if moved_from:
            coalesced.append((action, path, moved_from, body))
            continue

        existing_index = indexes.get(path)
        if existing_index is None:
            indexes[path] = len(coalesced)
            coalesced.append((action, path, None, body))
            continue

        previous_action, _, _, previous_body = coalesced[existing_index]
        if previous_action == "Add":
            combined_action = "Delete" if action == "Delete" else "Add"
        elif action == "Delete":
            combined_action = "Delete"
        else:
            combined_action = "Update"
        coalesced[existing_index] = (
            combined_action,
            path,
            None,
            f"{previous_body}\n{body}",
        )
    return coalesced


def summarize_patch(
    root: Path,
    command: str,
    *,
    profile_name: str | None = None,
) -> DiffSummary:
    config, selected_profile, _ = load_policy(root, profile_name)
    baseline_has_test_suite = _head_exists(root) and has_test_suite(root, config, "HEAD")
    product_files: list[str] = []
    test_files: list[FileMetrics] = []
    inline_test_files: list[FileMetrics] = []
    test_infrastructure_files: list[FileMetrics] = []
    snapshot_files: list[FileMetrics] = []
    fixture_files: list[FileMetrics] = []
    changed_paths: list[str] = []

    for action, path, moved_from, body in _coalesced_patch_blocks(command):
        status = {"Add": "A", "Update": "R" if moved_from else "M", "Delete": "D"}[action]
        old_path = moved_from or path
        old_kind = classify_path(old_path, config)
        new_kind = "deleted" if action == "Delete" else classify_path(path, config)
        added = _added_lines_from_patch(body)
        deleted = _deleted_lines_from_patch(body)
        cases = count_test_cases(added)
        changed_paths.append(f"{old_path} -> {path}" if moved_from else path)

        if old_kind == "product" or new_kind == "product":
            product_files.append(changed_paths[-1])

        category_kind: str | None = None
        if new_kind in TEST_BEARING_KINDS:
            category_kind = new_kind
        elif old_kind in TEST_BEARING_KINDS:
            category_kind = old_kind

        if category_kind:
            category = TEST_KINDS if category_kind in TEST_KINDS else {category_kind}
            metric = FileMetrics(
                path=path,
                old_path=moved_from,
                status=status,
                kind=category_kind,
                is_new=_category_new(status, old_kind, new_kind, category),
                is_deleted=_category_deleted(status, old_kind, new_kind, category),
                added_lines=len(added),
                deleted_lines=deleted,
                added_test_cases=cases if category_kind in TEST_KINDS else 0,
            )
            if category_kind in TEST_KINDS:
                test_files.append(metric)
            elif category_kind == TEST_INFRASTRUCTURE_KIND:
                test_infrastructure_files.append(metric)
            elif category_kind == "snapshot":
                snapshot_files.append(metric)
            elif category_kind == "fixture":
                fixture_files.append(metric)
        elif new_kind == "product" and cases:
            inline_test_files.append(
                FileMetrics(
                    path=path,
                    old_path=moved_from,
                    status=status,
                    kind="inline_test",
                    is_new=False,
                    is_deleted=False,
                    added_lines=0,
                    deleted_lines=deleted,
                    added_test_cases=cases,
                )
            )

    return DiffSummary(
        profile=selected_profile,
        base=None,
        baseline_has_test_suite=baseline_has_test_suite,
        changed_paths=tuple(changed_paths),
        product_files=tuple(sorted(set(product_files))),
        test_files=tuple(test_files),
        inline_test_files=tuple(inline_test_files),
        test_infrastructure_files=tuple(test_infrastructure_files),
        snapshot_files=tuple(snapshot_files),
        fixture_files=tuple(fixture_files),
    )


def evaluate_patch(
    root: Path,
    command: str,
    *,
    profile_name: str | None = None,
) -> PolicyReport:
    """Preflight one apply_patch request without rejecting valid TDD ordering.

    Product/test co-location is checked after the patch against the cumulative Git
    diff. The preflight only rejects a single patch that already exceeds mechanical
    authoring limits.
    """

    summary = summarize_patch(root, command, profile_name=profile_name)
    return evaluate_summary(
        root,
        summary,
        patch_local=True,
        profile_name=profile_name,
    )


def profile_details(root: Path, profile_name: str | None = None) -> dict[str, Any]:
    config, selected, profile = load_policy(root, profile_name)
    return {
        "profile": selected,
        "limits": profile,
        "available_profiles": sorted(config["profiles"]),
        "policy_path": POLICY_PATH,
    }
