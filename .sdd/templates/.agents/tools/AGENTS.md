# Tools

The root `AGENTS.md` defines when to reuse or create a tool.

## Contract

- Go 1.22+, standard library only, one module in this directory; no dependencies, `go.sum` entries, or committed binaries.
- One lower-kebab-case directory per tool, with `main.go`; shared code belongs in `internal/`. `main.go` opens with a one-line comment stating the question the tool answers; this catalog owns its flags and output.
- Run `go run -C .agents/tools ./<tool> [flags]` from the repository root. From elsewhere, adjust `-C`. Tools find the nearest ancestor with `.sdd/.sdd-config.json`; `--root <absolute-dir>` overrides it. Input/output paths are root-relative; configured `paths` replace default roots.
- Tools report; skills write. Exceptions: `metadata` writes sidecars, manages `.sdd/reviews/<id>.json` checkpoints, and stages sidecars with `--staged`; `update --apply` writes framework files and the recorded version.
- Output is plain text. Exit 0 on success, 1 with a reason on stderr on failure, 2 on a usage error; `go run` collapses failure codes to 1.
- Rules govern implementations. Fix disagreement in the same session; change parsers alongside their formats.
- Tests live beside code, using standard `testing`. Run `go vet ./...` and `go test ./...` before committing. Remove obsolete tools with their catalog entries.

## Catalog

Every tool also accepts `--root`.

### `rules [--format antigravity|cursor]`

Do the editor copies match the root UNIVERSAL RULES? Prints `ok <path>` per copy. `--format` prints one generated copy for redirection (**Harnesses** in the root `AGENTS.md`).

### `metadata [--write | --sync | --approve | --staged | --review <spec> [...]] [node|dir...]`

Does each node's metadata match its content? Targets follow **Writers** in the root `AGENTS.md`; without targets, the check and `--write` cover every node.

- No flag: lists the targeted nodes' drift, files at or under the targets in no node, metadata files that are no node, and invalid review checkpoints.
- `--write`: recomputes the tool-owned fields (`hash`, `modified`, `rules`, `adopted`, `reviewed`, `audited`), creating perspective sidecars; never touches `sources` or `files`.
- `--sync <node…>`: `--write`, then records each source's current snapshot (`hash` and full `rules` in integral mode; pruned `rules` for `uses` with `hash` omitted in selective mode; `adopted` on technical → technical edges); refusals are in root **Writers**.
- `--approve <node…>`: `--write`, prunes selective `sources[].rules` to `uses` and clears selective `hash`, then adds one approval to `audited`.
- `--staged`: `--write` on every node holding a staged file, validates `.sdd/reviews/`, then stages sidecars; takes no target.
- `--review <spec>`: inspects or operates the applicability review of `<spec>`:
  - No sub-flag: prints the Git diff since `reviewed` (rule deltas, `## Adopted patterns`, chrome, affected rules) and checkpoint status.
  - `--scope preserved|expanded|unknown [path...]`: classifies the change since `reviewed`; closes immediately when `preserved` and no checkpoint is open, else opens/updates `.sdd/reviews/<id>.json` preserving valid judgments. An invalid checkpoint requires `unknown`, rebuilding every potential candidate without reusing lost evidence.
  - `--verdict keep|add|reduce|exclude|block [--uses <title>...] [--reason <text>] <candidate>`: saves a candidate judgment in the checkpoint before metadata edits.
  - `--apply <candidate...>`: idempotently applies the saved verdict to each candidate's metadata and marks it applied in the checkpoint.
  - `--close`: verifies all candidates are judged, valid, applied, and unblocked at the current spec hash, records `reviewed` on `<spec>`, and removes `.sdd/reviews/<id>.json`.

Review judgments bind candidate content, all source selections and directory file assignments, and the content and links of supporting sources. Changed evidence invalidates dependent judgments; pending implementation alone does not prevent closure. Checkpoint writes and closure compare the saved checksum under a per-source process lock (`.sdd/.review-<id>.lock`). Concurrent operations fail and require reload; after a terminated writer, remove its leftover lock directory only after verifying no writer remains.

Output: one line per node — `ok`, `written`, `synced`, `approved` with the count, `unchanged` (success), `staged`, or review status (`reviewed`, `checkpoint`, `verdict`, `applied`, `closed review`) — or its drift; `no node under …` when none exists.

### `pending [--batch | --audit [--all] [--review <spec>] [node|dir...]]`

What work is pending, and for whom?

- `# Design`: perspective specs no technical spec serves or with unconsumed rules; stale technical specs with rule/`adopted` deltas and the Spec gap parking one; dangling sources.
- `# Implement`: units (root **Units**), sources first, with rule deltas, unconsumed feature rules, parked rules, and whether each runs or waits.
- `# Review`: specs owing an applicability review with source owner, affected rules, saved checkpoint progress, and next candidate per consumer writer.
- `# Next batch`: the units **Continue** of `/sdd-implement` takes now.
- `# Audit`: code nodes at 0 approvals with their standing (root **Audit**).
- `# Drift`: each drift with its owner.

`No pending work.` when every section is empty. `--batch` prints only the next batch in full with waiting units and why — or, when nothing is implementable, what `/sdd-design` owes — then drift. `--audit` groups targeted code nodes by directory: the standing of each not judged, and one dossier per node to judge — its file or linked files with their specs, listed technical and perspective specs with selected (`uses`) and unselected rules, `Rule link hints` for `#anchor` links from selected to unselected rules, active `Review targets`, unlisted candidate technical specs in tiers (names the node, global, the rest) with their purposes, and listed perspectives/rules none of its technical specs serves (`Perspectives to drop`). `--review <spec>` restricts `--audit` to code candidates of `<spec>`'s open review checkpoint, skipping valid applied judgments unless `--all` is given. `--all` adds approved and parked nodes.

### `update [--apply] [--from <dir>]`

What must change for this project to hold the framework's version? `--from` reads a local checkout relative to the project root, its working tree as incoming and its history as base; otherwise a temporary clone of config `remote`.

Mapping, applied alike to base and incoming templates so the merge keeps project adaptations:

- A template in a kind directory of the framework's specs root goes to `<paths.specs>/<name>/` for every perspective whose `from` is that kind, and is skipped when none is; templates under the framework's technical and code roots go to `paths.technical` and `paths.src`; a skill missing from a present `skills` is skipped; the config is never copied.
- In `.md` files, the framework's root and kind-directory paths become the project's in one pass, longest first. A kind with zero or several perspectives keeps its path, listed `by hand`.
- A config `from` or skill the framework does not ship is an error.

The base is the source's commit at the recorded version, or the newest below it. Version 0 makes every template an `add`; an older incoming version is refused.

The plan prints `version N -> M` with source and base, one line per file — `add`, `update`, `merge`, `conflict`, `delete`, `skip` — and `by hand` work. `--apply` writes it and records M: `delete` only removes files the project left unchanged; a `conflict` either holds markers or is `kept` unwritten; it exits 1 while a conflict remains.
