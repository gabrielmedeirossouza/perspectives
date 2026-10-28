---
name: spec
description: >-
  Writes, changes, or removes rules in perspectives/ — functional (product
  policy, flow, screen behavior, UI copy) and tech (mechanism, stack, HTTP,
  persistence, placement, contracts) — classifying every item of the request
  by layer on its own. Use when adding, changing, or removing any product or
  mechanism rule, including a request that mixes both, or when the user types
  /spec.
---

# spec

Reach `perspectives/AGENTS.md` and the nested `AGENTS.md` of the target perspective before classifying or writing. The perspectives that exist are named in `perspectives/AGENTS.md`. Law, tree, rule format, glossary shape, postulates, Vocabulary, and tracking headings live there — not here.

One request carries functional and tech items. Splitting them is this skill's job, not the user's.

## User Input

```text
$ARGUMENTS
```

## Arguments

```text
<arg> ::= [<perspective-or-feature>][:]<request>
<request> ::= what to document, change, or remove
```

Examples: `/spec add invite expiry and its HTTP contract`; `/spec web/home:empty list state`; `/spec business/users:account HTTP contract`; `/spec remove invite-by-link`.

The argument is a **request**. Path and slug are optional; locate files under `perspectives/` from the text. **No argument = stop.** Do not infer the target from a prior session or a git branch.

## Steps

**Done when:** every agreed item has passed the Writing gate; tracking headings and flags are in sync; paths are untouched.

1. **Reach** both `AGENTS.md` files. Consult `CONTEXT.md` and `TODO.md`.
2. **Decompose** the request into atomic items. One item asserts one thing.
3. **Classify** each item: **layer** by the layer test of the candidate perspective's `AGENTS.md`, then the perspective, then the owner by step 2 of the **Writing gate**. No item is handed back to the user as another skill's problem. An item that is neither layer — code, the law of a folder — is named as out of scope and the rest continues.
4. **Discover** in the repo (code, specs under `perspectives/`, layout, `package.json`). Facts are not questions. Decisions are the user's.
5. **Order** the items: functional before tech of the same owner, because a tech rule cites by title the functional rule it serves and that title has to exist first. Within a layer, dependencies first.
6. **Interview** until shared understanding. One question at a time, in the order of step 5, with a recommendation. Walk every branch; if an answer opens a new one, follow it. Per-layer obligations are in the table below. Check drafts against the postulates **before** proposing a write.
7. **Confirm:** "Shall I apply these changes?" Only then the Writing gate.
8. **Write** through the Writing gate. Sync headings and flags. Park what this request cannot close; delete what this session unblocked. Do not write paths.
9. **Report** by layer and owner: what was written, where, and what was parked.

## Layers

| | Functional item | Tech item |
|--|-----------------|-----------|
| `rule` is | policy or surface | mechanism |
| Owner files | `functional-spec.md`, `## Global functional rules` | `tech-spec.md`, `## Global tech rules` |
| The interview closes | the decision, with its branches | **one** library or contract, with the cost of the rejected alternative |
| `CONTEXT.md` | **edit** in the same session a term is defined, renamed, or retired | consume; do not edit |
| Before writing, also check | the glossary uses the canonical term | **Vocabulary**: the package named in the body, no `or equivalent`, no `X or Y`, no facultative modal |
| Citation | never cites `tech-spec.md` | cites the functional rule it serves **by title only** |

Indirection — interface, port, layer, wrapper — is written only with a pain trigger declared in the tech body.

Boundary between the two: a passage that decides who may, when it expires, what the screen shows, or what the click does is **functional**, whatever the request called it. A passage about a library, placement, contract, or persistence is **tech**.

### Interview

```markdown
**Question:** [one decision]
**Recommendation:** [option + why + cost of the alternative, for a tech item]
**Context:** [fact already found, if any]
```

Glossary conflict: point at the `CONTEXT.md` entry and the intended wording. Postulate violation: name the P, the passage, the recommended fix.

Out of scope: the law of the folder in a nested `AGENTS.md`, tracking paths, code under `src/`. Code is `/implement`.
