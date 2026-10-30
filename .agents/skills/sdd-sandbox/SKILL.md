---
name: sdd-sandbox
description: >-
  Creates, resumes, resets, or discards a sandbox of the framework: a git
  worktree under .sdd/sandbox/<name> on branch sandbox/<name>, where the
  framework is bootstrapped for the product of the brief .sdd/sandbox/<name>.md
  and then exercised with its own skills. Use when the user wants to try the
  framework on a product, validate a change end to end, or types /sdd-sandbox.
---

# sdd-sandbox

Reach the root `AGENTS.md` (**Sandboxes**) and the brief `.sdd/sandbox/<name>.md` before acting. A sandbox is a consumer of the framework, never its source: a defect found there is fixed in `.sdd/templates/` on `main` and brought into the sandbox with `/sdd-update-framework --from ../..`.

Invoking `/sdd-sandbox` authorizes the worktree, the branch, the bootstrap commit, and the push of the branch. `--discard` still asks once before destroying.

## User Input

```text
$ARGUMENTS
```

## Arguments

```text
<arg> ::= <name> | --resume <name> | --reset <name> | --discard <name>
<name> ::= lower kebab-case name of a brief in .sdd/sandbox/
```

Examples: `/sdd-sandbox finance`; `/sdd-sandbox --resume finance`; `/sdd-sandbox --reset finance`; `/sdd-sandbox --discard finance`.

**No argument = stop.** List the briefs and the worktrees (`git worktree list`).

## Brief

`.sdd/sandbox/<name>.md` is tracked and self-contained: a session runs the bootstrap and the skills of the sandbox from it, in the person's place.

```markdown
# Sandbox: <name>

## Product
<one paragraph: what it is, for whom>

## Bootstrap answers
- perspectives: <name (from kind)>, …
- paths: <specs>, <src>, <technical>
- skills: all | <list>
- remote: <the framework's URL>
- history: worktree, never reset

## Decisions
<what the person would answer when a skill interviews: first feature, policies, limits, stack>
```

## Create

**Done when:** the worktree exists on `sandbox/<name>`, the framework is bootstrapped in it for the brief's product, the bootstrap commit is pushed, and the user has the report.

1. **Stop** when the brief is missing (write it first), when `.sdd/sandbox/<name>/` exists (`--resume` or `--reset`), or when branch `sandbox/<name>` exists without a worktree (`--resume`).
2. **Worktree:** `git worktree add -b sandbox/<name> .sdd/sandbox/<name> main`.
3. **Bootstrap** inside `.sdd/sandbox/<name>/`: follow its `TEMP_BOOTSTRAP.md` with the brief's **Bootstrap answers** as the person's; a worktree never resets its history. Run every command with `-C`, or from inside the worktree, never at this root.
4. **Push:** `git -C .sdd/sandbox/<name> push -u origin sandbox/<name>`.
5. **Report:** the worktree, the branch, the framework version bootstrapped, and the next step — open an agent session inside `.sdd/sandbox/<name>/`, whose `.claude/skills/` is the materialized one, and run the flow from the brief's **Decisions**: `/sdd-brainstorming`, `/sdd-spec`, `/sdd-design`, `/sdd-implement`, `/sdd-audit`, `/sdd-commit`.

## Resume

`git fetch origin sandbox/<name>` when only the remote has the branch, then `git worktree add .sdd/sandbox/<name> sandbox/<name>`. Report as in **Create**.

## Reset

**Discard**, then **Create** — a fresh bootstrap from the current `main`.

## Discard

**Done when:** the worktree, the branch (local and remote) are gone; the brief stays; the user has the report.

1. **Inventory and confirm once:** uncommitted changes in the worktree, commits ahead of `main`, and whether `git ls-remote --heads origin sandbox/<name>` finds the branch. Discarding is irreversible.
2. **Leave** the worktree when the session's working directory is inside it.
3. **Remove:** `git worktree remove --force .sdd/sandbox/<name>`, `git worktree prune`, `git branch -D sandbox/<name>`, and `git push origin --delete sandbox/<name>` when the remote has it.
4. **Report** what was removed.

Permissions: git worktrees and branches `sandbox/*`; every file inside the worktree, through `TEMP_BOOTSTRAP.md`; the push of `sandbox/<name>`. Out of scope: `.sdd/templates/` (edited on `main`, by its own law), the brief (the user's), `main`.
