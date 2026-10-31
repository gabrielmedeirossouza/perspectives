---
name: sdd-prototype
description: Build a disposable brainstorm prototype in a local worktree without specification or quality gates. Use to try an idea or discard its prototype.
---

# sdd-prototype

Read root **Working files** and the source brainstorm. Invocation authorizes the worktree, branch, and local commits. Discard requires its confirmation below.

## Input

`$ARGUMENTS`: `[<name>]` or `--discard [<name>]`; names are lower kebab-case.

Resolve a brainstorm for Build or a manifest for Discard. Use an explicit name; if absent on disk, list candidates and stop. With no name, use the sole candidate or ask which. No candidates → stop, suggesting `/sdd-brainstorming` only for Build.

## Manifest

`.sdd/prototypes/<name>/` is a worktree on `prototype/<name>`; its sibling `<name>.md` is the self-contained cleanup record. Both are git-ignored.

```markdown
# Prototype: <name>

- Brainstorm: `.sdd/brainstorming/<name>.md`
- Base: `<branch>` @ `<sha>`
- Branch: `prototype/<name>`
- Worktree: `.sdd/prototypes/<name>`
- Created: <YYYY-MM-DD>

## Compensation log

1. <side effect> — undo: `<command>`
```

Before any effect outside tracked worktree content, append its undo command: containers, databases, ports, local env files, external generated files. No undo → no effect. Global installs and remote/real resources require user approval.

## Build

1. Resume an existing manifest/worktree. Otherwise reject detached HEAD, active merge/rebase/cherry-pick, or an orphan branch. Warn that uncommitted base changes will not enter the prototype.
2. For a new prototype, write the manifest and run `git worktree add -b prototype/<name> .sdd/prototypes/<name> <base-sha>`.
3. Implement inside the worktree, reading relevant specs/code for fit. Skip tests, C1–C8, Implementation gate, lint/type-check gates, and hooks. Make it run.
4. Commit with `git -C .sdd/prototypes/<name> add -A` and `git -C .sdd/prototypes/<name> commit --no-verify -m "chore(prototype): <summary>"`. This bypass is allowed only on `prototype/*`. Never push.
5. Report worktree, branch, run instructions, simplifications, findings for `/sdd-brainstorming <name>`, and the discard command.

Completion requires a running, committed prototype with every side effect recorded. Do not edit specs, metadata, glossary, TODO, or brainstorm, even inside the worktree. The manifest is the only permitted write outside it.

## Discard

1. From the manifest, inventory uncommitted changes, commits since base, remote branch (check `git ls-remote --heads origin prototype/<name>`), and compensation entries. Show this and confirm once.
2. Leave the worktree if currently inside it.
3. Undo logged effects in reverse order. On failure, stop and retain the manifest for resumption.
4. Remove the worktree with `git worktree remove --force`, prune, delete the local branch with `git branch -D`, and delete the remote branch if present.
5. Delete the manifest after all cleanup succeeds. Report removals; keep the brainstorm.
