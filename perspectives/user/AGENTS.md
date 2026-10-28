# User

What the user sees and does, and the mechanism of each user-facing feature, live here, plus the behavior and mechanism that hold across every one of them.

## Feature

`<feature>` names a surface boundary that owns visible behavior independently, by screen, flow, or what the user can see, on whichever channel delivers it. A channel (web, mobile, CLI) is not a feature; a feature owns its behavior on every channel that delivers it.

Shared UI stacks, design tokens, component conventions, layouts, form infrastructure, accessibility mechanisms, and other technical classifications are not User features. Rules for those concerns belong to `## Global mechanism rules` when they constrain more than one surface boundary, or to `development/` when they constrain the form of the code regardless of feature.

Owner examples:

- Shared UI stack, design tokens, HTTP client conventions, authenticated layouts, and accessibility mechanisms are global mechanism rules.
- Page-specific routes, states, copy, interactions, and their exclusive delivery mechanisms belong to that surface feature.
- A mechanism that each surface feature must adopt or reject independently is an adopted pattern, not a global runtime rule.

## Layers

| File | Holds | Does not hold |
|------|--------|----------------|
| `policy.md` | Surface: routes, clicks, flow states, UI copy, layout | Mechanism: libraries, paths. Product who/when/limit |
| `mechanism.md` | Mechanism: placement, contracts, persistence, libraries, operation | Policy: who may, when it expires, what the screen shows, what the click does |

## Vocabulary

| Layer | Write | Leave out |
|-------|--------|-----------|
| Policy | Ubiquitous English surface prose (`empty list`, `confirm deletion`) | Mechanism, product who/when/limit |
| Mechanism | Package names, placement, HTTP verbs, status codes, env vars, numeric limits | Product who/when/limit, what the screen shows, `or equivalent`, `X or Y` |
