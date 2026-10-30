# Tools

Procedures agents turned into code: the law of this folder and the catalog of what exists. When to reuse or create a tool is in the root `AGENTS.md`.

## Contract

- Go, standard library only: one module at `.agents/tools/go.mod`, no dependency, no `go.sum` entry, no committed binary. Go 1.22 or newer.
- One directory per tool: `.agents/tools/<name>/main.go`, `<name>` in lower kebab-case. Code two tools share lives in `.agents/tools/internal/`.
- Run from the repository root as `go run -C .agents/tools ./<name> [flags]` (`-C` resolves against the working directory: from a subdirectory, give it the path to `.agents/tools` from there). A tool resolves the repository root by walking up from its working directory to the directory that holds `.sdd/.sdd-config.json`; `--root <dir>`, absolute, overrides it. Paths it takes and prints are relative to that root. Where the product specs, the technical specs, and the code live is the `paths` of that file (**Working files** in the root `AGENTS.md`); a tool never hardcodes those roots.
- Read-only, with two exceptions: `metadata` writes `*.metadata.json` and nothing else; `update --apply` writes the files of the framework and `version` of `.sdd/.sdd-config.json` and nothing else. No other tool writes; the skill that owns a write performs it.
- `main.go` opens with a comment: the question the tool answers, its flags, and its output.
- Output is plain text an agent reads as is. Exit 0 with the answer; non-zero with the reason on stderr when it cannot answer (`go run` reports every failure as 1).
- A tool that implements a rule follows it: where they disagree, the rule wins and the tool is fixed in the same session. A tool that parses a format defined in an `AGENTS.md` changes in the same session as that format.
- `go vet ./...` and `go test ./...` in `.agents/tools` are green before a tool is committed. Tests sit beside the code they cover, standard `testing` only.
- Obsolete → delete the directory and its catalog row.

## Catalog

| Tool | Answers | Usage |
|------|---------|-------|
| `metadata` | Does the metadata of each node match its content, does every markdown link of a spec name, case included, a path git tracks or would track and a heading it holds, and does every file under the roots belong to a node? The check lists the drift of each targeted node, then the files at or under the targets in no node and the metadata files that are no node, and fails on any. It also writes it: `--write` refreshes a node's own hash and rules, `--sync <node…>` also records the current version of its sources, `--approve <node…>` adds one approval of `/sdd-audit` to a code node's `audited`, `--staged` serves the pre-commit hook. A target is a node — its path (`specs/business/invite/spec.md`, `src/api/auth/db.ts`) or its metadata file (`src/api/.metadata.json` for a directory node) — or a directory, which names its directory node and every node at or under it; `--sync` and `--approve` take one node per target. `--staged` also refuses a staged file in no node | `go run -C .agents/tools ./metadata [--write \| --sync \| --approve \| --staged] [node\|dir...] [--root <dir>]` |
| `pending` | What work is pending, for whom: the specs `/sdd-design` owes, the units `/sdd-implement` owes — one directory with its pending code nodes and the union of their rules, or one technical spec no code node lists — the gaps `TODO.md` parks, the next `/sdd-implement continue` batch, the code nodes `/sdd-audit` owes — with `--audit`, one dossier per node, grouped by directory for one judge each — and the drift | `go run -C .agents/tools ./pending [--batch \| --audit [--all] [node\|dir...]] [--root <dir>]` |
| `update` | What must change in this project to hold framework version M: the files `.sdd/templates/` of the framework adds, changes, or removes since the version `.sdd/.sdd-config.json` records, each mapped to where this project keeps it by `paths`, `perspectives`, and `skills`; with `--apply`, the change itself — a three-way merge against the base version, conflict markers left in place — and the new version recorded. Version 0 is a bootstrap: every file is an add. The config is the record of what the project keeps: a file removed by hand comes back, a skill or perspective dropped there stays out. `--from <dir>` reads a local checkout of the framework instead of cloning `remote` | `go run -C .agents/tools ./update [--apply] [--from <dir>] [--root <dir>]` |
