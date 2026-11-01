---
name: sdd-commit
description: Group working-tree changes into atomic Conventional Commits and push the current branch. Use when the user asks to commit and push or invokes /sdd-commit.
---

# sdd-commit

Invocation authorizes commits and push; do not ask again.

## Input

`$ARGUMENTS` is an optional intent hint. Messages, types, and scopes describe what the diff does, never file names alone or the session's narrative. Without a hint, include all working-tree changes.

## Procedure

1. Read branch, status, staged/unstaged diffs, untracked contents, and the last 20 commit headers.
2. Stop and report on no changes, detached HEAD, `prototype/*`, an active merge/rebase/cherry-pick, conflict markers, or exposed secrets, including unignored `.env`, `*.env`, key, or token files.
3. Group by logical change, one type per commit, each file in one group. Mixed files follow their main change. Apply **Metadata grouping**; order dependencies first.
4. Stage each group's explicit paths, including deletions: `git add -- <paths>`. Commit with **Message**. Hook-staged files are expected under the repository's law. A failing hook stops the run with a report.
5. Push with `git push`, or `git push -u origin <branch>` without an upstream. Rejection stops the run with a report; do not pull, rebase, or merge.
6. Report branch, commit hashes/headers, push result, and the highest SemVer impact.

Never use broad staging, amend, hook bypass, force push, tags, manual version bumps, changelogs, pull requests, or history rewriting. Completion requires all changes committed and the branch pushed.

## Metadata grouping

| Change | Commit |
|--------|--------|
| Node hash, `adopted`, or `reviewed` changes with content | With that content (and any removed `.sdd/reviews/<id>.json` checkpoint when closing its review) |
| Sync changes only recorded sources (`hash`, `rules`, `adopted`) | Separate `docs(<scope>): acknowledge <source>` |
| Approval, `sources` / `uses` edit, or `.sdd/reviews/<id>.json` checkpoint update | Separate `docs(<scope>): audit sources` (keeping checkpoint verdicts and their applied metadata edits in the same commit) |
| File links change without linked content changes | Separate docs commit, including the resulting hash |
| Previously uncommitted metadata | Whole file with its content |

## Type

Conventional Commits 1.0.0; repository-specific scopes come from root law.

| Type | Change | SemVer |
|------|--------|--------|
| `feat` | New observable code behavior | minor |
| `fix` | Corrected code behavior | patch |
| `perf` | Faster existing behavior | patch |
| `refactor` | Code structure, same behavior | none |
| `test` | Tests only | none |
| `docs` | Specs, law, prose, brainstorms, or metadata-only changes | none |
| `build` | Dependencies, manifests, build configuration | none |
| `ci` | CI configuration | none |
| `chore` | Skills, tools, hooks, bootstrap, other harness tooling | none |

A public-contract break adds `!` and a `BREAKING CHANGE:` footer explaining migration; its impact is major regardless of type.

## Message

```text
<type>(<scope>)[!]: <description>

<optional body>

[BREAKING CHANGE: <break and migration>]
```

- Scope: `<perspective>/<feature>`, `<perspective>`, `src/specs/<name>`, code module, or tooling area (`skills`, `tools`, `hooks`, `brainstorming`). Omit for unrelated areas.
- Description: English imperative, lowercase start, no final period, header ≤ 72 characters.
- Body: why and impact, wrapped at 72 columns; omit when the header suffices.
