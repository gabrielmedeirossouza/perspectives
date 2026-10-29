---
name: commit
description: >-
  Reads the working tree changes, groups them into atomic Conventional Commits
  whose types map to SemVer impact, commits them, and pushes the current
  branch. Use when the user wants to commit and push the current work, or
  types /commit.
---

# commit

Reach `git status` and the diffs before writing any message. The message is derived from what the diff does, never from file names alone or from the session's narrative.

Invoking `/commit` authorizes the commits and the push of the current branch. Do not ask again.

## User Input

```text
$ARGUMENTS
```

## Arguments

```text
<arg> ::= [<hint>]
<hint> ::= free text about the intent of the changes
```

Examples: `/commit`; `/commit remove review skills and TDD`. The hint informs the messages; the diff still decides type and scope. No argument = commit every change in the working tree.

## Steps

**Done when:** every change in the working tree is in exactly one commit; every message follows **Message**; the current branch is pushed; the user has the report.

1. **Read** the state: `git branch --show-current`, `git status --porcelain`, `git diff`, `git diff --staged`, untracked file contents, and `git log --oneline -n 20`.
2. **Stop** and report when: there is nothing to commit; HEAD is detached; the current branch is `prototype/*` (prototypes stay local; `/prototype` commits them); a merge, rebase, or cherry-pick is in progress; a file has conflict markers; a file carries a secret or matches an ignored-secret pattern (`.env`, `*.env`, keys, tokens) without being ignored.
3. **Group** the changes into atomic commits: one logical change per commit, one type per commit. A file belongs to one commit; a file with mixed changes goes with its main change. Order the commits so none depends on a file a later commit adds.
4. **Classify** each group by **Type** and name its scope.
5. **Commit** each group in order: `git add -- <paths>` with explicit paths (deletions included), then `git commit -m "<header>" -m "<body>"`. Never `git add -A`, `git add .`, `--amend`, or `--no-verify`. A failing hook → stop and report; do not bypass it.
6. **Push** the current branch: `git push` when it has an upstream, otherwise `git push -u origin <branch>`. Never `--force` or `--force-with-lease`. A rejected push → stop and report; do not pull, rebase, or merge.
7. **Report**: branch, each commit (short hash and header), push result, and the SemVer impact of the set — the highest of its commits.

Out of scope: version tags, version-file bumps, changelogs, pull requests, history rewriting.

## Type

[Conventional Commits 1.0.0](https://www.conventionalcommits.org/en/v1.0.0/). The type sets the SemVer impact.

| Type | When the group | SemVer |
|------|----------------|--------|
| `feat` | adds observable behavior under `src/` | minor |
| `fix` | corrects wrong behavior under `src/` | patch |
| `perf` | makes existing behavior faster without changing it | patch |
| `refactor` | restructures code without changing behavior | none |
| `test` | only adds or changes tests | none |
| `docs` | changes specs, tracking, `CONTEXT.md`, `TODO.md`, `AGENTS.md`, brainstorms under `.sdd/brainstorming/`, or other prose | none |
| `build` | changes dependencies, manifests, or build configuration | none |
| `ci` | changes CI configuration | none |
| `chore` | changes agent skills under `.agents/skills/` or other harness tooling | none |

A change that breaks a public contract — removed or renamed endpoint, field, export, or setting; incompatible behavior — adds `!` after the type or scope and a `BREAKING CHANGE:` footer. SemVer impact: major, whatever the type.

## Message

```text
<type>(<scope>)[!]: <description>

<body>

[BREAKING CHANGE: <what breaks and how to migrate>]
```

- **Scope**: the area of the group — `<perspective>/<feature>` or `<perspective>` for specs, the module for code, `skills` for `.agents/skills/`, `tools` for `.agents/tools/`, `brainstorming` for `.sdd/brainstorming/`. Omit it when the group spans unrelated areas.
- **Description**: English, imperative, lowercase start, no final period, header ≤ 72 characters. Says what the change does, not which files moved.
- **Body**: why the change was made and what it affects, wrapped at 72 columns. Omit it when the header is self-explanatory.

Example, as illustration:

```text
chore(skills)!: remove review skills and verdict panels

Spec and implementation reviews are no longer part of the flow;
/spec and /implement close without independent judges.

BREAKING CHANGE: /spec-review and /implement-review no longer exist.
```
