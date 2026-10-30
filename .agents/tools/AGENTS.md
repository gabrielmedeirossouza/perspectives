# Tools

Tools of the framework repository itself. The templates' tools live in `.sdd/templates/.agents/tools/`, and their law — Go, standard library only, one directory per tool, a comment that opens `main.go` with the question, flags, and output, plain-text output, tests beside the code, `go vet` and `go test` green before a commit — applies here too.

| Tool | Answers | Usage |
|------|---------|-------|
| `guard` | Is this commit of the framework sound, and which version does it make? Vets and tests the templates' module and this one; on `main`, increments `version` in `.sdd/templates/.sdd/.sdd-config.json` and stages it. The pre-commit hook runs it; `--check` only tests and prints the version | `go run -C .agents/tools ./guard [--check] [--root <dir>]` |
