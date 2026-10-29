---
name: implement
description: >-
  Implements product code under src/ from named rules in perspectives/, or,
  with continue, the next batch of the backlog — 1 to 5 units of one
  perspective, in dependency order. Use when the user wants code for a
  feature or rule title, wants to continue implementing what is pending, or
  types /implement.
---

# implement

Reach `perspectives/AGENTS.md` and `src/AGENTS.md` before resolving a target or writing code. Spec law lives in the first; the second defines the source tree, the implementation postulates (C1–C8), and the Implementation gate.

The queue is `$ARGUMENTS`, not the completion mark — except under `continue`, which builds it from the completion marks (**Continue**). `[ ]` means the rule is pending or incomplete; `[x]` means the rule is 100% implemented and validated.

## User Input

```text
$ARGUMENTS
```

## Arguments

```text
<arg> ::= continue | <target>[:<titles>]
<target> ::= perspective/feature | feature | perspective
<titles> ::= comma-separated rule titles
```

Examples: `/implement continue`; `/implement business/users`; `/implement user/home:Empty list`; `/implement users`. A bare perspective (`/implement business`, `/implement development`) targets its global rules.

`continue` also answers to a single word that asks to resume: `next`, `resume`, `continuar`, `próximo`, `seguir`. The keyword wins over a slug; a feature of that name is reached by `perspective/feature`.

**No argument = stop.** Ask for a perspective, feature, titles, or `continue`.

## Steps

**Done when:** every rule in the queue has passed the Implementation gate, its tests are green after the last code phase, and it is marked `[x]`, or it is parked as a spec gap; `Removed:` in the closure is cleaned; the user has the phase plan and the report.

1. **Reach** both `AGENTS.md` files.
2. **Resolve the seed**: under `continue`, the batch of **Continue**. Otherwise against the feature directories of every perspective named in `perspectives/AGENTS.md`, or the global rules of a perspective's `AGENTS.md`. An existing tracking or spec path; otherwise a slug matched to the folder name. Ambiguous → list candidates and ask for an explicit path.
3. **Close** the set: follow **direct** title mentions and links (P7), including the global rules the seed cites. Deduplicate. Apply `:<titles>` after closure. Report paths and the phase plan **before** executing.
4. **Queue:** every `Removed:` in the closure, then the rules named by the argument (or every rule in the seed specs, if there is no title filter). Topological order by citations; policy before mechanism of the same owner; top → bottom in the file.
5. **Phases** = `##` section of the rule owner, band of 6–10 items. Merge only with the next section **in the same file**. `Removed:` forms the first phases, by owner. One phase = one owner and one origin (policy **or** mechanism). Each phase runs twice: first as a **code phase** (code and paths, no tests), then, once every code phase of the run is done, as a **test phase** (tests and paths), in the same order.
6. **Orchestrate** (below). Each item: Implementation gate. Update paths after each item. Mark `[x]` only after the test phases are green. If implementation or its tests are blocked or incomplete, leave `[ ]`. Spec files and `CONTEXT.md` stay untouched.
7. **Report** by phase and item: done, blocked, parked, files created, indirections (C7), spec gaps, and test results. No narrative on tracking.

A hole or open spec → park the rule under `## Spec gap` of `TODO.md` (shape in `perspectives/AGENTS.md`) and report `/spec`; do not edit the spec. Only the parent writes `TODO.md`; a child reports the block.

## Continue

`continue` implements the next batch of the backlog and stops. The backlog is every `[ ]` tracking entry under `perspectives/`. The completion marks are the only state: the next `continue` resumes where this one stopped, until no entry is `[ ]`.

**Done when:** every rule queued from the batch is `[x]` or parked as a spec gap; the user has the report and the next batch.

1. **Inventory:** run `node .agents/tools/pending.mjs --batch` (catalog in `.agents/tools/AGENTS.md`). It applies steps 2–4; where it disagrees with them, they win and the tool is fixed in the same session.
2. **Unit** = one owner: a feature — the entries of its `policy.md` and `mechanism.md` — or the global rules of one perspective — the entries beside its `AGENTS.md`. Perspectives in the order `development`, `business`, `user`, `operator`. Inside one, the global unit first, then features in topological order by cites between units, ties by name. Units in a cycle sit together.
3. **Waiting:** a rule waits when `TODO.md` holds it under `## Spec gap`, or when it cites — by title, link, or `Dependency:` — a `[ ]` rule that waits or lies outside the batch. A unit with no rule left to run is skipped and does not count.
4. **Batch:** the first perspective with a unit to run; its units in order, at most 5. One perspective per batch. A cycle enters whole, past 5 only when it opens the batch. No unit to run while `[ ]` entries remain → report what waits and on what, and stop. No `[ ]` entry → report that the backlog is done, plus drift, and stop.
5. **Run** Steps 3–7 with the batch as the seed. The queue holds only the rules of the batch left to run, `Removed:` first; `[x]` rules enter the closure for reading and are never queued. A `[ ]` entry that already has paths and no flag is an interrupted run: revalidate those paths against the body and finish it.
6. **Report**, on top of Step 7: rules parked in this run, drift between spec and tracking for `/spec`, and the next batch from a fresh run of the tool.

## Orchestration

The agent that receives `/implement` is the **parent**. Unit = **phase**, never the item.

**Parent**

1. Seed → closure → read specs and tracking for the set.
2. Build the queue and the phases. Report the plan.
3. Whole queue fits in one phase → run in this window, no subagent.
4. Otherwise, subagent waves (the harness's subagent tool, named in the root `AGENTS.md`; general-purpose, the parent's model, this workspace, fresh context, never resumed): up to 3 phases of **different owners** with no dependency edge, launched together. Phases of the **same** owner in series. Wait for every subagent of a wave before the next wave. Drain every code phase first; test phases start only after the last code phase, in the same waves.
5. After every test phase is green: read each owner's tracking once, confirm paths, mark `[x]` only for rules completed at 100% with green tests, and delete an entry if it was `Removed:`.
6. Once the last wave is done, park each spec gap a child reported.

**Child** — every item in the phase, in order; one read of the section; paths after each item; a code phase writes only code and tracking paths, a test phase only tests and tracking paths; per-item summary to the parent (title, completion status, paths, new files with C7 justification, block). The child never marks `[x]`: that belongs to the parent after the test phases are green.

**Child prompt** — self-contained: absolute paths of the spec, the tracking file, `src/AGENTS.md`, and `.agents/tools/AGENTS.md` (plus `src/INCREMENTAL-TESTING-GUIDELINES.md` for a test phase); phase kind (code or test); section name; ordered titles, completion marks, and current paths; process every item; Implementation gate steps 2–3 for a code phase, step 4 for a test phase; update paths but do not mark `[x]`; diagnostics down (code phase) or runner down (test phase) → block and leave `[ ]`; an open spec → block and report the gap, do not write `TODO.md`; no spec files; no items outside the phase.

Run in this window when it fits one phase. Existing-code search may go to a read-only search subagent; tracking reads stay with the parent (titles are the path-list key).
