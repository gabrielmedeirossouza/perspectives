---
name: audit
description: >-
  Audits the sources of the code nodes under src/ — every code node lists at
  least one technical spec, each listed spec governs its code, and no
  technical spec the code obeys is missing — with one fresh-context judge
  per node, corrects sources[].path on the verdict, parks the code no spec
  governs, and records the approval in audited so an unchanged node is not
  judged twice. Use when pending lists code nodes under # Audit, after
  /implement wrote or changed code nodes, or when the user types /audit.
---

# audit

Reach the root `AGENTS.md` (**Metadata**, its **Audit** paragraph included), `src/AGENTS.md` (C6), and `.agents/tools/AGENTS.md` before judging. What a code node lists, what `audited` means, and who owns each state live there — not here.

The deterministic part is the tool's: a source exists, is of a kind the node may list, at least one is a technical spec, and the perspective nodes follow from the technical ones. The judgment is a subagent's: does each listed technical spec govern this code, and does the code obey a technical spec it does not list. This skill never judges in its own window.

## User Input

```text
$ARGUMENTS
```

## Arguments

```text
<arg> ::= [--all] [<dir>...]
<dir> ::= a code node under src/, or a directory holding code nodes
```

Examples: `/audit`; `/audit src/backend/invite`; `/audit src/backend`; `/audit --all`.

No argument = every code node that owes an audit. `--all` also judges the nodes already approved and the parked ones; an approval on an approved node adds one to its count.

## Steps

**Done when:** every targeted node a judge read is approved, corrected, or parked; corrections are written; the user has the report.

1. **Reach** the three files above.
2. **Inventory:** `go run -C .agents/tools ./pending --audit [--all] [dir...]`. The nodes it does not judge — blocked by drift, after `/implement`, parked — go to the report for their owner. Each dossier names the files of the node, the listed technical specs with their rule titles, the listed perspective specs, the candidate technical specs it does not list in tiers — names the directory, global, the rest — with their purpose, and the listed perspectives none of its technical specs serves.
3. **Judge** (below): one fresh-context subagent per dossier, read-only, waves of up to 3. A single node still goes to a judge: the unanchored read is the point.
4. **Apply**, for every judged node: drop from `sources[].path` each perspective the dossier names as served by none of its technical specs. For a `corrections` verdict, also edit the technical specs — `ADD` and `DROP` — then run `go run -C .agents/tools ./pending` and add each perspective its `# Implement` line names as a `missing source` of the node. A verdict whose `DROP`s would leave no technical spec is not applied: the code is misplaced — report it to `/implement` with the placement rule the judge cited. Then `go run -C .agents/tools ./metadata --write <dir...>`, one call for every node edited, and no `--write` on a node that was not edited: an approval is refused on a node whose content changed since its metadata was last written. Never code, never `--sync`, never a spec file.
5. **Park** each `block` under `## Audit gap` of `TODO.md` (shape in `perspectives/AGENTS.md`): what the missing technical spec would decide, the code node, the blocker. Delete the items this session closed — a parked node a judge now approved or corrected.
6. **Approve:** `go run -C .agents/tools ./metadata --approve <dir...>`, one call, for every node whose verdict was `ok` and for every corrected node whose corrections were only `DROP`s — its remaining list is the one the judge kept. A node with an `ADD` lists a source with no recorded hash, so it is stale and is not approved: `/implement continue` validates the code against that spec's rules and syncs it, and the next `/audit` approves it. The tool refuses a node with drift, an `/implement` unit, or one whose content changed since its metadata was last written; such a node goes to the report, not to a retry.
7. **Report** per node: approvals, verdict, corrections applied, `UNMET` rules as `/implement <spec>:<title>` for the user to run — the audit approves the list, not the coverage, and nothing tracks an `UNMET` rule after the report — misplaced code for `/implement`, gaps parked for `/design`, nodes skipped and why, drift for its owner, and the nodes now stale for `/implement continue`.

## Judge

**Prompt** — self-contained: absolute paths of every file of the node, of the `spec.md` of every listed technical spec and of every candidate that names the directory or is global, the path and purpose of every other candidate with "ask for its spec by name if it fits", the **Metadata** section of the root `AGENTS.md` with its **Audit** paragraph, and C6 of `src/AGENTS.md`. Perspective specs stay out: they follow from the technical ones. The judge reads everything, edits nothing, and answers in the shape below and nothing else.

**Criteria.** `KEEP`: a rule of the spec governs code in this node. `ADD`: a rule of the candidate governs code in this node and the candidate's placement puts that code here; adding creates the obligation to implement every rule of it. `DROP`: a placement or boundary rule of the spec puts this code in another node — cite it; "no code obeys it" is not a `DROP`. `UNMET`: a rule of a listed spec applies to this node and no code implements it — a C1 failure for `/implement`, not a reason to drop. `BLOCK`: no listed spec and no candidate governs this code — name what the missing spec would decide.

```text
VERDICT: ok | corrections | block
KEEP  src/specs/<a> — rule "<title>" governs <file>
ADD   src/specs/<c> — rule "<title>" governs <file>; placement rule "<title>" puts it here
DROP  src/specs/<b> — rule "<title>" places this code in src/<other>
UNMET src/specs/<a> — rule "<title>" has no code in this node (/implement)
BLOCK — no technical spec governs this code; it would decide <what>
```

`ok` carries only `KEEP` lines, `UNMET` allowed. `corrections` carries at least one `ADD` or `DROP`. `block` carries one `BLOCK` line and no `ADD`.

## Orchestration

The agent that receives `/audit` is the **parent**: it runs the tools, builds the prompts, applies the verdicts, writes `TODO.md`, and reports. Judges are subagents (the harness's subagent tool, named in the root `AGENTS.md`; general-purpose, the parent's model, this workspace, fresh context, never resumed), read-only, up to 3 launched together, the next wave after every judge of the previous one has answered. A judge that answers out of shape is asked once more with the shape; a second miss is reported as not judged.

Permissions: `sources[].path` of code nodes' `.metadata.json`; `TODO.md` (`## Audit gap`); run the metadata and pending tools. Out of scope: code; `src/specs/` (`/design`); `perspectives/` (`/spec`); any hash or `audited` by hand; `metadata --sync` (`/implement`).
