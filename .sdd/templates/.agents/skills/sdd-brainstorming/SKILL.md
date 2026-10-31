---
name: sdd-brainstorming
description: Refine ideas against existing specs and architecture before specification or prototyping. Optionally save a draft in .sdd/brainstorming/ for /sdd-spec and /sdd-prototype.
---

# sdd-brainstorming

Read `specs/AGENTS.md`, its perspective catalog and relevant perspective laws, `src/AGENTS.md`, and the root **Working files** and **Interviews**.

## Input

`$ARGUMENTS`: `[<name>:]<idea>` or an existing brainstorm's `<name>`.

Load a matching file under `.sdd/brainstorming/`; otherwise treat the argument as a new idea. Without an argument, ask for the idea and stop.

## Procedure

1. Read relevant specs, code, manifests, `CONTEXT.md`, `TODO.md`, and overlapping brainstorms.
2. Frame each idea's problem, goal, and non-goals; explain relationships when several ideas are involved.
3. Interview about undefined flows, failures, exceptions, limits, conflicts, and architectural fit.
4. Map affected rules and atomic candidate items to their owners: `<perspective>/<feature>`, `<perspective>` for a global rule, or `src/specs/<name>`.
5. Summarize and ask: “Shall I materialize this brainstorm in `.sdd/brainstorming/<name>.md`?” Declined → stop without writing.
6. Write the shape below. Use an ASCII lower-kebab-case name, stripping diacritics; if the name holds another idea, ask whether to overwrite, merge, or rename.
7. Report the path, unresolved questions, and next steps: `/sdd-prototype <name>`, `/sdd-spec brainstorming:<name>`, then `/sdd-design`.

## Shape

Use the conversation's language; preserve quoted rule titles, glossary terms, and paths. Omit empty sections.

```markdown
# <Idea name>

<one purpose sentence>

## Problem
## Goal
## Non-goals
## Decisions
- <accepted decision> — why: <reason>; rejected: <alternative and cost>
## Flows
## Edge cases
## Integration
- **<Rule title>** (`<spec path>`): <effect; conflict and resolution>
## Candidate items
- <atomic item> — `<owner>`
## Glossary candidates
## Open questions
```

Only user decisions enter Decisions; unaccepted recommendations remain open questions. Completion means every discovered hole is decided or recorded and the user has chosen whether to save.

Permissions: brainstorm files only.
