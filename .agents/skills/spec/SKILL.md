---
name: spec
description: >-
  Writes, changes, or removes rules in perspectives/ — policy (product
  policy, flow, screen behavior, UI copy, operational policy) and mechanism
  (stack, HTTP, persistence, placement, contracts) — classifying every item
  of the request by layer on its own. Use when adding, changing, or removing
  any policy or mechanism rule, including a request that mixes both, when
  turning a brainstorm from .sdd/brainstorming/ into specs, or when the user
  types /spec.
---

# spec

Reach `perspectives/AGENTS.md` and the nested `AGENTS.md` of the target perspective before classifying or writing. The perspectives that exist are named in `perspectives/AGENTS.md`. Law, tree, rule format, glossary shape, postulates, Vocabulary, and tracking headings live there — not here.

One request carries policy and mechanism items. Splitting them is this skill's job, not the user's.

## User Input

```text
$ARGUMENTS
```

## Arguments

```text
<arg> ::= [<perspective-or-feature>][:]<request> | brainstorming[:<name>]
<request> ::= what to document, change, or remove
<name> ::= lower kebab-case name of a file in .sdd/brainstorming/
```

Examples: `/spec add invite expiry and its HTTP contract`; `/spec user/home:empty list state`; `/spec business/users:account HTTP contract`; `/spec development:test runner`; `/spec remove invite-by-link`; `/spec brainstorming`; `/spec brainstorming:invite-expiry`.

The argument is a **request**. Path and slug are optional; locate files under `perspectives/` from the text. **No argument = stop.** Do not infer the target from a prior session or a git branch.

### Brainstorm source

`brainstorming[:<name>]`, or a request that asks to use a brainstorm, makes one file of `.sdd/brainstorming/` the request. Resolve it: `<name>` given → that file, missing → stop and list the available ones; none exists → stop and suggest `/brainstorming`; one → use it; two or more → ask which. One brainstorm per run.

Decisions recorded in the brainstorm are the user's: the interview does not ask them again. It asks the brainstorm's open questions, what the current specs contradict, and what the postulates raise.

## Steps

**Done when:** every agreed item has passed the Writing gate; tracking headings and flags are in sync; paths are untouched; a consumed brainstorm is deleted.

1. **Reach** both `AGENTS.md` files. Consult `CONTEXT.md` and `TODO.md`. Read the brainstorm when it is the source.
2. **Decompose** the request into atomic items. One item asserts one thing. A brainstorm decomposes from its `Candidate items`, `Decisions`, `Flows`, and `Edge cases`.
3. **Classify** each item: **layer** by the layer test of the candidate perspective's `AGENTS.md`, then the perspective — a mechanism item that passes the boundary test of `perspectives/development/AGENTS.md` goes to `development/` — then the owner by step 2 of the **Writing gate**. No item is handed back to the user as another skill's problem. An item that is neither layer — code, the law of a folder — is named as out of scope and the rest continues.
4. **Discover** in the repo (code, specs under `perspectives/`, layout, `package.json`). Facts are not questions. Decisions are the user's. A brainstorm's `Integration` is re-checked against the current specs; it may be stale.
5. **Order** the items: policy before mechanism of the same owner, because a mechanism rule cites by title the policy rule it serves and that title has to exist first. Within a layer, dependencies first.
6. **Interview** until shared understanding. One question at a time, in the order of step 5, with a recommendation. Walk every branch; if an answer opens a new one, follow it. Per-layer obligations are in the table below. Check drafts against the postulates **before** proposing a write.
7. **Confirm:** "Shall I apply these changes?" Only then the Writing gate.
8. **Write** through the Writing gate. Sync headings and flags. Park what this request cannot close; delete what this session unblocked. Do not write paths. Once every agreed item has passed the gate, delete the source brainstorm. Declined at **Confirm** or stopped before → the brainstorm stays.
9. **Report** by layer and owner: what was written, where, and what was parked. Name the deleted brainstorm. When `.sdd/prototypes/<name>.md` exists, say the prototype is still alive and is removed with `/prototype --discard <name>`; do not discard it.

## Layers

| | Policy item | Mechanism item |
|--|-------------|----------------|
| `rule` is | policy or surface | mechanism |
| Owner files | `policy.md`, `## Global policy rules` | `mechanism.md`, `## Global mechanism rules` |
| The interview closes | the decision, with its branches | **one** library or contract, with the cost of the rejected alternative |
| `CONTEXT.md` | **edit** in the same session a term is defined, renamed, or retired | consume; do not edit |
| Before writing, also check | the glossary uses the canonical term | **Vocabulary**: the package named in the body, no `or equivalent`, no `X or Y`, no facultative modal |
| Citation | never cites `mechanism.md` | cites the policy rule it serves **by title only**; a `development/` rule cites none |

Indirection — interface, port, layer, wrapper — is written only with a pain trigger declared in the mechanism body.

Boundary between the two: a passage that decides who may, when it expires, what the screen shows, or what the click does is **policy**, whatever the request called it. A passage about a library, placement, contract, or persistence is **mechanism**.

### Interview

```markdown
**Question:** [one decision]
**Recommendation:** [option + why + cost of the alternative, for a mechanism item]
**Context:** [fact already found, if any]
```

Glossary conflict: point at the `CONTEXT.md` entry and the intended wording. Postulate violation: name the P, the passage, the recommended fix.

Out of scope: the law of the folder in a nested `AGENTS.md`, tracking paths, code under `src/`, editing a brainstorm. Code is `/implement`; brainstorms are `/brainstorming`.
