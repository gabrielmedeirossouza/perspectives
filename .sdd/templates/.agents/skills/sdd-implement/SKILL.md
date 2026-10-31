---
name: sdd-implement
description: Implement code from technical and product specs, or use continue for the next pending batch. Use for implementation requests naming a technical spec or rule, or to resume the backlog.
---

# sdd-implement

Read **Metadata** in the root `AGENTS.md`, `src/AGENTS.md` (C1–C8 and Implementation gate), and `specs/AGENTS.md` (spec format and gaps). The request selects work; only **Continue** derives it from state.

## Input

`$ARGUMENTS`: `continue` or `<target>[:<comma-separated rule titles>]`. Target is a technical spec's name or `src/specs/<name>`.

`next`, `resume`, `continuar`, `próximo`, and `seguir` also mean `continue`; the keyword wins over a name. Without an argument, ask for a target or `continue` and stop. Ask when a target is ambiguous.

## Procedure

1. Resolve the target, or take the batch **Continue** passes. Read its technical specs, served perspectives, technical dependencies, code consumers and metadata, and glossary.
2. Run `go run -C .agents/tools ./pending` for changed rules and waiting nodes. Apply title filters after resolving this set. A waiting unit does not run; report its blocker and stop.
3. Queue removed rules first, then the pending deltas. For an unconsumed technical spec, queue all its rules and served product rules. Order by citations, product before its technical rule, then file order.
4. Group phases by the owner's `##` section, in bands of 6–10 items; merge only with the next section in that file. One owner per phase; removals go first. Present the plan.
5. Execute **Orchestration** through the Implementation gate. Maintain sources and clean removals under C6.
6. Report completion by unit, tests, synced or unfinished nodes, gaps, and audit work. Include created files and C7 justifications for indirections without repeating the phase log.

Finish when queued rules pass the gate or are parked as Design gaps, and removed rules are cleaned.

## Continue

1. Run `go run -C .agents/tools ./pending --batch`. Units are defined by root **Units**.
2. Take the first 5 runnable units, sources first: each follows the technical specs its nodes realize or depend on, with path order breaking ties. Skip parked rules.
3. If nothing can run, report blockers; if no implementation work remains, report that, what `/sdd-design` owes, and any drift. Stop.
4. Run the Procedure with this batch. Revalidate existing code against deltas before finishing interrupted or stale work.
5. Refresh `pending --batch`; report the next batch and drift for its owners.

## Orchestration

Each phase has a code pass and a test pass. Finish every code pass before any test pass. One phase runs in the parent; otherwise children (root **Harnesses**) each take a whole phase, never a single item, in waves of independent phases with different owners. Same-owner phases run sequentially. Test passes follow the same order and waves.

| Actor | Responsibility |
|-------|----------------|
| Parent | Resolve work, plan phases, run waves, park the gaps children report, and, after all test passes, sync under gate step 5. Then run `pending` once to verify. |
| Code child | Apply gate steps 2–3 to every item; write code and metadata skeletons; refresh metadata. |
| Test child | Apply gate step 4; test files are code nodes with metadata. |
| Either child | Never sync, edit specs/glossary, write TODO, or work outside its phase. Report each item's status, files, C7 justification for new files, and blockers. |

Give each child absolute paths to the root law, `src/AGENTS.md`, tool catalog, target technical/product specs, code and metadata; include testing guidelines for test passes. Specify pass kind, section, ordered titles and states, applicable gate steps, and the restrictions above. Open specs block the item under C8.

Existing-code search may use a read-only subagent; pending and metadata reads stay with the parent. Permissions otherwise follow the Implementation gate.
