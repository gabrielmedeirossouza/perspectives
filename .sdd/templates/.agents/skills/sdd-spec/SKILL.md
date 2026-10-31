---
name: sdd-spec
description: Write, change, or remove product rules in specs/, including perspective changes and consuming brainstorms. Use for product behavior requests or drift assigned to /sdd-spec; route technical decisions to /sdd-design.
---

# sdd-spec

Read `specs/AGENTS.md`, its perspective catalog, and the target perspective's law. Use the root `AGENTS.md`'s **Interviews**.

## Input

`$ARGUMENTS`: `[<perspective-or-feature>:]<request>` or `brainstorming[:<name>]`.

Locate owners from the request; paths are optional. Without an argument, run `go run -C .agents/tools ./pending`, list its Drift entries assigned to `/sdd-spec`, and stop. Never infer a target from an earlier session or branch.

For `brainstorming[:<name>]`, or a request asking to use a brainstorm, resolve one file in `.sdd/brainstorming/`: use the explicit name; if missing, list available files and stop. Without a name, use the sole file, ask when several exist, or suggest `/sdd-brainstorming` when none exists.

## Procedure

1. Read the source brainstorm if any and relevant specs/code; recheck a brainstorm's Integration against current specs.
2. Decompose into atomic items — a brainstorm supplies its Candidate items, Decisions, Flows, and Edge cases — and apply **Writing gate** step 1. Name technical items for `/sdd-design` with the product rules they serve; name unrelated items as out of scope and continue the rest.
3. Interview to close open decisions and conflicts.
4. Confirm: “Shall I apply these changes?”
5. Apply the **Writing gate**, including perspective lifecycle changes when requested. Delete a consumed brainstorm only after every agreed item passes; keep it if declined or interrupted.
6. Report changes by owner, remaining blockers, technical handoffs, and any deleted brainstorm. If its prototype manifest still exists, name `/sdd-prototype --discard <name>` as cleanup; leave the prototype alive.

Permissions are the Writing gate's; never edit a brainstorm.
