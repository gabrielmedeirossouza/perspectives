# Source

Technical specs decide how the product is built; code implements them and the product specs they serve. Nodes and dependencies follow **Metadata** in the root `AGENTS.md`.

## Tree

```text
src/
  AGENTS.md
  BUG-AUTOPSIES.md
  INCREMENTAL-TESTING-GUIDELINES.md
  specs/<name>/                 # ASCII kebab-case
    spec.md
    spec.md.metadata.json
  <dir>/                        # placement decided by a technical spec
    <file>
    <file>.metadata.json
    .metadata.json              # non-code files in this directory
```

A global technical spec places root manifests and configuration. When it places them directly under `src/`, that directory is their node.

## Technical spec

A technical spec decides placement, contracts, persistence, packages and versions, toolchain, runtime, tests, lint, layering, environment variables, limits, and commands. Product policy and visible behavior belong to `specs/`.

Name the spec by the code boundary it decides: a module, shared mechanism, or global constraint that holds independently of features.

### Skeleton and format

Use **Spec file**, **Rule format**, and P1–P7 from `specs/AGENTS.md`, with this section's scope test. Consume the glossary; do not edit it here.

### Vocabulary

- State one decision per rule with `must`, `must not`, `never`, `only`, or `prohibited`. Open alternatives and facultative wording (`may`, `ideally`, `prefer`, `or equivalent`) leave the rule unfinished.
- Name the concrete package or tool in the body, and its version when that is a decision. Titles remain prose.
- An indirection requires a pain trigger stated in its rule body: second real implementation, anti-corruption boundary, or second real consumer.

### Adopted patterns

A global spec may list mechanisms under `## Adopted patterns`, each with the location of its concrete rules. These are list items obligating spec authors, not `###` product rules. Each feature technical spec answers under the same heading: adopts, or declines with a reason. Omit the heading until needed.

## Implementation postulates

These govern code under `src/`; volume, age, or prior review never excuses a violation.

**C1 Fidelity.** Implement the whole body of every listed product and technical rule, including conditions, exceptions, effects, and forbidden values. Partial implementation is never complete.

**C2 Tests after implementation.** Write tests only after every rule in the run has code; only unit suites are required. Test the module's public surface: entry points, exports, and pure helpers. Every rule has a suite; risk determines its scenarios:

| Band | Applies to | Suite |
|------|------------|-------|
| Critical | Money, irreversible data, authorization, isolation, state integrity, third-party contracts | Happy path, negatives, and relevant edges: bounds, empty, duplicate, concurrency, timezone, size, malformed input |
| Common | Observable behavior with branches | Public behavior |
| Trivial | Static values, linear flows, copy, persistence without a rule | Smallest observable contract |

When critical versus common is unclear, use critical. Test behavior rather than internal names or call counts; coverage percentage does not replace scenarios. Database mocks are allowed, subject to C7.

**C3 Technical law.** Obey every listed technical spec, including global constraints, and use the glossary's canonical names.

**C4 Broken windows.** Follow the rule in force; report neighboring deviations without copying them.

**C5 One implementation.** If a correct change to A requires the same change to B, give that obligation one implementation accessed through its public API. Extract at the second real consumer; independently variable behavior stays local, and a shared helper with one consumer violates C7.

**C6 Sources.** Maintain each written file's node and sources under the root **Metadata** contract. Add newly served perspectives; remove dangling sources after deleting their exclusive code. If no technical source remains, remove the node and its exclusive content or identify the governing spec. Clean removed rules before new work.

**C7 Economy.** Use the fewest files and indirections satisfying the specs. Interfaces, ports, layers, mappers, wrappers, and shared helpers require a pain trigger stated in the governing technical spec (**Vocabulary**). Test convenience is insufficient. Technical-type folders within a module require an explicit technical rule.

**C8 Closed spec.** Add no behavior, field, endpoint, package, or stack choice beyond the rule bodies. An unnamed mechanism, ambiguity, or open alternative blocks that rule: park a Design gap for `/sdd-design`, which resolves it or escalates to `/sdd-spec`.

## Implementation gate

1. Read the target technical specs, served product specs, glossary, and `pending` deltas.
2. Implement under C1–C8, including metadata skeletons; refresh metadata under root **Writers** after writes.
3. Check diagnostics before the first edit and after every executable edit. Fix introduced blocking errors, distinguishing pre-existing ones. Diagnostics unavailable → stop and report.
4. After all run code is complete, apply C2 and the testing guidelines below; run the suites green. Runner unavailable → stop and report.
5. Sync, under root **Writers**, only nodes whose entire source rules are implemented, required tests pass, and validation is complete. Unfinished nodes stay stale, with no other pending marker.

Permissions: code/tests outside `src/specs/`; code metadata skeletons; Design gaps in `TODO.md`; bug records and testing guidelines below. Technical specs belong to `/sdd-design`. Never edit product specs, the glossary, tool-owned metadata fields, or run `--approve`.

## Bug autopsy

Every corrected bug gets an entry in `BUG-AUTOPSIES.md` using its format. When it reveals a reusable testing practice, add that practice to `INCREMENTAL-TESTING-GUIDELINES.md`. Consult and apply that file whenever writing a unit test.
