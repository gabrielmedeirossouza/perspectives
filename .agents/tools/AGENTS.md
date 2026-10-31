# Tools

Tools of the framework repository itself. The templates' tools live in `.sdd/templates/.agents/tools/`, and their law — Go, standard library only, one directory per tool, a comment that opens `main.go` with the question, flags, and output, plain-text output, tests beside the code, `go vet` and `go test` green before a commit — applies here too, with two differences: a tool of this root finds the framework root by walking up to the directory that holds `.sdd/templates/`, or takes `--root`; and `guard` writes `version` of `.sdd/templates/.sdd/.sdd-config.json` and stages it, the one write of this root.

| Tool | Answers | Usage |
|------|---------|-------|
| `rules` | Do this repository's editor rules match its root UNIVERSAL RULES? Symlink to the templates' tool; `--format` prints a generated copy, otherwise checks both | `go run -C .agents/tools ./rules [--format antigravity\|cursor] [--root <dir>]` |
| `guard` | Is this commit of the framework sound, and which version does it make? Vets and tests the templates' module and this one; on `main`, increments `version` in `.sdd/templates/.sdd/.sdd-config.json` and stages it. The pre-commit hook runs it; `--check` only tests and prints the version | `go run -C .agents/tools ./guard [--check] [--root <dir>]` |
