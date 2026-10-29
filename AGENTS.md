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

## Tools

`.agents/tools/` holds procedures agents turned into code, so a later session runs them instead of redoing them by hand. Its law and catalog are in `.agents/tools/AGENTS.md`. Reach it before any mechanical procedure over the repository — a scan, a cross-check, a lookup.

- **Reuse.** A catalog entry answers the question → run it; do not redo it by hand. Wrong or incomplete output → fix the tool in the same session; do not work around it.
- **Create.** A session that performs by hand a deterministic procedure — same inputs, same answer, no judgment — that a later session will repeat, such as a skill step or a scan over `perspectives/`, writes it as a tool, runs it against the repository, and adds its catalog row, in the same session.
- **One tool per question.** A neighboring question becomes a flag of the existing tool, not a second tool.
- **Judgment is not a tool.** A recurring procedure that needs judgment goes in the report as a proposed skill change.

Every skill may create and edit files under `.agents/tools/` on top of its own permissions. `/prototype` is the exception: it names tool candidates in its report.
