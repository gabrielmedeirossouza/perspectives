# Business

## Feature

A business domain boundary owns behavior independently, such as invites, accounts, or notifications.

## Scope

| Holds | Leaves out |
|-------|------------|
| Policy: states, permissions, invariants, who may do what and when | Visible behavior; architecture layers, persistence, storage, and queues |

## Vocabulary

| Write | Leave out |
|-------|-----------|
| English domain prose: `active account`, `email verified` | Technical names, surface, layout |
