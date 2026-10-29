Perspectives (`perspectives/`) are the product source of truth. Reach `perspectives/AGENTS.md` when the work is product behavior or specs. Reach `src/AGENTS.md` when implementing code against a rule.

## Harnesses

Cursor, Claude Code (v2.1.277+), Codex, and Antigravity (1.20.5+) read this file and every nested `AGENTS.md`. No `CLAUDE.md`, `CLAUDE.local.md`, or `GEMINI.md` exists in this repository: a `CLAUDE.md` or `CLAUDE.local.md` stops Claude Code from reading `AGENTS.md`.

Skills live once, in `.agents/skills/<name>/SKILL.md`. Claude Code reads only `.claude/skills/`, so `.claude/skills/<name>` is a symlink to `../../.agents/skills/<name>`. A new skill gets both; edits go to `.agents/skills/`.

`/<name>` names a skill. Invoke it as `/<name>` in Cursor, Claude Code, and Antigravity, and as `$<name>` in Codex. `$ARGUMENTS` in a skill is the text typed with the invocation; where the harness leaves it literal, read it from the user's message.

A skill that delegates uses the harness's subagent tool: `Task` in Cursor, `Agent` in Claude Code, subagents in Codex, `invoke_subagent` in Antigravity. Diagnostics mean the harness LSP; where the harness has none, the project's type checker and linter.

## Working files

`.sdd/` holds working files of the flow `/brainstorming` → `/prototype` → `/spec` → `/implement`, never rules. `perspectives/` stays the source of truth; a brainstorm never overrides a rule.

- `.sdd/brainstorming/<name>.md`: versioned draft of an idea. `/brainstorming` writes it, `/prototype` reads it, `/spec` consumes and deletes it.
- `.sdd/prototypes/`: git-ignored. One worktree and one manifest per prototype, owned by `/prototype`.
