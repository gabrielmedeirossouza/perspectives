# Tools

Procedures agents turned into code: the law of this folder and the catalog of what exists. When to reuse or create a tool is in the root `AGENTS.md`.

## Contract

- One file per tool: `.agents/tools/<name>.mjs`, `<name>` in lower kebab-case. It runs from any directory as `node .agents/tools/<name>.mjs [flags]` and resolves the repository root from its own path.
- Node built-ins only: no dependency, no `package.json`, no build step.
- Read-only: a tool reads and reports. It never writes; the skill that owns a write performs it.
- The file opens with a comment: the question it answers, its flags, and its output.
- Output is plain text an agent reads as is. Exit 0 with the answer; non-zero with the reason on stderr when it cannot answer.
- A tool that implements a rule follows it: where they disagree, the rule wins and the tool is fixed in the same session. A tool that parses a format defined in an `AGENTS.md` changes in the same session as that format.
- Obsolete → delete the file and its catalog row.

## Catalog

| Tool | Answers | Usage |
|------|---------|-------|
| `pending` | Every `[ ]` tracking entry under `perspectives/` with its flags, paths, pending cites, and parked spec gap; drift between spec and tracking; the next `/implement continue` batch | `node .agents/tools/pending.mjs [--batch]` |
