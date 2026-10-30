# Tools

Procedures agents turned into code: the law of this folder and the catalog of what exists. When to reuse or create a tool is in the root `AGENTS.md`.

## Contract

- Go, standard library only: one module at `.agents/tools/go.mod`, no dependency, no `go.sum` entry, no committed binary. Go 1.22 or newer.
- One directory per tool: `.agents/tools/<name>/main.go`, `<name>` in lower kebab-case. Code two tools share lives in `.agents/tools/internal/`.
- Run from the repository root as `go run -C .agents/tools ./<name> [flags]` (`-C` resolves against the working directory). A tool resolves the repository root by walking up from its working directory to the directory that holds `AGENTS.md` and `perspectives/`; `--root <dir>`, absolute, overrides it. Paths it takes and prints are relative to that root.
- Read-only, with one exception: `metadata` writes `.metadata.json` and nothing else. No other tool writes; the skill that owns a write performs it.
- `main.go` opens with a comment: the question the tool answers, its flags, and its output.
- Output is plain text an agent reads as is. Exit 0 with the answer; non-zero with the reason on stderr when it cannot answer (`go run` reports every failure as 1).
- A tool that implements a rule follows it: where they disagree, the rule wins and the tool is fixed in the same session. A tool that parses a format defined in an `AGENTS.md` changes in the same session as that format.
- `go vet ./...` and `go test ./...` in `.agents/tools` are green before a tool is committed. Tests sit beside the code they cover, standard `testing` only.
- Obsolete → delete the directory and its catalog row.

## Catalog

| Tool | Answers | Usage |
|------|---------|-------|
| `metadata` | Does the `.metadata.json` of each node match its content? It also writes it: `--write` refreshes a node's own hash and rules, `--sync <dir>` also records the current version of its sources, `--staged` serves the pre-commit hook | `go run -C .agents/tools ./metadata [--write \| --sync \| --staged] [dir...]` |
| `pending` | What work is pending, for whom: the specs `/design` owes, the code `/implement` owes with the rules that changed, the gaps `TODO.md` parks, the next `/implement continue` batch, and the drift | `go run -C .agents/tools ./pending [--batch]` |
