---
name: sdd-update-framework
description: Update a project from its framework remote or a local checkout, preserving adaptations through a three-way merge. Use to upgrade or sync the SDD framework.
---

# sdd-update-framework

Read root **Working files** and the tool catalog's `update` contract. The tool computes changes; this skill resolves adaptations and conflicts.

## Input

`$ARGUMENTS`: optional `--from <dir>`, a framework checkout; otherwise config `remote`.

## Procedure

1. Require a clean working tree, config, and either remote or `--from`. Save pre-update `metadata` and `pending` output outside the repository.
2. Run `go run -C .agents/tools ./update [--from <dir>]`. Same version → report and stop. Read the plan, including skipped and manual work.
3. Show the concrete plan and ask: “Shall I apply this update?”
4. Run `go run -C .agents/tools ./update --apply [--from <dir>]`.
5. Resolve conflicts under **Adaptations**; complete manual work.
6. Run **Verification** and report versions, changed files, conflict decisions, intentional departures, and findings by owner. Name `/sdd-commit` with `chore: update sdd framework to v<M>` as the next step.

## Adaptations

- Keep product text in the perspective catalog, renamed/custom perspective laws, and deliberately adapted lines; take the framework's new structure and repair its pointers for this layout.
- Elsewhere, take framework text unless it replaces a deliberate project change. Ask when the choice is unclear, or when the framework removed a file the project changed.
- If the user wants a skipped skill or new perspective, add its config selection and rerun Apply. Write a custom perspective's missing law and adapt the catalog under **Perspectives** in `specs/AGENTS.md`.
- Resolve every `conflict` line of the plan, including `kept` ones without markers; leave no conflict markers.

## Verification

Run vet/tests in `.agents/tools`, `metadata`, `pending`, and `rules`. Regenerate editor copies under root **Harnesses** if their source changed. Verify skill symlinks, executable hook, and absence of conflict markers.

`metadata` prints only `ok` or `no node under …`, and `pending` names the same work as the saved baseline, except these new findings:

| New finding | Action |
|-------------|--------|
| A `!` version changed the metadata shape, and `metadata`/`pending` name files to rename, delete, or rewrite | Do exactly that to the named files, then `metadata --write` and recheck. This is the sole allowance for manual metadata migration. |
| Check introduced or widened by the update's States contract or tool changes | Report to the owning skill; leave product specs, code, glossary, and TODO to their owners. |
| Any other new finding | Fix the update's regression before reporting. |

Completion requires the new recorded version, every planned change applied or deliberately kept, and successful verification except findings explicitly handed to owners.

Permissions: update tool; conflict resolution in its written files; generated editor rules; new perspective laws; config skill/perspective selections; required breaking metadata migrations. Never edit `version` manually.
