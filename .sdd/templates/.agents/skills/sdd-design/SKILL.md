---
name: sdd-design
description: Write, change, or remove technical specs in src/specs/. Use for technical decisions, unconsumed product specs, stale or dangling technical nodes, and drift or audit gaps assigned to /sdd-design.
---

# sdd-design

Read **Technical spec** in `src/AGENTS.md`, the spec format and P1–P7 in `specs/AGENTS.md`, and **Metadata** and **Interviews** in the root `AGENTS.md`.

## Input

`$ARGUMENTS`: `[<name>:]<request>`. The technical spec's name is optional; resolve it from the request.

Without an argument, run `go run -C .agents/tools ./pending`, show Design entries, Drift assigned to `/sdd-design`, and Audit gaps from `TODO.md`; ask which to take and stop. Never infer the target from an earlier session or branch.

## Procedure

1. Run `pending`; read the target, its product sources, `CONTEXT.md`, `TODO.md`, relevant code, manifests, and lockfiles.
2. Decompose into decisions. Resolve each technical owner by its code boundary; use a global spec for a constraint independent of features. Park product holes as Spec gaps for `/sdd-spec`; dependent technical rules wait.
3. Interview, naming each indirection's pain trigger.
4. Confirm: “Shall I apply these changes?”
5. Write the spec and its metadata skeleton under the root contract, or delete a removed spec's directory, leaving its code consumers dangling for `/sdd-implement`. Handle source changes and gaps as below.
6. Sync each completed technical node under root **Writers**.
7. Report written, removed, or revalidated specs, the product rules served, blockers, and product handoffs.

Completion means every agreed item is written or removed, each written spec is synced, product holes are parked, and closed gaps are gone.

### Source changes

- For a dangling source, delete rules exclusive to it, remove its links and source entry; delete the whole spec directory if nothing remains.
- Blocking drift reported by `pending` waits for its owner. Report broken links in perspective sources to `/sdd-spec`.

Permissions: technical specs/directories and their skeletons; Spec gaps and resolved Design/Audit gaps in `TODO.md`; metadata operations except `--approve`. Product specs, code, and glossary remain with their owners.
