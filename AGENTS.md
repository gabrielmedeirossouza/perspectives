# SDD framework

This repository is the spec-driven development (SDD) framework as a template. Nothing at this root is a project. A project is made by cloning this repository and following `TEMP_BOOTSTRAP.md`, which materializes `.sdd/templates/` at the project's root, adapted to that product, and deletes itself.

## Tree

```
AGENTS.md                 # this file: law of the framework repository
TEMP_BOOTSTRAP.md         # the prompt a project follows after cloning; not read for framework work
.githooks/pre-commit      # shim: exec go run -C .agents/tools ./guard — tests the templates, bumps the version on main
.agents/tools/            # Go module `framework`: the tools of this repository only (guard); catalog in its AGENTS.md
.agents/skills/           # skills of this repository only: sdd-sandbox; sdd-commit is a symlink into the templates
.claude/skills/           # symlinks to .agents/skills/, as the templates' AGENTS.md says
.sdd/templates/           # the framework: everything a project receives, laid out as the project will hold it
.sdd/sandbox/<name>.md    # brief of a sandbox: the product, the bootstrap answers, the decisions its interviews get
.sdd/sandbox/<name>/      # git worktree on branch sandbox/<name>, ignored
```

## Templates

`.sdd/templates/` is the framework. Its law is its own `AGENTS.md` and the nested ones, read as if `.sdd/templates/` were a repository root: a path they name, such as `specs/AGENTS.md` or `.agents/tools/`, is relative to it. Edit the framework there and only there. The templates carry the default layout — `specs/`, `src/`, `src/specs/`, the perspectives business, user, and operator — and `TEMP_BOOTSTRAP.md` says what a project adapts.

The templates' `.gitignore` is live for `.sdd/templates/**` in this repository: never add to it a pattern that would hide a template file.

Tests: `go vet ./... && go test ./...` with `-C .sdd/templates/.agents/tools`, and the same with `-C .agents/tools`. The hook runs both before every commit; `go run -C .agents/tools ./guard --check` runs them by hand.

## Version

`.sdd/templates/.sdd/.sdd-config.json` holds the framework version: an integer the hook increments on every commit to `main`, so a version names one commit and a project can tell what changed since the version it holds. Never edit it by hand; never commit to `main` with `--no-verify` or `--amend` — the first skips the bump, the second repeats it. Enable the hook once per clone: `git config core.hooksPath .githooks`.

## Sandboxes

`/sdd-sandbox <name>` creates `.sdd/sandbox/<name>/`, a git worktree on `sandbox/<name>`, bootstraps the framework there for the product of `.sdd/sandbox/<name>.md` through `TEMP_BOOTSTRAP.md`, and pushes the branch. The flow — `/sdd-brainstorming`, `/sdd-spec`, `/sdd-design`, `/sdd-implement`, `/sdd-audit`, `/sdd-commit` — runs in an agent session opened inside the worktree, whose `.claude/skills/` is the materialized one, answering the interviews from the brief's **Decisions**. A sandbox is a consumer: a defect found there is fixed in `.sdd/templates/` on `main` and brought in with `/sdd-update-framework --from ../..`. The first sandbox is `finance`.

## Commits

`/sdd-commit`, with the types its table gives to the path inside `.sdd/templates/`: `docs` for prose and law, `chore` for skills, tools, hooks, and `TEMP_BOOTSTRAP.md`; scope `templates` for a change a project receives, `tools`, `skills`, or `hooks` for this root. `!` when a project's update will conflict with what it adapted: a renamed path, a removed skill, a changed shape.
