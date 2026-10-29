---
name: brainstorming
description: >-
  Refines one or more ideas against the current architecture and the specs
  in perspectives/ — framing, holes, conflicts, integration — questioning
  the user on every decision, and optionally materializes the result in
  .sdd/brainstorming/<name>.md for /spec and /prototype. Use when the user
  wants to discuss, refine, or stress-test an idea or feature before it
  becomes a spec, or types /brainstorming.
---

# brainstorming

Reach `perspectives/AGENTS.md` and `src/AGENTS.md` before framing the idea. Spec law, perspectives, layers, and postulates live in the first; the source tree and implementation postulates in the second. The working-file law of `.sdd/` is in the root `AGENTS.md`.

A brainstorm is a draft, not a rule. It never overrides `perspectives/`; it prepares the request `/spec` writes and the idea `/prototype` tries.

## User Input

```text
$ARGUMENTS
```

## Arguments

```text
<arg> ::= [<name>:]<idea> | <name>
<name> ::= lower kebab-case name of a brainstorm in .sdd/brainstorming/
<idea> ::= the idea or ideas to refine
```

Examples: `/brainstorming invite links that expire`; `/brainstorming invite-expiry:add a grace period`; `/brainstorming invite-expiry`.

A bare `<name>` that matches `.sdd/brainstorming/<name>.md` loads that brainstorm and refines it. `<name>:<idea>` refines it with the new idea. Otherwise the argument is a new idea. **No argument = stop.** Ask for the idea.

## Steps

**Done when:** every hole found is decided or listed under `Open questions`; the user has decided whether to materialize; if yes, the file is written to **Shape**.

1. **Reach** both `AGENTS.md` files, the nested `AGENTS.md` of every perspective the idea touches, `CONTEXT.md`, and `TODO.md`. Load the brainstorm when `<name>` names one.
2. **Discover** in the repo: specs under `perspectives/`, code under `src/`, `package.json`, and the other files in `.sdd/brainstorming/` (overlap). Facts are not questions. Decisions are the user's.
3. **Frame** the idea: problem, goal, non-goals. Several ideas → frame each, then name how they relate.
4. **Challenge** until shared understanding. One question at a time, with a recommendation. Walk every branch; if an answer opens a new one, follow it. Hunt for:
   - flow holes and undefined real paths;
   - omitted exceptions, limits, and failure cases;
   - conflict with an existing rule — cite its title and path;
   - conflict with a `CONTEXT.md` term;
   - overlap with a `TODO.md` item or another brainstorm;
   - fit with the current architecture and stack.
5. **Map** the integration: which perspectives, features, and global rules the idea touches; the atomic candidate items, each with a preliminary owner (perspective/feature) and layer (policy or mechanism) by the layer test of that perspective's `AGENTS.md`.
6. **Offer:** summarize, then ask "Shall I materialize this brainstorm in `.sdd/brainstorming/<name>.md`?". No → stop without writing.
7. **Write** to **Shape**. Name: the idea's name in lower kebab-case ASCII, diacritics stripped. A file of that name holding another idea → ask: overwrite, merge, or another name.
8. **Report:** path written, open questions left, and next steps (`/prototype <name>`, `/spec brainstorming:<name>`).

### Interview

```markdown
**Question:** [one decision]
**Recommendation:** [option + why + cost of the alternative]
**Context:** [fact already found, if any]
```

Conflict with a rule: name the title, the path, the clash, and the recommended resolution. Glossary conflict: point at the `CONTEXT.md` entry and the intended wording.

## Shape

The file is written in the language of the conversation. Rule titles, `CONTEXT.md` terms, and paths are quoted verbatim. Omit empty sections.

```markdown
# <Idea name>

<one purpose sentence>

## Problem
## Goal
## Non-goals
## Decisions
- <decision> — why: <reason>; rejected: <alternative and its cost>
## Flows
## Edge cases
## Integration
- **<Rule title>** (`perspectives/<perspective>/<feature>/policy.md`): <how it is touched; conflict and resolution>
## Candidate items
- <atomic item> — `<perspective>/<feature>`, <policy|mechanism>
## Glossary candidates
## Open questions
```

`Decisions` holds only what the user decided. A recommendation the user did not accept is not a decision; it stays an open question.

Permissions: create, edit, and delete files under `.sdd/brainstorming/`. Out of scope: spec files, tracking, `CONTEXT.md`, `TODO.md`, code under `src/`. Specs are `/spec`; code is `/implement` or `/prototype`.
