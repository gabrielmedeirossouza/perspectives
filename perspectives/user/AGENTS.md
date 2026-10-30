# User

What the user sees and does lives here, plus the behavior that holds across every surface.

## Feature

`<feature>` names a surface boundary that owns visible behavior independently, by screen, flow, or what the user can see, on whichever channel delivers it. A channel (web, mobile, CLI) is not a feature; a feature owns its behavior on every channel that delivers it.

UI stacks, design tokens, component conventions, layouts, form infrastructure, accessibility mechanisms, and other technical classifications are not User features and not User rules. They are technical specs under `src/specs/`.

Owner examples:

- Routes, states, copy, and interactions of a page belong to that surface feature.
- A behavior every surface shares — every destructive action asks for confirmation, every list has an empty state — is a global rule in `user/spec.md`.
- The component library, the design tokens, and the HTTP client the surfaces use are technical specs under `src/specs/`, serving these rules by title.

## Scope

| `spec.md` holds | `spec.md` does not hold |
|-----------------|-------------------------|
| Surface: routes, clicks, flow states, UI copy, layout | Product who/when/limit. Technical: libraries, placement, contracts |

## Vocabulary

| Write | Leave out |
|-------|-----------|
| Ubiquitous English surface prose (`empty list`, `confirm deletion`) | Technical names, product who/when/limit |
