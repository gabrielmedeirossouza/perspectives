# Operator

Operation policy of each operational feature lives here, plus the policy that holds across every one of them.

## Feature

`<feature>` names a delivery-lifecycle or operational boundary that owns policy independently, such as environment promotion, rollback, or incident response.

Image conventions, pipeline foundations, environment conventions, observability infrastructure, runtime, and other technical classifications are not Operator features and not Operator rules. They are technical specs under `src/specs/`.

Owner examples:

- Who may ship to production, what gates a promotion, the rollback window, and what an incident obliges belong to their operational feature.
- A policy every operational feature shares — every change to production is traceable to a person — is a global rule in `operator/spec.md`.
- The orchestrator, the image, the pipeline steps, and the observability stack are technical specs under `src/specs/`, serving these rules by title.

## Scope

| `spec.md` holds | `spec.md` does not hold |
|-----------------|-------------------------|
| Policy: who may ship to an environment, what gates a promotion, the rollback window, what an incident obliges | Technical: tools, files, commands. What the product does, what the screen shows |

## Vocabulary

| Write | Leave out |
|-------|-----------|
| Ubiquitous English operations prose (`production environment`, `rollback`, `promotion`) | Tool names, file paths, commands |
