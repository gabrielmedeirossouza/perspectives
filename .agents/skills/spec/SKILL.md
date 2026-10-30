---
name: spec
description: >-
  Writes, changes, or removes product rules in specs/ — product
  policy, flow, screen behavior, UI copy, operational policy — one spec.md
  per feature or perspective, and names the technical items of a request
  (stack, HTTP, persistence, placement, contracts) for /design. Use when
  adding, changing, or removing any product rule, including a request that
  mixes product and technical items, when turning a brainstorm from
  .sdd/brainstorming/ into specs, or when the user types /spec.
---

# spec

Reach `specs/AGENTS.md` and the nested `AGENTS.md` of the target perspective before scoping or writing. The perspectives that exist are named in `specs/AGENTS.md`. Law, tree, rule format, glossary shape, postulates, scope, and metadata live there — not here.

One request carries product and technical items. Splitting them is this skill's job, not the user's: the product items are written here; the technical ones are named for `/design`, after the product rules they will serve exist.

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

Examples: `/spec add invite expiry`; `/spec user/home:empty list state`; `/spec operator/release:who may promote to production`; `/spec remove invite-by-link`; `/spec brainstorming`; `/spec brainstorming:invite-expiry`.

The argument is a **request**. Path and slug are optional; locate files under `specs/` from the text. **No argument = stop.** Do not infer the target from a prior session or a git branch.

### Brainstorm source

`brainstorming[:<name>]`, or a request that asks to use a brainstorm, makes one file of `.sdd/brainstorming/` the request. Resolve it: `<name>` given → that file, missing → stop and list the available ones; none exists → stop and suggest `/brainstorming`; one → use it; two or more → ask which. One brainstorm per run.

Decisions recorded in the brainstorm are the user's: the interview does not ask them again. It asks the brainstorm's open questions, what the current specs contradict, and what the postulates raise.

## Steps

**Done when:** every agreed product item has passed the Writing gate; every written directory has fresh metadata; technical items are named for `/design`; a consumed brainstorm is deleted.

1. **Reach** both `AGENTS.md` files. Consult `CONTEXT.md` and `TODO.md`. Read the brainstorm when it is the source.
2. **Decompose** the request into atomic items. One item asserts one thing. A brainstorm decomposes from its `Candidate items`, `Decisions`, `Flows`, and `Edge cases`.
3. **Scope** each item by the scope test of the candidate perspective's `AGENTS.md`: a product rule → that perspective, then the owner by step 2 of the **Writing gate**; a technical item → named for `/design`, with the product rule it will serve. No item is handed back to the user as another skill's problem. An item that is neither — code, the law of a folder — is named as out of scope and the rest continues.
4. **Discover** in the repo: specs under `specs/`, technical specs under `src/specs/`, code under `src/`. Facts are not questions. Decisions are the user's. A brainstorm's `Integration` is re-checked against the current specs; it may be stale.
5. **Order** the items: dependencies first — a rule another cites by title comes before the citing one.
6. **Interview** until shared understanding. One question at a time, in the order of step 5, with a recommendation. Walk every branch; if an answer opens a new one, follow it. Check drafts against the postulates **before** proposing a write.
7. **Confirm:** "Shall I apply these changes?" Only then the Writing gate.
8. **Write** through the Writing gate, metadata refresh included. Park what this request cannot close; delete what this session unblocked. Once every agreed item has passed the gate, delete the source brainstorm. Declined at **Confirm** or stopped before → the brainstorm stays.
9. **Report** by owner: what was written, where, what was parked, and the technical items handed to `/design` with the product rules they serve. Name the deleted brainstorm. When `.sdd/prototypes/<name>.md` exists, say the prototype is still alive and is removed with `/prototype --discard <name>`; do not discard it.

## Scope

The boundary is **Scope** in `specs/AGENTS.md`: a technical item goes to `/design`, which cites the product rule by title.

The interview closes the decision with its branches. `CONTEXT.md` is edited in the same session a term is defined, renamed, or retired; before writing, check the spec uses the canonical term.

### Interview

```markdown
**Question:** [one decision]
**Recommendation:** [option + why]
**Context:** [fact already found, if any]
```

Glossary conflict: point at the `CONTEXT.md` entry and the intended wording. Postulate violation: name the P, the passage, the recommended fix.

Out of scope: the law of the folder in a nested `AGENTS.md`; anything under `src/` — technical specs are `/design`, code is `/implement`; `.metadata.json` by hand; editing a brainstorm (`/brainstorming`).
