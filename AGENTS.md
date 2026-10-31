# UNIVERSAL RULES

THESE RULES APPLY IN EVERY SITUATION:

- **DO NOT BE VERBOSE**
- **BE DIRECT**
- **DO NOT SCATTER INFORMATION ACROSS MULTIPLE PLACES; BE CONCISE AND CLEAR**
- **DO NOT BE REDUNDANT**
- **DO NOT REPEAT YOURSELF**
- **WRITE A LITTLE AND CLEARLY**

THE RULES ABOVE ARE NON-NEGOTIABLE.

---

# SDD framework

This repository is the framework template, not a product. A clone becomes a project through `TEMP_BOOTSTRAP.md`; read it only for bootstrap work.

## Tree

| Path | Owns |
|------|------|
| `.sdd/templates/` | Everything a project receives |
| `.agents/tools/` | Framework guard and a symlink to the templates' rules tool; catalog in its `AGENTS.md` |
| `.agents/skills/` | Sandbox skill and a symlink to the templates' commit skill |
| `.claude/skills/` | Skill symlinks |
| `.githooks/pre-commit` | Guard entry point |
| `.sdd/sandbox/<name>.md` | Tracked sandbox brief |
| `.sdd/sandbox/<name>/` | Ignored worktree on `sandbox/<name>` |

## Templates

Edit project-facing framework files only in `.sdd/templates/`. Read its laws as if it were a repository root: paths in those laws are relative to it. Bootstrap adapts the default layout and perspectives.

Keep this root's UNIVERSAL RULES as its sole editable source. Editor copies and regeneration follow **Harnesses** in the template root law, with commands run from this root; templates retain their own project arrangement.

The templates' `.gitignore` applies live here: never add a pattern hiding template files.

Validate with `go run -C .agents/tools ./guard --check`. Tool contracts and checks belong to their catalogs.

## Version

The hook increments the integer in `.sdd/templates/.sdd/.sdd-config.json` on each main commit, making each version identify one commit. Never edit it manually or commit to main with `--no-verify` or `--amend`. Enable hooks once per clone: `git config core.hooksPath .githooks`.

## Sandboxes

`/sdd-sandbox <name>` creates `.sdd/sandbox/<name>/`, a git worktree on `sandbox/<name>`, bootstraps the framework there for the product of `.sdd/sandbox/<name>.md` through `TEMP_BOOTSTRAP.md`, and pushes the branch. The flow — `/sdd-brainstorming`, `/sdd-spec`, `/sdd-design`, `/sdd-implement`, `/sdd-audit`, `/sdd-commit` — runs in an agent session opened inside the worktree, whose `.claude/skills/` is the materialized one, answering the interviews from the brief's **Decisions**. A sandbox is a consumer: a defect found there is fixed in `.sdd/templates/` on `main` and brought in with `/sdd-update-framework --from ../../..`. The first sandbox is `finance`.

## Commits

Use `/sdd-commit`; a template path takes the type of the path it becomes in a project, and `TEMP_BOOTSTRAP.md` is bootstrap. Scope `templates` for project-facing changes; `tools`, `skills`, or `hooks` for this root. Use `!` when updates conflict with project adaptations through path renames, removed skills, or changed formats.
