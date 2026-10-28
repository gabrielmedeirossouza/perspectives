# Business

Product policy and the mechanism of each business feature live here, plus the policy and mechanism that hold across every one of them.

## Feature

`<feature>` names a business domain boundary that owns product behavior independently, such as invites, notifications, authentication, or external integrations.

Architecture layers, test runners, persistence conventions, shared infrastructure, storage mechanisms, and other technical classifications are not Business features. Rules for those concerns belong to `## Global mechanism rules` when they constrain more than one business feature, or to `development/` when they constrain the form of the code regardless of feature.

Owner examples:

- ID policy, transactions, queues, realtime infrastructure, and shared blob storage are global mechanism rules.
- Vitest, backend layering, and DTO placement are `development/` rules.
- Invite HTTP contracts belong to the invite feature.
- Notification aggregation and realtime contracts belong to the notification feature when they exist only to deliver that domain capability.
- A mechanism that each feature must adopt or reject independently is an adopted pattern, not a global runtime rule.

## Layers

| File | Holds | Does not hold |
|------|--------|----------------|
| `policy.md` | Policy: states, permissions, invariants, who may do what and when | Mechanism: libraries, paths |
| `mechanism.md` | Mechanism: placement, contracts, persistence, libraries, operation | Policy: who may, when it expires, what the screen shows, what the click does |

## Vocabulary

| Layer | Write | Leave out |
|-------|--------|-----------|
| Policy | Ubiquitous English domain prose (`active account`, `email verified`) | Mechanism, surface, layout |
| Mechanism | Package names, placement, HTTP verbs, status codes, env vars, numeric limits | Product who/when/limit, what the screen shows, `or equivalent`, `X or Y` |
