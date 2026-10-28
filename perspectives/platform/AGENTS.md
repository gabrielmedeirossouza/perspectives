# Platform

Operation policy and the mechanism of each platform feature live here, plus the policy and mechanism that hold across every one of them.

## Feature

`<feature>` names a delivery-lifecycle or operational boundary that owns policy independently, such as environment promotion, rollback, or incident response.

Repository-wide runtime choices, toolchains, image conventions, pipeline foundations, environment conventions, observability infrastructure, and other technical classifications are not Platform features. Rules for those concerns belong to `## Global tech rules` when they constrain more than one delivery or operational boundary.

Owner examples:

- Shared runtime, deployment foundations, environment conventions, and observability infrastructure are global tech rules.
- Promotion, rollback, and incident obligations belong to their independently owned operational feature.
- A mechanism that each operational feature must adopt or reject independently is an adopted pattern, not a global runtime rule.

## Layers

| File | Holds | Does not hold |
|------|--------|----------------|
| `functional-spec.md` | Policy: who may ship to an environment, what gates a promotion, the rollback window, what an incident obliges | Mechanism: tools, files, commands. What the product does, what the screen shows |
| `tech-spec.md` | Mechanism: orchestrator, images, command entry points, pipeline steps, env vars, placement of delivery files | Policy: who may ship, when a promotion is allowed. What the product does, what the screen shows |

## Vocabulary

| Layer | Write | Leave out |
|-------|--------|-----------|
| Functional | Ubiquitous English operations prose (`production environment`, `rollback`, `promotion`) | Tool names, file paths, commands |
| Tech | Tool and package names, file placement, command entry points, env vars, numeric limits | Who may ship and when, `or equivalent`, `X or Y` |
