# Business

Product policy of each business feature lives here, plus the policy that holds across every one of them.

## Feature

`<feature>` names a business domain boundary that owns product behavior independently, such as invites, notifications, authentication, or external integrations.

Architecture layers, test runners, persistence, shared infrastructure, storage, and other technical classifications are not Business features and not Business rules. They are technical specs under `src/specs/`.

Owner examples:

- Who may invite, when an invite expires, and what makes an account active belong to the invite and account features.
- A rule that holds for every business feature — every mutation is audited, every record belongs to one workspace — is a global rule in `business/spec.md`.
- The HTTP contract of an invite, its persistence, and the queue that delivers a notification are technical specs under `src/specs/`, serving these rules by title.

## Scope

| `spec.md` holds | `spec.md` does not hold |
|-----------------|-------------------------|
| Policy: states, permissions, invariants, who may do what and when | Surface: what the screen shows, what the click does. Technical: libraries, placement, contracts, persistence |

## Vocabulary

| Write | Leave out |
|-------|-----------|
| Ubiquitous English domain prose (`active account`, `email verified`) | Technical names, surface, layout |
