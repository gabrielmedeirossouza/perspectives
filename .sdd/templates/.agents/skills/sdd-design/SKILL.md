---
name: sdd-design
description: >-
  Writes, changes, or removes technical rules in src/specs/ — stack,
  placement, HTTP and persistence contracts, toolchain, layering — one
  decision per rule with the cost of the rejected alternative, and declares
  which perspective specs each technical spec serves so a product change
  reaches it. Use when a perspective spec has no technical spec, when pending
  reports a technical spec stale or dangling, when adding, changing, or
  removing a technical rule, or when the user types /sdd-design.
---

# sdd-design

Reach the root `AGENTS.md` (**Metadata**), `src/AGENTS.md` (**Technical spec**), and `specs/AGENTS.md` before writing. The tree, the scope test, the skeleton, the vocabulary, and the sources law live there — not here. A technical spec says how; the product rules it serves say what, and are not edited here.

## User Input

```text
$ARGUMENTS
```

## Arguments

```text
<arg> ::= [<name>][:]<request>
<name> ::= lower kebab-case name of a technical spec, src/specs/<name>
<request> ::= what to design, change, or remove
```

Examples: `/sdd-design invite:HTTP contract and persistence`; `/sdd-design toolchain:test runner`; `/sdd-design review invite against the new expiry rule`; `/sdd-design remove http-client`.

The argument is a **request**. The name is optional; locate the technical spec from the text. **No argument = run `go run -C .agents/tools ./pending`, present its `# Design` section — perspective specs with no technical spec, technical specs stale or dangling — and the `## Audit gap` items of `TODO.md` — code nodes no technical spec governs — and stop.** Ask which to take. Do not infer the target from a prior session or a git branch.

## Steps

**Done when:** every agreed item has passed the gate below; each technical spec written lists its sources and is synced; spec gaps are parked; design gaps this session closed are gone from `TODO.md`; product items are named for `/sdd-spec`.

1. **Reach** the three `AGENTS.md` files. Run `go run -C .agents/tools ./pending`: when the target is stale, its line names the rules that changed. Read the perspective specs the target serves, `CONTEXT.md`, and `TODO.md`.
2. **Decompose** the request into atomic items. One item decides one thing: one library, one placement, one contract, one limit.
3. **Scope** each item by **Technical spec** in `src/AGENTS.md`. A product item — who may, when, what the screen shows — is not written here: park it under `## Spec gap` of `TODO.md`, name it for `/sdd-spec`, and continue; a technical rule that would serve it waits for it. Resolve the owner: an existing technical spec; a new `src/specs/<name>` named by the code boundary it decides; a global one when the rule would hold with no feature in the repository.
4. **Discover** in the repo: technical specs under `src/specs/`, code under `src/`, package manifests and lockfiles. Facts are not questions. Decisions are the user's.
5. **Order** the items: dependencies first — a global technical spec before the ones that depend on it; the rule another cites before the citing one.
6. **Interview** until shared understanding. One question at a time, in the order of step 5, with a recommendation: the option, why, and the cost of the rejected alternative. Walk every branch; if an answer opens a new one, follow it. An indirection needs its pain trigger named. Check drafts against P1–P7 and the technical **Vocabulary** before proposing a write.
7. **Confirm:** "Shall I apply these changes?" Only then write.
8. **Write** `src/specs/<name>/spec.md` to the skeleton. For a new node, write `src/specs/<name>/spec.md.metadata.json` as `{"version": 2, "sources": [{"path": "..."}]}`: the perspective specs it serves and the technical specs it depends on, each by its `spec.md`, paths relative to the repository root — `{"version": 2}` for a global spec, which lists none. Then `go run -C .agents/tools ./metadata --sync src/specs/<name>`. A stale spec that needs no change after review is only synced — but a source rule removed or renamed takes its citation along first: `pending` lists the link that names no heading as drift of this spec, and a code node cannot sync against a spec with drift. A dangling source is a perspective spec that is gone: remove the rules that served only it, drop its `sources[].path`, then sync — or delete the whole spec when nothing of it remains. Park a product hole under `## Spec gap` of `TODO.md`; delete the `## Design gap` and `## Audit gap` items this write closes. Removing a spec deletes its directory; the code nodes that listed it show as dangling in `pending`, for `/sdd-implement`.
9. **Report** by technical spec: what was written, where, which product rules it serves, what was synced without change, what was parked, and the product items handed to `/sdd-spec`.

### Interview

```markdown
**Question:** [one decision]
**Recommendation:** [option + why + cost of the rejected alternative]
**Context:** [fact already found, if any]
```

Postulate violation: name the P, the passage, the recommended fix. Vocabulary violation: the facultative modal, `or equivalent`, or `X or Y` to replace by one decision.

Permissions: create, edit, and delete `src/specs/<name>/spec.md` with its directory, and the `sources[].path` of its `spec.md.metadata.json`; run the metadata tool (never `--approve`, which is `/sdd-audit`'s); edit `TODO.md` (`## Spec gap`, `## Design gap`, and the `## Audit gap` items a write closes). Out of scope: `specs/` (`/sdd-spec`); code and code nodes under `src/` (`/sdd-implement`); `CONTEXT.md`; any hash in a metadata file.
