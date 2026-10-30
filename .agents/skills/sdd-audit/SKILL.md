---
name: sdd-audit
description: >-
  Audits the sources of the code nodes under src/ — every code node lists at
  least one technical spec, each listed spec governs its code, each linked
  file obeys the specs it is linked to, and no technical spec the code obeys
  is missing — with one fresh-context judge per directory, corrects
  sources[].path and files on the verdict, parks the code no spec governs,
  and records the approval in audited so an unchanged node is not judged
  twice. Use when pending lists code nodes under # Audit, after
  /sdd-implement wrote or changed code nodes, or when the user types /sdd-audit.
---

# sdd-audit

Reach the root `AGENTS.md` (**Metadata**, its **Audit** paragraph included), `src/AGENTS.md` (C6), and `.agents/tools/AGENTS.md` before judging. What a code node lists, what `audited` means, and who owns each state live there — not here.

The deterministic part is the tool's: a source exists, is of a kind the node may list, at least one is a technical spec, a link names a listed source, and the perspective nodes follow from the technical ones. The judgment is a subagent's: does each listed technical spec govern this code, does each linked file obey the specs it is linked to, does the code obey a technical spec it does not list, and is each file placed where it belongs — as a node of its own when it is code, as a link when it is not (**Code** under **Metadata** in the root `AGENTS.md`). This skill never judges in its own window.

## User Input

```text
$ARGUMENTS
```

## Arguments

```text
<arg> ::= [--all] [<path>...]
<path> ::= a code node under src/ — a file, or <dir>/.metadata.json for a directory node — or a directory holding code nodes
```

Examples: `/sdd-audit`; `/sdd-audit src/backend/invite/invite.ts`; `/sdd-audit src/.metadata.json`; `/sdd-audit src/backend`; `/sdd-audit --all`.

No argument = every code node that owes an audit. `--all` also judges the nodes already approved and the parked ones; an approval on an approved node adds one to its count.

## Steps

**Done when:** every targeted node a judge read is approved, corrected, or parked; corrections are written; the user has the report.

1. **Reach** the three files above.
2. **Inventory:** `go run -C .agents/tools ./pending --audit [--all] [path...]`. It groups the targeted nodes by directory: the standing of each node it does not judge — blocked by drift, after `/sdd-implement`, parked, approved without `--all` — which goes to the report for its owner, and one dossier per node it does. Each dossier names the node's file, or the files a directory node links with the specs each is linked to, the listed technical specs with their rule titles, the listed perspective specs, the candidate technical specs it does not list in tiers — names the node, global, the rest — with their purpose, and the listed perspectives none of its technical specs serves.
3. **Judge** (below): one fresh-context subagent per directory, read-only, waves of up to 3; it reads every dossier of the directory and gets the nodes not judged as context, with their standing. A directory with a single node still goes to a judge: the unanchored read is the point.
4. **Apply**, for every judged node: drop from `sources[].path` each perspective the dossier names as served by none of its technical specs. For a `corrections` verdict, also edit the technical specs — `ADD` and `DROP` on `sources[].path` of a file node; on `files[<file>]` of the directory node for a linked file, adding to `sources[].path` of the directory node a spec its links come to name — then run `go run -C .agents/tools ./pending` and add each perspective its `# Implement` line names as a `missing source` of the node. Every `DROP` that cites a placement rule is also reported to `/sdd-implement` with that rule, applied or not: code of that spec may still sit in this file. A verdict whose `DROP`s would leave a file node with no technical spec is not applied: with a placement `DROP` the file is misplaced, reported as above; otherwise no spec governs it, which is a `BLOCK` — launch a new judge for that node alone, quoting the verdict and asking for `BLOCK` or `KEEP`; a second answer of the same kind is reported as not judged. A linked file whose `DROP`s empty its list takes them, and its path is parked as a `block` (step 5). After editing `files` of a directory node, drop from its `sources[].path` each technical spec its links named before the edit and name no more. After every edit, drop each perspective none of the node's remaining technical specs serves — the dossier was computed before the verdict. Then `go run -C .agents/tools ./metadata --write <node...>`, one call naming every node edited, and no `--write` on a node that was not edited: an approval is refused on a node whose content changed since its metadata was last written; `unchanged` means the edit moved no hash and stands. Never code, never `--sync`, never a spec file; a `NOTE` is reported, never applied.
5. **Park** each `block` under `## Audit gap` of `TODO.md` (shape in `specs/AGENTS.md`): what the missing technical spec would decide, the owner — the file node, or the linked file's path, which parks the directory node that links it and nothing else — and the blocker. Delete the items this session closed — a parked node a judge now approved or corrected.
6. **Approve:** `go run -C .agents/tools ./metadata --approve <node...>`, one call naming one node per target (`<dir>/.metadata.json` for a directory node), for every file node whose verdict was `ok` or whose corrections were only `DROP`s — its remaining list is the one the judge kept — and for every directory node whose every linked file's verdict was `ok`, `DROP`s only, or `ADD`s of specs the node already listed and had synced, and none of whose linked files this session parked. A node with an `ADD` that enters `sources` lists a source with no recorded hash, so it is stale and is not approved: `/sdd-implement continue` validates the code against that spec's rules and syncs it, and the next `/sdd-audit` approves it. The tool refuses a node with drift, an `/sdd-implement` unit, or one whose content changed since its metadata was last written; such a node goes to the report, not to a retry.
7. **Report** per node: approvals, verdict, corrections applied, `UNMET` rules as `/sdd-implement <spec>:<title>` for the user to run — the audit approves the list, not the coverage, and nothing tracks an `UNMET` rule after the report — misplaced code and files placed wrong (`NOTE`) for `/sdd-implement`, gaps parked for `/sdd-design`, nodes skipped and why, drift for its owner, and the nodes now stale for `/sdd-implement continue`.

## Judge

**Prompt** — self-contained: the directory's group from `pending --audit` as is; absolute paths of every file and metadata file of every node of the directory, of the `spec.md` of every listed technical spec and of every candidate that names a node or is global, the path and purpose of every other candidate with "ask for its spec by name if it fits", the **Metadata** section of the root `AGENTS.md` with its **Code** and **Audit** paragraphs, and C6 of `src/AGENTS.md`. Perspective specs stay out: they follow from the technical ones. The judge reads everything, edits nothing, and answers in the shape below and nothing else.

**Criteria**, per node — a file node, or each file a directory node links, judged by its own path, so a directory node gets one verdict per linked file and none of its own. `KEEP`: a rule of the spec governs this node. `ADD`: a rule of the candidate governs this node and the candidate's placement puts it here — a global candidate, which places nothing, needs only the governing rule; adding creates the obligation to implement every rule of it. `DROP`: no rule of the spec governs this node — either a placement or boundary rule of the spec puts it elsewhere, cited, or no rule of the spec applies to what this file is or does, said in a phrase; a global spec is dropped the same way from a node it does not constrain. `UNMET`: a rule of a listed spec applies to this node and no code implements it — a C1 failure for `/sdd-implement`. The two never overlap: a rule that applies and has no code yet is `UNMET`, never a `DROP` — "no code obeys it" is not a `DROP` — and a rule that does not apply to this file is never `UNMET`. `BLOCK`: no listed spec and no candidate governs this node — name what the missing spec would decide. `NOTE`: a file placed wrong — code linked in `files`, or a file that is not code with metadata of its own — for `/sdd-implement`; never a correction.

```text
VERDICT <file node, or linked file>: ok | corrections | block
KEEP  src/specs/<a>/spec.md — rule "<title>" governs it
ADD   src/specs/<c>/spec.md — rule "<title>" governs it; placement rule "<title>" puts it here | global
DROP  src/specs/<b>/spec.md — rule "<title>" places it in src/<other> | no rule applies: <what this file is or does>
UNMET src/specs/<a>/spec.md — rule "<title>" is not implemented (/sdd-implement)
BLOCK — no technical spec governs it; it would decide <what>
NOTE  <file> — is not code; link it in <dir>/.metadata.json | is code; give it <file>.metadata.json (/sdd-implement)
```

One `VERDICT` per file node and per linked file, its lines under it; `NOTE` lines come last and belong to no verdict. `ok` carries only `KEEP` lines, `UNMET` allowed. `corrections` carries at least one `ADD` or `DROP`. `block` carries one `BLOCK` line and nothing else.

## Orchestration

The agent that receives `/sdd-audit` is the **parent**: it runs the tools, builds the prompts, applies the verdicts, writes `TODO.md`, and reports. Judges are subagents (the harness's subagent tool, named in the root `AGENTS.md`; general-purpose, the parent's model, this workspace, fresh context, never resumed), read-only, one per directory, up to 3 launched together, the next wave after every judge of the previous one has answered. A judge that answers out of shape is asked once more with the shape; a second miss is reported as not judged.

Permissions: `sources[].path` and `files` of code nodes' metadata; `TODO.md` (`## Audit gap`); run the metadata and pending tools. Out of scope: code; `src/specs/` (`/sdd-design`); `specs/` (`/sdd-spec`); any hash or `audited` by hand; `metadata --sync` (`/sdd-implement`).
