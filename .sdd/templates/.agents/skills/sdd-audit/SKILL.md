---
name: sdd-audit
description: Audit code nodes' governing specs with an independent judge per directory. Correct source and file links, park ungoverned code, and record approvals. Use for pending audit work or an explicit source audit.
---

# sdd-audit

Read root **Metadata**, C6 in `src/AGENTS.md`, and the tool catalog. Tools check structural validity; judges decide which specs govern each file and whether it belongs in its own node or a directory node. The parent never substitutes its judgment for a judge.

## Input

`$ARGUMENTS`: `[--all] [<path>...]`, passed to `pending --audit`; paths select code nodes, metadata files, or directories under `src/`. Without paths, every node owing an audit.

## Procedure

1. Run `go run -C .agents/tools ./pending --audit [--all] [path...]`. Keep skipped nodes and their standing for the report.
2. Send each directory's dossiers to a fresh, read-only judge under **Judge**.
3. Apply each verdict under **Corrections**. Refresh edited nodes with one `metadata --write <node...>` call; never `--write` an unedited node. `unchanged` is success.
4. Park BLOCK verdicts under Audit gaps in `TODO.md`, using its shape in `specs/AGENTS.md`. For a directory node, name the blocked linked file. Remove gaps this audit closes.
5. Approve eligible nodes under **Approval** with one `metadata --approve <node...>` call.
6. Report per node: verdict, applied corrections/approval, skips and blockers. Route gaps to `/sdd-design`, stale nodes to `/sdd-implement continue`, UNMET rules as `/sdd-implement <spec>:<title>`, and NOTEs and placement DROPs, applied or not, with the cited rule to `/sdd-implement`. UNMET is reported only; approval certifies the source list.

Completion means every judged node is approved, corrected, or parked, with remaining work reported.

## Corrections

- Apply ADD/DROP to a file node's `sources[].path`, or to `files[<file>]` for a linked file. Add newly linked technical specs to the directory node's sources.
- If DROPs would leave a file node without a technical source, do not apply that verdict. A placement DROP means misplaced code; otherwise request a new judge for that node, quoting the verdict and requiring BLOCK or KEEP. A repeated invalid verdict is reported as not judged.
- A linked file may lose its last spec: apply the DROPs and park that file as BLOCK.
- Remove from judged directory nodes any technical source that no linked file names.
- After all edits, remove from every judged node each perspective none of its remaining technical sources serves; then run `pending` and add the perspectives it reports missing.

## Approval

Approve file nodes with an OK verdict or only DROP corrections. Approve directory nodes only when every linked file is OK, has only DROPs, or adds specs already listed and synced by that node, and none was parked.

An ADD introducing a new node source leaves it stale: implementation validates and syncs it before another audit. If the tool refuses approval because of drift, pending implementation, or changed content, report the reason; do not retry around it.

## Judge

Use one subagent (root **Harnesses**) per directory; even a single-node directory gets an independent judge.

Supply the directory's `pending --audit` group unchanged, including skipped nodes as context; absolute paths to node content and metadata, listed technical specs, and candidates that name a node or are global; the remaining candidates' paths and purposes with instructions to request fitting specs. Include root **Metadata** and C6. Perspective specs follow from technical sources and are not sent.

The judge reads the supplied material and edits nothing. For directory nodes, judge each linked file by its own path, never the directory as a whole:

| Finding | Criterion |
|---------|-----------|
| KEEP | A spec listed for this file governs it. |
| ADD | Another spec governs this file and its placement puts it here; a global spec needs only the governing rule. Adding obligates implementation of its rules. |
| DROP | No rule applies to this file, or a cited placement/boundary rule puts it elsewhere. |
| UNMET | A listed rule applies but is not implemented. Never use DROP for missing implementation. |
| BLOCK | No listed spec or candidate governs the file; name the missing decision. |
| NOTE | Code linked as a non-code file, or non-code with its own sidecar; report placement for implementation, never apply it here. |

Require this output only:

```text
VERDICT <file node or linked file>: ok | corrections | block
KEEP  src/specs/<a>/spec.md — rule "<title>" governs it
ADD   src/specs/<c>/spec.md — rule "<title>" governs it; placement rule "<title>" puts it here | global
DROP  src/specs/<b>/spec.md — rule "<title>" places it in src/<other> | no rule applies: <reason>
UNMET src/specs/<a>/spec.md — rule "<title>" is not implemented (/sdd-implement)
BLOCK — no technical spec governs it; it would decide <what>
NOTE  <file> — <placement correction for /sdd-implement>
```

One VERDICT per file. OK has KEEP lines, optionally UNMET; corrections has at least one ADD/DROP; block has only BLOCK. NOTE lines come last, outside verdicts. Ask once to repair malformed output; a second miss is not judged.

Permissions: code metadata's `sources[].path` and `files`, Audit gaps, metadata and pending tools; never code, specs, tool-owned fields, or `--sync`.
