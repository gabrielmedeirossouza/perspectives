# Perspectives

The perspectives of Ledger, a personal finance web app where a member tracks accounts, transactions, and budgets: one directory per actor of the system, each with its own law in `AGENTS.md`. Reach this file to know which perspectives exist; reach a perspective's `AGENTS.md` for its feature boundary, scope test, and vocabulary.

- **Business** (`specs/business/`): product policy of each business feature. Product policy lives here, not on the member surface.
- **Member** (`specs/member/`): what the member sees and does — screens, flows, copy. Screen behavior lives here, not in business.
- **Operator** (`specs/operator/`): how the system is shipped and run — environments, deployment, observability — as policy. Nothing the product does for its members lives here.
