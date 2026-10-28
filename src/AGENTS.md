# Source

Product code lives in this tree. Product and technical rules live in `perspectives/`; how code materializes them — the implementation postulates and the Implementation gate — lives here.

## Tree

```
src/
  backend/
  frontend/
```

## Implementation postulates

These rules govern how source code materializes the specifications in `perspectives/`. They apply to code under `src/` and do not define product behavior or its technical mechanism.

Violation is a grave failure. Volume, age, or “already reviewed” does not excuse it. Numbered **C** so they do not collide with the spec postulates **P** in `perspectives/AGENTS.md`.

**C1 Fidelity.** Code covers the whole rule body — conditions, exceptions, effects, forbidden values. Extra-normative behavior is a failure. Partial coverage presented as done is a failure. Ambiguous rule: stop and report to `/spec`.

**C2 Tests after implementation.** Tests are written only after the code of every rule in the `/implement` run is complete — never before or alongside the code. Only unit tests are required. Tests hit the module's public surface (`index.ts`, exported rules and pure helpers), not internal file shape. Risk sets **how many** scenarios, never whether a suite exists.

| Band | When the rule is | Suite |
|------|------------------|-------|
| Critical | money, irreversible data, authorization, isolation, state integrity, third-party contract | Happy path, negatives, and edges: bounds, empty, duplicate, concurrency, timezone, max size, malformed input |
| Common | observable behavior with branches | Public-surface behavior |
| Trivial | static value, linear flow, copy, persistence without a rule | Smallest observable contract test; still a suite |

On critical vs common, treat as critical. A suite that breaks on an internal rename is pinned to implementation. Mocking the database is allowed; a port whose only job is to enable a mock is C7. Asserting “the repository was called once” is not a behavior test. Coverage percentage does not replace scenarios.

**C3 Tech law.** New or changed code obeys the applicable `tech-spec.md` and the global tech rules of the perspective's `AGENTS.md`. Missing tech spec for the mechanism, a package named only as “the persistence client”, or an open alternative (`X or Y`, `or equivalent`): stop and report to `/spec`. Copy and names follow `CONTEXT.md`; this tree does not edit the glossary.

**C4 Broken windows.** A neighboring deviation is not a warrant to copy it. Implement the rule in force; report the deviation.

**C5 One implementation.** The same documented obligation is not reimplemented in parallel. Copy vs coincidence: if the correct change to A requires the same change in B, it is copy — one owner, consumers call that module's public API. If A and B may diverge, leave them. Extract at the **second real consumer**, not in anticipation. A shared helper with one consumer is C7, not a C5 win.

**C6 Paths.** After the code exists, the tracking entry lists every file that realizes the rule (code, tests, relevant directories), one repo-relative backtick path per line. Dead paths and omitted essentials are failures. Mark `[x]` only after the rule is 100% implemented, its required tests are green, and applicable validation is complete. `Removed:` is processed before new work on that feature: delete exclusive orphan code, then delete the entry. Spec skills write headings and change flags; implementation writes completion marks and paths.

**C7 Economy.** The fewest files and indirections that satisfy the rule. Interface, port, layer, mapper, wrapper, or shared helper only with a pain trigger declared in the tech spec: second real implementation, an anti-corruption layer, or C5's second consumer. “Needed for tests” is not a trigger. Folder-by-technical-type inside a module (`domain/`, `use-cases/`, `dtos/`, `mappers/`) is a failure unless the tech spec obligates it.

**C8 Closed spec.** Only what the functional and tech rule bodies state, plus what applicable tech rules fix for the mechanism. Inventing a field, endpoint, or library, picking among open alternatives, or inferring stack from surrounding code or market default is a failure. Stop and report.

## Implementation gate

**Done when:** every targeted rule has code that holds C1–C8; the path list matches those files; the tracking entry is marked `[x]`; `Removed:` on the path is cleaned or gone; diagnostics show no introduced blocking errors; the unit suite written after the run's code is green.

1. **Reach** this file, `perspectives/AGENTS.md`, and the nested `AGENTS.md` of the perspective that owns the rule — that file holds the perspective's global rules as well. Read the rule body, its sibling tech spec, and `CONTEXT.md`.
2. **Implement** the whole rule body. Place it where the tech spec says (C3, C7). Write no test in this step.
3. **Sense:** diagnostics (as the root `AGENTS.md` defines them) before the first edit and after each executable edit. Distinguish pre-existing diagnostics from introduced ones. Fix introduced ones. If diagnostics are down, stop and report.
4. **Test** once every rule in the run has code: band each rule (C2), write its public-surface suite, and run it green. If the test runner is down, stop and report.
5. **Complete tracking:** process `Removed:` first when present; write paths on the existing tracking heading (C6), and mark `[x]` only when the rule is 100% implemented and its tests are green.
6. **Stop.** Do not edit spec files, `CONTEXT.md`, or `TODO.md` during implementation. Do not create, rename, or delete a tracking heading except to delete an entry after `Removed:` is cleaned.

| | Ask |
|--|------|
| C1 | Does the code cover the whole body, including exceptions? |
| C2 | Tests written only after the run's code? Band scenarios and critical edges present? |
| C3 | Does placement and stack match the tech spec? Is a mechanism unnamed? |
| C4 | Am I copying a neighboring deviation? |
| C5 | Copy (one owner) or coincidence? Second real consumer? |
| C6 | Does the path list match the files? Is `Removed:` still pending? |
| C7 | Does every extra file or indirection have a declared trigger? |
| C8 | Did I invent behavior or pick an open alternative? |

Permissions for implementation: create, edit, and delete code and tests under `src/`. Write, update, and delete **paths** on tracking entries, and delete an entry after `Removed:` is cleaned. Spec skills own spec files and tracking headings; `CONTEXT.md` and `TODO.md` follow the Writing gate of `perspectives/AGENTS.md`.

## Bug autopsy

Every bug correction under `src/` **must** add one entry to
`src/BUG-AUTOPSIES.md`. The entry **must** record the bug's symptom, impact,
root cause, why the existing code allowed it, and the correction.

During the creation of every unit test under `src/`, the author **must**
consult `src/INCREMENTAL-TESTING-GUIDELINES.md` and apply every applicable
guideline recorded there.

When an autopsy reveals a reusable testing practice, the practice **must** be
added to `src/INCREMENTAL-TESTING-GUIDELINES.md` as a concrete testing
guideline.
