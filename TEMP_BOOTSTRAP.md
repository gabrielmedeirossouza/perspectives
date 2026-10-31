# Bootstrap

Run in a fresh framework clone to configure one product. Collect layout and actor choices; product rules, technical decisions (including stack), and code belong to the project's skills afterwards. This file deletes itself on completion.

## Preconditions

Require git, Go 1.22+, a clean working tree, and the clone root containing `.sdd/templates/` with `AGENTS.md`, `specs/`, `src/`, `.agents/`, `.claude/`, `.githooks/`, `.gitignore`, and config.

Determine whether `.git` is a directory (clone) or a file (worktree): a worktree keeps its history, so it skips the History question and the reset in **Finish**. Stop with the reason if a precondition fails.

## Interview

Use **Interviews** in the template root law. Record:

| Choice | Question and recommendation | Config |
|--------|-----------------------------|--------|
| Product | Name and one sentence: what it is, for whom | `product` |
| Perspectives | Show each shipped perspective's **Scope** (`.sdd/templates/specs/<kind>/AGENTS.md`). Keep all by default; offer rename, drop, or additional actors | `perspectives` |
| Paths | Product specs, code, technical specs; default `specs`, `src`, `src/specs` | `paths` |
| Skills | List descriptions from template skills; recommend all. Omitted skills retain references; dropping update removes the project's update path | `skills`; omit for all |
| Remote | Recommend this clone's origin URL; a local path is machine-specific | `remote` |
| History | Fresh history recommended; otherwise retain | — |

Perspective entries are `{"name":"<kebab-case>","from":"<shipped-kind>"}`; omit `from` for a new kind. Ask a new kind's feature boundary, scope, and vocabulary to build its law.

Paths are fixed after bootstrap. Product specs and code are separate; technical specs sit under code or separately, never above it or overlapping product specs. Perspectives and skill selections remain editable.

## Materialize

From the clone root, after the interview:

1. Write `.sdd/.sdd-config.json` with `version: 0` and the recorded choices:

   ```json
   {
     "version": 0,
     "product": "<name and purpose>",
     "remote": "<URL>",
     "paths": { "specs": "<dir>", "src": "<dir>", "technical": "<dir>" },
     "perspectives": [{ "name": "<actor>", "from": "<kind>" }]
   }
   ```

   Add `"skills": ["<name>", …]` only when selecting a subset.

2. Remove framework-only root files: `rm -rf AGENTS.md .gitignore .githooks .agents .claude .cursor`.
3. Run `go run -C .sdd/templates/.agents/tools ./update --apply --root "$(pwd)" --from .`. Explicit root prevents selecting the template config. Read skipped and `by hand` entries.
4. Remove `.sdd/templates` and `.sdd/sandbox`; retain this file until Finish.
5. Enable hooks: `git config core.hooksPath .githooks`.

## Adapt

Use English and the shipped structure, without product rules.

1. Write the perspective catalog in config order, as **Perspectives** in the materialized spec law defines it.
2. Adapt renamed actors' laws and write new kinds' laws as **Perspectives** in the materialized spec law says, using interview answers.
3. Repair examples and pointers naming renamed or dropped perspectives.

## Verify

- Vet and test the materialized `.agents/tools` module.
- `rules` reports matching copies; `metadata` reports `no node under …`; `pending` reports `No pending work.`.
- Every skill symlink resolves; the hook is executable; retained skills match the selection.
- Template/sandbox directories are gone.
- Config records a positive version, remote, selected paths, and perspectives.

Run tools with `go run -C .agents/tools ./<tool>`. Fix failures before Finish.

## Finish

1. For requested fresh history: `rm -rf .git && git init -b main`, then enable hooks again.
2. Delete `TEMP_BOOTSTRAP.md`.
3. Run `git add -A && git commit -m "chore: bootstrap sdd framework v<version>"`. Broad staging is permitted for this initial materialization.
4. Report layout, perspective origins, retained skills, version, and applicable next steps: brainstorm, first spec, later framework updates.

Completion requires successful verification, this file removed, and the bootstrap commit.
