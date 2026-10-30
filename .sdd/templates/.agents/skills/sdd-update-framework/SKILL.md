---
name: sdd-update-framework
description: >-
  Updates this project to a newer version of the SDD framework: reads the
  framework at its remote or a local checkout, has the update tool compute
  what its templates add, change, or remove since the version
  .sdd/.sdd-config.json records, applies it with a three-way merge that keeps
  what the project adapted, resolves the conflicts with the user, and records
  the new version. Use when the user wants to update, upgrade, or sync the
  framework, or types /sdd-update-framework.
---

# sdd-update-framework

Reach the root `AGENTS.md` (**Working files**: `.sdd/.sdd-config.json`) and `.agents/tools/AGENTS.md` (`update`) before running. The deterministic part is the tool's: which framework files changed between the version this project holds and the new one, where each lives in this project, and the three-way merge. The judgment is this skill's: a conflict, a perspective or skill the new version adds, and whether a change the framework made still fits what the project adapted.

## User Input

```text
$ARGUMENTS
```

## Arguments

```text
<arg> ::= [--from <dir>]
<dir> ::= a local checkout of the framework: its root holds .sdd/templates/
```

Examples: `/sdd-update-framework`; `/sdd-update-framework --from ../framework`.

No argument = the `remote` of `.sdd/.sdd-config.json`. `--from` reads a local checkout instead: its working tree is the new version, its history the base.

## Steps

**Done when:** every file the plan names is applied or deliberately kept; no conflict marker is left; `.sdd/.sdd-config.json` records the new version; tests, `metadata`, and `pending` are green; the user has the report.

1. **Stop** when `git status --porcelain` is not empty: an update is reviewed as one diff. Stop when `.sdd/.sdd-config.json` is missing, or has no `remote` and no `--from` was given.
2. **Plan:** `go run -C .agents/tools ./update [--from <dir>]`. Same version on both sides → report and stop. Read every line: `add`, `update`, `merge` are applied as they are; `conflict` needs a decision; `missing` and `skip` are reported, never written; `by hand` names what this skill writes.
3. **Confirm:** show the plan and ask "Shall I apply this update?" Only then continue.
4. **Apply:** `go run -C .agents/tools ./update --apply [--from <dir>]`. The tool writes the files, leaves markers in the conflicts, and records the new version.
5. **Resolve** each conflict in its file, asking when the choice is not obvious:
   - in `PERSPECTIVES.md`, in the `AGENTS.md` of a perspective the project renamed (`name` differs from `from`) or added (no `from`), or on a line the project wrote: keep the project's text and take from the framework only the new structure — a heading, a rule, a pointer;
   - elsewhere: take the framework's text unless the user changed the project's deliberately;
   - a file the framework removed and the project changed: ask; delete it or keep it.
   Leave no `<<<<<<<`, `=======`, `>>>>>>>` behind.
6. **Write by hand** what the plan asked: the `AGENTS.md` of a perspective with no `from`, by analogy to a shipped one; a skill the plan reports as not listed, when the user wants it now — add it to `skills` of the config and run step 4 again; a perspective kind the new version ships (`skip … no perspective from <kind>`), when this product has that actor — add its entry to `perspectives` with that `from`, run step 4 again, and adapt its `AGENTS.md` and its bullet as **Perspectives** of `specs/AGENTS.md` says.
7. **Verify:** `go vet ./... && go test ./...` in `.agents/tools`; `go run -C .agents/tools ./metadata` prints only `ok` lines — or `no node under …` while no spec exists; `go run -C .agents/tools ./pending` prints what it printed before the update; every symlink under `.claude/skills/` resolves; `.githooks/pre-commit` is executable; no conflict marker under the project.
8. **Report:** versions from and to, files by verb, each conflict and its resolution, what was kept against the framework and why, and the commit to make: `/sdd-commit` with `chore: update sdd framework to v<M>`.

Permissions: run the update tool; edit the files it wrote, to resolve conflicts; write the `AGENTS.md` of a new perspective, and `skills` and `perspectives` of `.sdd/.sdd-config.json`. Out of scope: product specs, technical specs, code, `.metadata.json`, `CONTEXT.md`, `TODO.md`; `version` of the config by hand.
