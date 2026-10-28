# Web

Screen behavior and the mechanism of each web feature live here, plus the behavior and mechanism that hold across every one of them.

## Feature

`<feature>` names a screen boundary that owns visible behavior independently, by page, flow, or what the user can see.

Shared UI stacks, design tokens, component conventions, layouts, form infrastructure, accessibility mechanisms, and other technical classifications are not Web features. Rules for those concerns belong to `## Global tech rules` when they constrain more than one screen boundary.

Owner examples:

- Shared UI stack, design tokens, HTTP client conventions, authenticated layouts, and accessibility mechanisms are global tech rules.
- Page-specific routes, states, copy, interactions, and their exclusive delivery mechanisms belong to that screen feature.
- A mechanism that each screen feature must adopt or reject independently is an adopted pattern, not a global runtime rule.

## Layers

| File | Holds | Does not hold |
|------|--------|----------------|
| `functional-spec.md` | Policy: routes, clicks, flow states, UI copy, layout | Mechanism: libraries, paths |
| `tech-spec.md` | Mechanism: placement, contracts, persistence, libraries, operation | Policy: who may, when it expires, what the screen shows, what the click does |

## Vocabulary

| Layer | Write | Leave out |
|-------|--------|-----------|
| Functional | Ubiquitous English surface prose (`empty list`, `confirm deletion`) | Mechanism, product who/when/limit |
| Tech | Package names, placement, HTTP verbs, status codes, env vars, numeric limits | Product who/when/limit, what the screen shows, `or equivalent`, `X or Y` |
