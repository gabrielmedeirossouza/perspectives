# Development

The mechanism that constrains how code is written, organized, built, and verified lives here, regardless of which rule that code materializes.

## Scope

This perspective owns global rules only. It has no `<feature>/` directory and no `## Global policy rules` slot. How developers work is process and lives in `src/AGENTS.md`; what the code is made of lives here.

A mechanism rule belongs here when it constrains the form of the code and would hold with no feature in the repository: language and compiler settings, runtime, package manager, test runner, lint and formatting, module layering of the source tree, placement that applies to every module.

A mechanism rule that names what a perspective's features run on or deliver through — persistence, queues, realtime, UI stack, design tokens, deployment foundations, observability — belongs to the `## Global mechanism rules` of that perspective, not here.

Owner examples:

- TypeScript settings, the Node runtime, the package manager, Vitest, ESLint, backend module layering, and DTO placement are rules of this perspective.
- Transactions, queues, and blob storage are Business global mechanism rules. UI stack and design tokens are User global mechanism rules. Orchestrator, environments, and observability are Operator global mechanism rules.
- A mechanism that each module must adopt or reject independently is an adopted pattern, not a global rule.

A rule here serves no single policy rule and cites none. Other perspectives cite a rule here by title, with a link.

## Layers

| Slot | Holds | Does not hold |
|------|--------|----------------|
| `## Global mechanism rules` | Mechanism: toolchain, runtime, test runner, lint, code layering, placement that applies to every module | Policy of any kind — a policy item never classifies here. What a perspective's features run on or deliver through |

## Vocabulary

| Layer | Write | Leave out |
|-------|--------|-----------|
| Mechanism | Package names, versions, file placement, command entry points, numeric limits | Product who/when/limit, what the screen shows, who may ship, `or equivalent`, `X or Y` |
