---
name: sdd-audit
description: Audit code nodes' governing specs with an independent judge per directory. Correct source and file links, park ungoverned code, and record approvals. Use for pending audit work or an explicit source audit.
---

# sdd-audit

Read root **Metadata**, C6 in `src/AGENTS.md`, and the tool catalog. Tools check structural validity; judges decide which specs and `###` rules govern each file and whether it belongs in its own node or a directory node. The parent never substitutes its judgment for a judge.

## Input

`$ARGUMENTS`: `[--all] [--review <spec>] [<path>...]`, passed to `pending --audit`; paths select code nodes, metadata files, or directories under `src/`. Without paths, every node owing an audit (or the pending code candidates of `<spec>`'s open review checkpoint when `--review <spec>` is given).

## Procedure

1. Run `go run -C .agents/tools ./pending --audit [--all] [--review <spec>] [path...]`. Keep skipped nodes and their standing for the report.
2. Send each directory's dossiers to a fresh, read-only judge under **Judge**.
3. For candidates of an open review checkpoint, save each verdict first with `metadata --review <spec> --verdict … <node>`, then run `metadata --review <spec> --apply <node...>`. If remaining **Corrections** change other sources or directory `files`, apply them, refresh edited nodes, verify the final links against the judge's verdict, and record/apply the verdict again to bind that evidence. For other audited nodes, apply **Corrections** and refresh edited nodes with one `metadata --write <node...>` call; never `--write` an unedited node. `unchanged` is success.
4. Park BLOCK verdicts under Audit gaps in `TODO.md`, using its shape in `specs/AGENTS.md` (and for review candidates where an applicable perspective rule lacks technical support, record `--verdict block --reason …` for `/sdd-design`). For a directory node, name the blocked linked file. Remove gaps this audit closes.
5. Approve eligible nodes under **Approval** with one `metadata --approve <node...>` call. If a review checkpoint now has all candidates resolved without blocks, close it (`metadata --review <spec> --close`) or report ready-to-close to its owner.
6. Report per node: verdict, applied corrections/approval, review checkpoint progress, skips and blockers. Route gaps and blocked perspective rules to `/sdd-design`, stale nodes to `/sdd-implement continue`, UNMET rules as `/sdd-implement <spec>:<title>`, and NOTEs and placement DROPs, applied or not, with the cited rule to `/sdd-implement`. UNMET is reported only; approval certifies the source and `uses` selections.

Completion means every judged node is approved, corrected, or parked, with remaining work reported.

## Corrections

- Apply ADD/KEEP/DROP to a file node's `sources[].path` and `sources[].uses` (setting `uses` to the exact applicable rule titles, or omitting `uses` when all rules apply), or to `files[<file>]` for a linked file. Consolidate the union of applicable rules across linked files in `sources[].uses` on the directory node.
- If DROPs would leave a file node without a technical source, do not apply that verdict. A placement DROP means misplaced code; otherwise request a new judge for that node, quoting the verdict and requiring BLOCK or KEEP. A repeated invalid verdict is reported as not judged.
- A linked file may lose its last spec: apply the DROPs and park that file as BLOCK.
- Remove from judged directory nodes any technical source that no linked file names.
- After all edits, remove from every judged node each perspective or perspective rule none of its remaining technical sources serves, and align perspective `uses` with the rules served by its technical specs and applicable to the code. If an applicable perspective rule is not served by any technical spec, do not add the unserved link; record a review block / Audit gap for `/sdd-design`.

## Approval

Approve file nodes with an OK verdict or only DROP / `uses` reduction corrections. Approve directory nodes only when every linked file is OK, has only DROPs/reductions, or adds specs and `uses` rules already listed and synced by that node, and none was parked. `--approve` prunes `sources[].rules` to `uses` and clears selective `hash`.

An ADD introducing a new source or a new rule in `uses` without a snapshot leaves the node stale: implementation validates and syncs it before another audit. If the tool refuses approval because of drift, pending implementation, or changed content, report the reason; do not retry around it.

## Judge

Use one subagent (root **Harnesses**) per directory; even a single-node directory gets an independent judge.

Supply the directory's `pending --audit` group unchanged, including skipped nodes, unselected rules, `Rule link hints`, and `Review targets`; absolute paths to node content and metadata, listed technical and perspective specs, and candidates that name a node or are global; the remaining candidates' paths and purposes with instructions to request fitting specs. Include root **Metadata** and C6.

The judge reads the supplied material and edits nothing. For directory nodes, judge each linked file by its own path, never the directory as a whole:

| Finding | Criterion |
|---------|-----------|
| KEEP | A listed spec governs this file; list the exact applicable `###` rule titles (`uses`). |
| ADD | Another spec or an unselected rule of a listed spec governs this file and its placement puts it here; a global spec needs only the governing rule. Adding a rule obligates its implementation. |
| DROP | No rule of the spec (or a listed rule in `uses`) applies to this file, or a cited placement/boundary rule puts it elsewhere. |
| UNMET | An applicable rule is not implemented. Never use DROP for missing implementation. |
| BLOCK | No listed spec or candidate governs the file, or an applicable perspective rule lacks technical spec support; name the missing decision. |
| NOTE | Code linked as a non-code file, or non-code with its own sidecar; report placement for implementation, never apply it here. |

Require this output only:

```text
VERDICT <file node or linked file>: ok | corrections | block
KEEP  <spec-path> — uses: "<title>", "<title>"
ADD   <spec-path> — uses: "<title>"; placement rule "<title>" puts it here | global | perspective
DROP  <spec-path> [rule "<title>"] — rule "<title>" places it in src/<other> | no rule applies: <reason>
UNMET <spec-path> — rule "<title>" is not implemented (/sdd-implement)
BLOCK — no technical spec governs it; it would decide <what>
NOTE  <file> — <placement correction for /sdd-implement>
```

One VERDICT per file. OK has KEEP lines matching current `uses`, optionally UNMET; corrections has at least one ADD, DROP, or `uses` change; block has only BLOCK. NOTE lines come last, outside verdicts. Ask once to repair malformed output; a second miss is not judged.

Permissions: code metadata's `sources[].path`, `sources[].uses`, and `files`, Audit gaps, metadata and pending tools; never code, specs, tool-owned fields, or `--sync`.
