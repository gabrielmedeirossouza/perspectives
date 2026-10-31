# Tools

Follow the contract in `.sdd/templates/.agents/tools/AGENTS.md`. This module finds the framework root by its `.sdd/templates/` directory, or accepts `--root`.

| Tool | Purpose | Invocation |
|------|---------|------------|
| `rules` | Symlink to the templates' editor-rule tool; same arguments and contract | `go run -C .agents/tools ./rules [--format antigravity\|cursor] [--root <dir>]` |
| `guard` | Is this framework commit sound, and which version does it make? Prints `ok <module>` after vetting and testing each Go module. On main, bumps and stages the template config's version, this module's only write (`version N -> N+1, staged`); `--check` or another branch only prints `version N` | `go run -C .agents/tools ./guard [--check] [--root <dir>]` |
