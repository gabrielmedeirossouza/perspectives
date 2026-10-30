---
name: implement
description: >-
  Implements product code under src/ from a technical spec in src/specs/ and
  the perspective specs it serves, or, with continue, the next batch of the
  backlog — up to 5 units, sources first, as the pending tool reports them.
  Use when the user wants code for a technical spec or rule title, wants to
  continue implementing what is pending, or types /implement.
---

# implement

Reach the root `AGENTS.md` (**Metadata**) and `src/AGENTS.md` before resolving a target or writing code. The first defines nodes, sources, and states; the second the source tree, the implementation postulates (C1–C8), and the Implementation gate.

The queue is `$ARGUMENTS`, not the state — except under `continue`, which builds it from `pending` (**Continue**). A code node is **stale** while a source changed since its last sync; **fresh** means implemented and tested, because only a passed gate syncs it. The metadata is the only state.

## User Input

```text
$ARGUMENTS
```

## Arguments

```text
<arg> ::= continue | <target>[:<titles>]
<target> ::= src/specs/<name> | <name>
<titles> ::= comma-separated rule titles
```

Examples: `/implement continue`; `/implement invite`; `/implement src/specs/invite:Invite creation endpoint`; `/implement http-client`.

`continue` also answers to a single word that asks to resume: `next`, `resume`, `continuar`, `próximo`, `seguir`. The keyword wins over a name.

**No argument = stop.** Ask for a technical spec, titles, or `continue`.

## Steps

**Done when:** every rule in the queue has passed the Implementation gate and its node is synced, or it is parked as a design gap; removed rules in the closure are cleaned; the user has the phase plan and the report.

1. **Reach** the root `AGENTS.md`, `src/AGENTS.md`, and `perspectives/AGENTS.md`.
2. **Resolve the seed**: under `continue`, the batch of **Continue**. Otherwise a technical spec under `src/specs/`, by path or by name. Ambiguous → list candidates and ask.
3. **Close** the set: the technical spec, the perspective specs it serves, the technical specs it depends on (for reading), and the code nodes that list it. Run `go run -C .agents/tools ./pending`: its `# Implement` lines name, per node, the rules to run and their state — new, changed, removed, renamed — and whether the node waits on `/design`. Apply `:<titles>` after closure. A unit that waits is not run: report what `/design` owes and stop. Report the plan **before** executing.
4. **Queue:** removed rules first, then the rules `pending` names — or every rule of the seed and of the perspective specs it serves, for a technical spec with no code node. Topological order by citations; the product rule before the technical rule that serves it; top → bottom in the file.
5. **Phases** = `##` section of the rule owner — a perspective spec or the technical spec — restricted to the queued rules, band of 6–10 items. Merge only with the next section **in the same file**. Removed rules form the first phases. One phase = one owner. Each phase runs twice: first as a **code phase** (code and nodes, no tests), then, once every code phase of the run is done, as a **test phase** (tests), in the same order.
6. **Orchestrate** (below). Each item: Implementation gate. A new directory gets its `.metadata.json` with `sources[].path` and `metadata --write`. A code node that `pending` reports as missing a source gets that perspective node added to `sources[].path`; a dangling source is dropped once its exclusive code is deleted. A code node is synced only after the test phases are green and every rule its sources hold is 100% done; otherwise it stays stale. Spec files, `src/specs/`, and `CONTEXT.md` stay untouched.
7. **Report** by phase and item: done, blocked, parked, files created, indirections (C7), design gaps, test results, nodes synced and nodes left stale, and the nodes that now owe `/audit`.

A hole or open spec, product or technical → park the rule under `## Design gap` of `TODO.md` (shape in `perspectives/AGENTS.md`) and report `/design`, which closes it or escalates it to `/spec`; do not edit the spec. Only the parent writes `TODO.md`; a child reports the block.

## Continue

`continue` implements the next batch of the backlog and stops. The backlog is what `pending` reports under `# Implement`: technical specs no code node realizes, and code nodes stale against a source or holding a dangling one. The next `continue` resumes where this one stopped, until `pending` prints "No pending work."

**Done when:** every unit of the batch is synced or parked as a design gap; the user has the report and the next batch.

1. **Inventory:** run `go run -C .agents/tools ./pending --batch` (catalog in `.agents/tools/AGENTS.md`). It applies steps 2–3; where it disagrees with them, they win and the tool is fixed in the same session.
2. **Unit** = one technical spec with no code node, or one code node stale against a source, holding a dangling one, or missing a perspective node its technical specs serve. Units run sources first: a unit comes after every technical spec it realizes or depends on; ties by path. A unit **waits** when its technical spec, or one upstream of it, is stale or dangling — that is `/design`'s work first — or when every rule it has left is parked under `## Design gap`. A parked rule is skipped inside a unit that runs.
3. **Batch:** the first 5 units that can run, in order. No unit can run while units remain → report what waits and on what, and stop. Nothing pending → report that the backlog is done, plus drift, and stop.
4. **Run** Steps 3–7 with the batch as the seed. A stale node that already has code is an interrupted or outdated run: revalidate its code against the rules `pending` names and finish it. A dangling source means the spec is gone: delete the code exclusive to it, then drop the source from the node; if no technical spec remains, delete the node or list the technical spec that now governs it.
5. **Report**, on top of Step 7: rules parked in this run, drift for whoever owns it, and the next batch from a fresh run of the tool.

## Orchestration

The agent that receives `/implement` is the **parent**. Unit = **phase**, never the item.

**Parent**

1. Seed → closure → read specs and metadata for the set.
2. Build the queue and the phases. Report the plan.
3. Whole queue fits in one phase → run in this window, no subagent.
4. Otherwise, subagent waves (the harness's subagent tool, named in the root `AGENTS.md`; general-purpose, the parent's model, this workspace, fresh context, never resumed): up to 3 phases of **different owners** with no dependency edge, launched together. Phases of the **same** owner in series. Wait for every subagent of a wave before the next wave. Drain every code phase first; test phases start only after the last code phase, in the same waves.
5. After every test phase is green: `go run -C .agents/tools ./metadata --sync <dir>` for each code node whose rules are all done with green tests; a node with a blocked rule stays unsynced. Then run `pending` once to confirm.
6. Once the last wave is done, park each design gap a child reported.

**Child** — every item in the phase, in order; one read of the section; a code phase writes only code and, for a new directory, its `.metadata.json` with `sources[].path` followed by `metadata --write`; a test phase only tests; per-item summary to the parent (title, completion status, files, new files with C7 justification, block). The child never runs `metadata --sync`: that belongs to the parent after the test phases are green.

**Child prompt** — self-contained: absolute paths of the technical spec, the perspective specs it serves, the code node directories and their `.metadata.json`, the root `AGENTS.md`, `src/AGENTS.md`, and `.agents/tools/AGENTS.md` (plus `src/INCREMENTAL-TESTING-GUIDELINES.md` for a test phase); phase kind (code or test); section name; ordered titles with their state from `pending`; process every item; Implementation gate steps 2–3 for a code phase, step 4 for a test phase; never `--sync`; diagnostics down (code phase) or runner down (test phase) → block; an open spec → block and report the gap, do not write `TODO.md`; no spec files; no items outside the phase.

Run in this window when it fits one phase. Existing-code search may go to a read-only search subagent; `pending` and metadata reads stay with the parent.
