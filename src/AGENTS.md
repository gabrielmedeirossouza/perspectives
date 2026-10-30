# Source

Technical specs and product code live in this tree. Product rules live in `specs/`; how they are built lives in `src/specs/`; how code materializes both — the implementation postulates and the Implementation gate — lives here. Nodes, sources, and states are defined under **Metadata** in the root `AGENTS.md`.

## Tree

```
src/
  AGENTS.md                          # this file: law of the tree
  BUG-AUTOPSIES.md                   # stateful record of corrected bugs
  INCREMENTAL-TESTING-GUIDELINES.md  # stateful record of testing practices
  specs/<name>/                      # one technical spec; <name> ASCII kebab-case
    spec.md
    .metadata.json                   # sources: the perspective specs it serves, the technical specs it depends on
  <code directory>/                  # a code node: any directory /implement declares
    .metadata.json                   # sources: the technical specs its code obeys and the perspective specs they serve; audited: approvals of /audit at its hash
  .metadata.json                     # iff src/ itself is the code node of the root manifests
```

Where code goes is a technical rule, not a convention of this file: a technical spec states the placement, and `/implement` declares the directory as a node. A file under `src/` outside every node is drift. Files directly under `src/` ending in `.md` are chrome: no node's content, never drift. The root manifests and configuration of the code — `go.mod`, `package.json`, lockfiles, compiler settings — live where the global technical spec that decides them says; when that is `src/` itself, `/implement` declares `src/` a code node listing that spec.

## Technical spec

A technical spec states **how** a part of the system is built: placement, contracts (HTTP verbs, status codes, payloads), persistence, libraries and their versions, toolchain, runtime, test runner, lint, code layering, env vars, numeric limits, command entry points. It never states who may, when it expires, what the screen shows, what the click does, or who may ship — those are product rules in `specs/`, which it serves.

`<name>` names what the spec builds, by the code boundary it decides, not by a perspective: a module (`invite`), a shared mechanism (`http-client`), or a global constraint that would hold with no feature in the repository (`toolchain`, `backend-layering`). A global technical spec is a node like any other; every code node it constrains lists it as a source.

### Skeleton and format

Same **Spec file** skeleton as `specs/AGENTS.md`: `#` the name of what it builds, one purpose sentence, then `##` subjects and `###` rules. Same **Rule format**, **Title** rules, **Anchors**, and postulates P1–P7; the scope test is this section. Spec language is English. The glossary `CONTEXT.md` is consumed, never edited here.

```markdown
# Invite

How invites are built: placement, persistence, and the HTTP contract.

## HTTP

### Invite creation endpoint

`POST /invites` **must** create one invite and answer `201` with its id. Serves [Invite expiry](../../../specs/business/invite/spec.md#invite-expiry).
```

### Vocabulary

- Titles stay prose (`Single HTTP client`). A rule that names a library or system tool **states the package** in the body (`acme-http`), with its version when the version is a decision.
- A rule states **one** decision with `must`, `must not`, `never`, `only`, `prohibited`. Facultative modals (`may`, `can be`, `ideally`, `prefer`), `or equivalent`, and `X or Y` are not a rule.
- An indirection — interface, port, layer, wrapper, shared helper — is written only with a pain trigger stated in the body: a second real implementation, an anti-corruption layer, or a second real consumer.
- A rule cites the product rule it serves by title, with a link into `specs/`, and never restates it. A global technical spec serves no single product rule and cites none.

### Adopted patterns

A global technical spec may carry a `## Adopted patterns` heading: the system uses a mechanism, and each technical spec that builds a feature **must** state whether it adopts it. Each pattern is one list item, never a `###` rule: the mechanism, and where its concrete rules live once adopted. The adopting spec answers with one list item under its own `## Adopted patterns` heading — adopts, or does not and why. A pattern obligates the person writing the spec, not the product. Create the heading when its first entry exists.

### Sources

The `.metadata.json` of a technical spec lists as `sources[].path` the perspective nodes whose rules it serves and the technical nodes it depends on. A perspective node no technical spec lists is pending design; a technical spec whose source changed is stale until `/design` reviews it and syncs. The write ends with `go run -C .agents/tools ./metadata --sync src/specs/<name>`.

## Implementation postulates

These rules govern how source code materializes the specifications. They apply to code under `src/` and do not define product behavior or its technical rules.

Violation is a grave failure. Volume, age, or “already reviewed” does not excuse it. Numbered **C** so they do not collide with the spec postulates **P** in `specs/AGENTS.md`.

**C1 Fidelity.** Code covers the whole rule body — conditions, exceptions, effects, forbidden values — of every product and technical rule the code node lists. Extra-normative behavior is a failure. Partial coverage presented as done is a failure. Ambiguous rule: stop and park a design gap; `/design` closes it or escalates it to `/spec`.

**C2 Tests after implementation.** Tests are written only after the code of every rule in the `/implement` run is complete — never before or alongside the code. Only unit tests are required. Tests hit the module's public surface (its entry file, exported rules, and pure helpers), not internal file shape. Risk sets **how many** scenarios, never whether a suite exists.

| Band | When the rule is | Suite |
|------|------------------|-------|
| Critical | money, irreversible data, authorization, isolation, state integrity, third-party contract | Happy path, negatives, and edges: bounds, empty, duplicate, concurrency, timezone, max size, malformed input |
| Common | observable behavior with branches | Public-surface behavior |
| Trivial | static value, linear flow, copy, persistence without a rule | Smallest observable contract test; still a suite |

On critical vs common, treat as critical. A suite that breaks on an internal rename is pinned to implementation. Mocking the database is allowed; a port whose only job is to enable a mock is C7. Asserting “the repository was called once” is not a behavior test. Coverage percentage does not replace scenarios.

**C3 Technical law.** New or changed code obeys every technical spec its code node lists as a source, the global ones included. A mechanism with no rule, a package named only as “the persistence client”, or an open alternative (`X or Y`, `or equivalent`): stop and report to `/design`. Copy and names follow `CONTEXT.md`; this tree does not edit the glossary.

**C4 Broken windows.** A neighboring deviation is not a warrant to copy it. Implement the rule in force; report the deviation.

**C5 One implementation.** The same documented obligation is not reimplemented in parallel. Copy vs coincidence: if the correct change to A requires the same change in B, it is copy — one owner, consumers call that module's public API. If A and B may diverge, leave them. Extract at the **second real consumer**, not in anticipation. A shared helper with one consumer is C7, not a C5 win.

**C6 Sources.** Every directory `/implement` writes code into belongs to a code node whose `.metadata.json` lists at least one technical spec — every technical spec its code obeys directly — and every perspective node those serve. `/implement` keeps that list: it adds a perspective node a technical spec came to serve (`pending`: missing source) and drops a source whose spec is gone (dangling); when no technical spec remains, it deletes the node with its exclusive code or lists the technical spec that now governs it. A code file under no node is a failure. The node is synced (`metadata --sync <dir>`) only when every rule its sources hold is 100% implemented, its required tests are green, and applicable validation is complete; until then it stays stale, and stale is the only pending mark. A removed rule is processed before new work on that node: delete exclusive orphan code, then sync. Whether each listed spec governs the code is judged by `/audit`, not asserted here: every code write, and every sync that records a newly listed source, returns the node's `audited` to 0 (**Audit** in the root `AGENTS.md`).

**C7 Economy.** The fewest files and indirections that satisfy the rule. Interface, port, layer, mapper, wrapper, or shared helper only with a pain trigger declared in the technical spec: second real implementation, an anti-corruption layer, or C5's second consumer. “Needed for tests” is not a trigger. Folder-by-technical-type inside a module (`domain/`, `use-cases/`, `dtos/`, `mappers/`) is a failure unless a technical rule obligates it.

**C8 Closed spec.** Only what the product and technical rule bodies state. Inventing a field, endpoint, or library, picking among open alternatives, or inferring stack from surrounding code or market default is a failure. Stop and report to `/design`.

## Implementation gate

**Done when:** every targeted rule has code that holds C1–C8; each code node written lists its sources; diagnostics show no introduced blocking errors; the unit suite written after the run's code is green; each finished node is synced.

1. **Reach** the root `AGENTS.md` (**Metadata**), this file, and `specs/AGENTS.md`. Read the technical spec, the perspective specs it serves, and `CONTEXT.md`. `pending` names the rules that changed and their state.
2. **Implement** the whole rule body. Place it where the technical spec says (C3, C7). A new directory gets its `.metadata.json` — `{"version": 1, "sources": [...]}`, paths relative to the repository root — then `go run -C .agents/tools ./metadata --write <dir>`. Write no test in this step.
3. **Sense:** diagnostics (as the root `AGENTS.md` defines them) before the first edit and after each executable edit. Distinguish pre-existing diagnostics from introduced ones. Fix introduced ones. If diagnostics are down, stop and report.
4. **Test** once every rule in the run has code: band each rule (C2), write its public-surface suite, and run it green. If the test runner is down, stop and report.
5. **Sync:** `go run -C .agents/tools ./metadata --sync <dir>` for each code node whose rules are 100% implemented with green tests. A node with anything left stays unsynced. Every node whose code was written, or whose sync recorded a new source, now owes `/audit` (`pending` → `# Audit`).
6. **Stop.** Do not edit spec files, `CONTEXT.md`, or any `.metadata.json` beyond `sources[].path` during implementation. In `TODO.md`, only park a design gap under `## Design gap`.

| | Ask |
|--|------|
| C1 | Does the code cover the whole body, including exceptions? |
| C2 | Tests written only after the run's code? Band scenarios and critical edges present? |
| C3 | Does placement and stack match the technical specs the node lists? Is a mechanism unnamed? |
| C4 | Am I copying a neighboring deviation? |
| C5 | Copy (one owner) or coincidence? Second real consumer? |
| C6 | Does the node list every technical spec it realizes and the perspective specs they serve? Is a removed rule still in the code? |
| C7 | Does every extra file or indirection have a declared trigger? |
| C8 | Did I invent behavior or pick an open alternative? |

Permissions for implementation: create, edit, and delete code and tests under `src/` outside `src/specs/`; write `sources[].path` of its code nodes' `.metadata.json`; run the metadata tool (never `--approve`, which is `/audit`'s). `/design` owns `src/specs/`; `CONTEXT.md` and `TODO.md` follow the Writing gate of `specs/AGENTS.md`, except the `## Design gap` items implementation parks.

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
