---
name: prototype
description: >-
  Implements a brainstorm from .sdd/brainstorming/ in a sandbox — branch
  prototype/<name> in a git worktree — without specs, tracking, tests, or
  quality gates, so an idea can be tried before it becomes a spec; with
  --discard, removes the prototype entirely: worktree, branch, and every
  recorded side effect. Use when the user wants to try, prototype, or spike
  an idea, discard a prototype, or types /prototype.
---

# prototype

Reach the root `AGENTS.md` for the working-file law of `.sdd/`, then the brainstorm. A prototype is disposable: nothing it writes is a rule, a tracking entry, or product code on a real branch.

Invoking `/prototype` authorizes the worktree, the branch, and local commits on it. `--discard` still asks once before destroying.

## User Input

```text
$ARGUMENTS
```

## Arguments

```text
<arg> ::= [<name>] | --discard [<name>]
<name> ::= lower kebab-case name of a brainstorm or a prototype
```

Examples: `/prototype`; `/prototype invite-expiry`; `/prototype --discard invite-expiry`.

## Sandbox

```
.sdd/prototypes/<name>/      # git worktree on branch prototype/<name>
.sdd/prototypes/<name>.md    # manifest
```

`.sdd/prototypes/` is git-ignored. The manifest is the only record of the prototype and is self-contained: `--discard` works from it alone, even after `/spec` deleted the brainstorm.

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

**Compensation log:** every side effect outside the worktree's tracked content — container, local database, open port, local env file, generated data outside the worktree — is appended with its undo command **before** it is performed. An effect with no undo command is not performed. A global install, or a remote or real resource (cloud, third-party API that creates data), needs the user's approval first.

## Build

**Done when:** the prototype runs in the worktree; its work is committed on `prototype/<name>`; the manifest lists every side effect; the user has the report.

1. **Resolve** the brainstorm in `.sdd/brainstorming/`: `<name>` given → that file, missing → stop and list the available ones; none exists → stop and suggest `/brainstorming`; one → use it; two or more → ask which.
2. **Resume** when `.sdd/prototypes/<name>.md` exists: keep iterating in its worktree. Otherwise **create**:
   - Stop when HEAD is detached, a merge, rebase, or cherry-pick is in progress, or `prototype/<name>` exists with no manifest (orphan — report it).
   - Warn when the working tree has uncommitted changes: they do not enter the prototype.
   - Write the manifest, then `git worktree add -b prototype/<name> .sdd/prototypes/<name> <base-sha>`.
3. **Implement** the brainstorm inside the worktree only. Read the relevant specs and code so the prototype fits the existing stack and layout. Do not apply: `/spec`, tracking, tests, C1–C8, the Implementation gate, linters or type checks as gates, hooks. Make it run.
4. **Commit** in the worktree: `git -C .sdd/prototypes/<name> add -A`, then `git -C .sdd/prototypes/<name> commit --no-verify -m "chore(prototype): <summary>"`. `--no-verify` is allowed on `prototype/*` only. Never push.
5. **Report:** worktree, branch, how to run it, what was faked or simplified, findings that belong in the brainstorm (`/brainstorming <name>`), and `/prototype --discard <name>`.

Never edit spec files, tracking, `CONTEXT.md`, `TODO.md`, the brainstorm, or any file outside the worktree except the manifest.

## Discard

`--discard [<name>]`.

**Done when:** the worktree, the branch (local and remote), every effect in the Compensation log, and the manifest are gone; the brainstorm is untouched; the user has the report.

1. **Resolve** the manifest in `.sdd/prototypes/`: `<name>` given → that manifest, missing → stop and list; none → stop; one → use it; two or more → ask which.
2. **Inventory and confirm once:** uncommitted changes in the worktree, commits ahead of the base, the remote branch if `git ls-remote --heads origin prototype/<name>` finds one, and each Compensation log entry. Discarding is irreversible.
3. **Leave** the worktree when the session's working directory is inside it.
4. **Compensate** the log in reverse order. A failing undo → stop and report; keep the manifest so the next `--discard` resumes.
5. **Remove** git state: `git worktree remove --force .sdd/prototypes/<name>`, `git worktree prune`, `git branch -D prototype/<name>`, and `git push origin --delete prototype/<name>` when the remote branch exists.
6. **Delete** the manifest.
7. **Report** what was removed.
