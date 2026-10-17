---
name: git-flow
description: "Trigger: Git workflow, commit, branch, push, pull request, CHANGELOG, README, live documentation. Enforce safe repository changes and pre-commit documentation checks."
license: MIT
metadata:
  author: "workshop-golang"
  version: "1.0"
---

## SKILL-43 Activation Contract

Load for Git operations, delivery preparation, commit planning, or changes that may affect repository documentation and history.

## SKILL-44 Hard Rules

- Run `git status --short --branch` before changing files and preserve unrelated user changes.
- Do not mix work units or stage files outside the current task.
- `git commit` is forbidden until the user separately authorizes the proposed commit(s). "Implement", "continue", "go ahead", task approval, or approval to modify files do not authorize commits.
- Require explicit authorization before merge, push, tag, release, reset, clean, or history rewrite.
- Before committing, review `README.md`, `CHANGELOG.md`, and relevant status, task, feature, or capability documents; update stale user-visible claims first.
- Use concise Conventional Commit messages in English. Verify the diff and relevant checks before delivery.

## SKILL-45 Decision Gates

| Condition | Action |
| --- | --- |
| Unrelated worktree changes | Preserve them and isolate the task files. |
| Living documentation is stale | Update it before proposing the commit. |
| Commit is not authorized | Report the proposed files, message, checks, and commit count; stop and wait for separate explicit authorization. |
| Remote or history-changing operation requested | Confirm that exact operation separately. |

## SKILL-46 Execution Steps

1. Inspect status, diff, recent commit style, and the current work unit.
2. Review and synchronize living documentation before staging.
3. Run focused checks and `git diff --check`.
4. Report proposed files, checks, rollback boundary, proposed commit message, and commit count.
5. Stop and wait for separate explicit commit authorization. After authorization, stage exact paths, commit once per coherent unit, and verify status and commit contents.

## SKILL-47 Output Contract

Return the worktree state, changed paths, documentation review, checks and results, commit authorization state, and any remote action still pending.

## SKILL-48 References

- [Repository Git contract](../../CONTRATO_FLUJO_GIT_VERSIONADO.md)
