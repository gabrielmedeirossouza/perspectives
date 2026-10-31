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

**Done when:** every file the plan names is applied or deliberately kept; no conflict marker is left; `.sdd/.sdd-config.json` records the new version; tests, `metadata`, and `pending` are green — or name only what step 7 hands to another owner; the user has the report.

1. **Stop** when `git status --porcelain` is not empty: an update is reviewed as one diff. Stop when `.sdd/.sdd-config.json` is missing, or has no `remote` and no `--from` was given. Then keep the output of `go run -C .agents/tools ./metadata` and `go run -C .agents/tools ./pending` from before the update, outside the repository: step 7 compares against it.
2. **Plan:** `go run -C .agents/tools ./update [--from <dir>]`. Same version on both sides → report and stop. Read every line: `add`, `update`, `merge` are applied as they are — an `add` of a file the project removed by hand brings it back, because the config is the record of what the project keeps: drop the skill or the perspective in the config to keep it out; `conflict` needs a decision; `skip` is reported, never written; `by hand` names what this skill writes.
3. **Confirm:** show the plan and ask "Shall I apply this update?" Only then continue.
4. **Apply:** `go run -C .agents/tools ./update --apply [--from <dir>]`. The tool writes the files, leaves markers in the conflicts, and records the new version.
5. **Resolve** each conflict in its file, asking when the choice is not obvious:
   - in `PERSPECTIVES.md`, in the `AGENTS.md` of a perspective the project renamed (`name` differs from `from`) or added (no `from`), or on a line the project wrote: keep the project's text and take from the framework only the new structure — a heading, a rule, a pointer;
   - elsewhere: take the framework's text unless the user changed the project's deliberately;
   - a file the framework removed and the project changed: ask; delete it or keep it.
   Leave no `<<<<<<<`, `=======`, `>>>>>>>` behind.
6. **Write by hand** what the plan asked: the `AGENTS.md` of a perspective with no `from`, by analogy to a shipped one; a skill the plan reports as not listed, when the user wants it now — add it to `skills` of the config and run step 4 again; a perspective kind the new version ships (`skip … no perspective from <kind>`), when this product has that actor — add its entry to `perspectives` with that `from`, run step 4 again, and adapt its `AGENTS.md` and its bullet as **Perspectives** of `specs/AGENTS.md` says.
7. **Verify:** `go vet ./... && go test ./...` in `.agents/tools`; `go run -C .agents/tools ./metadata` prints only `ok` lines — or `no node under …` while no spec exists; `go run -C .agents/tools ./pending` prints what it printed before the update — unless the version is marked `!` and changed the shape of the metadata: then both name what to migrate, node by node — a metadata file to rename, delete, or rewrite to the new skeleton — and this session does that migration by hand, the one time metadata is edited by hand, then runs `metadata --write` and the two checks again; or unless the new version checks something the old one did not: a line absent from the output kept in step 1 is a new check's only when its kind is one the update added to or widened in the **States** table of the root `AGENTS.md`, or one a tool the update changed now reports more of; then it is reported for its owner — a spec, code, or `TODO.md` this skill does not edit — and not fixed here; any other new line is this update's own and is fixed before the report; every symlink under `.claude/skills/` resolves; `.githooks/pre-commit` is executable; no conflict marker under the project.
   Also run `go run -C .agents/tools ./rules`: both editor rules must match the root **UNIVERSAL RULES**. If a merge changed the source block, regenerate the copies as **Harnesses** says before checking.
8. **Report:** versions from and to, files by verb, each conflict and its resolution, what was kept against the framework and why, what the new checks found and for whom, and the commit to make: `/sdd-commit` with `chore: update sdd framework to v<M>`.

Permissions: run the update tool; edit the files it wrote, to resolve conflicts; regenerate `.agents/rules/universal.md` and `.cursor/rules/universal.mdc` from the root **UNIVERSAL RULES**; write the `AGENTS.md` of a new perspective, and `skills` and `perspectives` of `.sdd/.sdd-config.json`; rename, delete, or rewrite to the new skeleton the `*.metadata.json` a `!` version's drift names. Out of scope: product specs, technical specs, code, any other metadata edit, `CONTEXT.md`, `TODO.md`; `version` of the config by hand.
