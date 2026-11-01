# UNIVERSAL RULES

THESE RULES APPLY IN EVERY SITUATION:

- **DO NOT BE VERBOSE**
- **BE DIRECT**
- **DO NOT SCATTER INFORMATION ACROSS MULTIPLE PLACES; BE CONCISE AND CLEAR**
- **DO NOT BE REDUNDANT**
- **DO NOT REPEAT YOURSELF**
- **WRITE A LITTLE AND CLEARLY**

THE RULES ABOVE ARE NON-NEGOTIABLE AND APPLY TO ALL ARTIFACTS/OUTPUTS/FILES, INCLUDING CODE, SPECIFICATIONS, RULES, PATTERNS, USER PROMPTS, AGENT PROMPTS, ETC.

---

Product behavior belongs to `specs/` (`specs/AGENTS.md`); technical decisions and code belong to `src/` (`src/AGENTS.md`). Their dependency contract is **Metadata** below.

Read the [code guide](src/specs/code-guide/spec.md) before technical design, implementation, or source audit. It owns coding conventions and examples; other specs cite its rules without restating them.

## Harnesses

Edit **UNIVERSAL RULES** only in this root file. Regenerate its editor copies from the repository root:

```sh
go run -C .agents/tools ./rules --format antigravity > .agents/rules/universal.md
go run -C .agents/tools ./rules --format cursor > .cursor/rules/universal.mdc
go run -C .agents/tools ./rules
```

Cursor, Claude Code (v2.1.277+), Codex, and Antigravity (1.20.5+) read this file and every nested `AGENTS.md`. No `CLAUDE.md`, `CLAUDE.local.md`, or `GEMINI.md` exists in this repository: a `CLAUDE.md` or `CLAUDE.local.md` stops Claude Code from reading `AGENTS.md`.

An `AGENTS.md` owns its tree's structure, permissions, and stable law. Product-specific content lives in sibling files named by theme, referenced from the local `AGENTS.md`; the law never grows with the product. Define each obligation once, at its owner; procedures cite it directly.

Skills live in `.agents/skills/<name>/SKILL.md`, with `.claude/skills/<name>` symlinked to `../../.agents/skills/<name>`. Create or remove both together; edit only the source. Removing a skill also drops it from the config's `skills`, writing the kept list when the field is absent; references elsewhere still describe what the framework offers.

`/<name>` names a skill. Invoke it as `/<name>` in Cursor, Claude Code, and Antigravity, and as `$<name>` in Codex. `$ARGUMENTS` in a skill is the text typed with the invocation; where the harness leaves it literal, read it from the user's message.

A skill that delegates uses the harness's subagent tool — `Task` in Cursor, `Agent` in Claude Code, subagents in Codex, `invoke_subagent` in Antigravity — general-purpose, with the parent's model, this workspace, and a fresh context, never resumed; at most 3 run at once, and the next wave waits for every reply. Diagnostics mean the harness LSP; where the harness has none, the project's type checker and linter.

## Interviews

Discover repository facts before asking. Ask the user one unresolved decision at a time, in dependency order, with a recommendation, its reason, and the cost of the rejected alternative; include context only when useful. Follow branches until shared understanding. Previously recorded decisions remain settled unless current evidence contradicts them. Name rule conflicts by title/path, glossary conflicts by entry, and law violations by ID (P, C, or Vocabulary) and passage; recommend a resolution. Validate drafts against their governing law before requesting the skill's confirmation.

## Working files

`.sdd/` holds configuration and drafts, never normative rules. The flow is `/sdd-brainstorming` → `/sdd-prototype` → `/sdd-spec` → `/sdd-design` → `/sdd-implement`.

- `.sdd/.sdd-config.json` marks the repository root. It records `version`, `remote`, and `paths.specs`, `paths.src`, `paths.technical`; omitted paths use the shipped layout. Bootstrap fixes `paths`; `/sdd-update-framework` advances `version`. `perspectives` and `skills` select what the project keeps; omission means all shipped entries. Removing a file without changing these selections lets the next update restore it. Perspective management is defined in `specs/AGENTS.md`.
- `.sdd/brainstorming/<name>.md`: versioned draft of an idea. `/sdd-brainstorming` writes it, `/sdd-prototype` reads it, `/sdd-spec` consumes and deletes it.
- `.sdd/prototypes/`: git-ignored. One worktree and one manifest per prototype, owned by `/sdd-prototype`.
- `.sdd/reviews/<id>.json`: open applicability-review checkpoint for one source spec, created, updated, and removed on close exclusively by `metadata`.

## Tools

Read `.agents/tools/AGENTS.md` before a mechanical scan, cross-check, or lookup. It owns the tool contract and catalog.

- **Reuse:** run the tool that answers the question; fix wrong or incomplete output in the same session.
- **Create:** turn a manually performed, repeatable deterministic procedure into a tool, run it, and add its catalog entry in that session. Extend an existing tool with a flag for a neighboring question.
- **Judgment:** propose a skill change for recurring decisions that cannot be automated.

Every skill may edit `.agents/tools/` beyond its own scope, except `/sdd-prototype`, which reports tool candidates. Scripts are Go, standard library only, never Python, TypeScript, or JavaScript; the pre-commit hook is the only shell file and only execs a Go tool.

## Metadata

Consumers record the source versions they validated against. Sources never record consumers; `pending` computes work from these links.

### Nodes and coverage

Paths are repository-relative; `specs/`, `src/`, and `src/specs/` stand for the configured `paths`. A node is named by its content path.

| Kind | Content | Metadata | Skeleton writer |
|------|---------|----------|-----------------|
| perspective | `specs/**/spec.md` | `spec.md.metadata.json` | `metadata` |
| technical | `src/specs/**/spec.md` | `spec.md.metadata.json` | `/sdd-design` |
| code file | source file under `src/`, outside the technical root | `<file>.metadata.json` | `/sdd-implement` |
| code directory | non-code files directly in a directory under `src/` | `<dir>/.metadata.json` | `/sdd-implement` |

**Code** is source read by a compiler, interpreter, bundler, or test runner: modules, components, tests, migration scripts. Each has its own node. Other governed files — manifests, lockfiles, settings, assets, fixtures, snapshots — belong in their directory node's `files`. The tool accepts either placement; `/sdd-audit` judges which applies. A `spec.md` under the code root outside the technical root is code.

Every tracked or trackable regular file under `specs/` or `src/` must belong to a node, be metadata, or be **chrome**: any `AGENTS.md`, files directly under `specs/`, and `.md` files directly under `src/`. Other files are drift.

Invalid metadata is also drift: orphan sidecars; sidecars attached to metadata, chrome, or directory paths instead of the directory's `.metadata.json`; `.metadata.json` outside a code directory, including beside a spec; perspective or technical nodes not named `spec.md`; and technical specs lacking their skeleton. A malformed technical node never becomes an implementation unit.

### Fields

Illustration of all fields; the tables below determine which apply:

```json
{
  "version": 2,
  "algorithm": "sha256",
  "hash": "<content hash>",
  "reviewed": "<last applicability-reviewed spec hash>",
  "adopted": "<adopted patterns hash>",
  "modified": "2026-10-29T18:04:12Z",
  "audited": 0,
  "rules": { "<title>": "<body hash>" },
  "sources": [
    {
      "path": "src/specs/code-guide/spec.md",
      "adopted": "<last synced adopted hash>",
      "rules": { "<title>": "<body hash>" },
      "uses": ["<title>"]
    }
  ],
  "files": { "package.json": ["src/specs/toolchain/spec.md"], "favicon.ico": [] }
}
```

A skill authors only `version`, `sources[].path`, `sources[].uses`, and `files`; the tool owns every other field.

| Field | Contract |
|-------|----------|
| `hash` | sha256 of sorted entries `<name>\0<sha256 of bytes>\n`, names relative to the directory. A file node has one entry; a directory node hashes the files it links. Raw bytes, no normalization. |
| `reviewed` | Specs only. Spec `hash` whose applicability across selective consumers was reviewed. Kept equal to `hash` while the spec has no selective consumers and no open checkpoint, initialized on the first selective `--sync` or `--approve` against an intact spec, and preserved by `--write` once selective consumers exist until the review closes. |
| `adopted` | Technical specs and technical → technical `sources` only. sha256 of the trimmed body of `## Adopted patterns`; omitted when absent. |
| `modified` | Changes only when `hash` changes. |
| `rules` | Specs only. Each `###` title outside code fences maps to the sha256 of its body, ending at the next level 1–3 heading, with outer blank lines removed. Titles stay outside the hash so renames differ from body changes. |
| `sources` | Source spec paths, optional `uses` (non-empty list of exact rule titles without duplicates; omit for integral dependency), and tool-owned snapshots (`hash` in integral mode only; `rules` pruned to `uses` on selective `--sync` and `--approve`; `adopted` on technical → technical edges). Paths name `spec.md`, never its directory. |
| `files` | Directory nodes only, at least one entry. Keys name tracked or trackable files directly in that directory, excluding metadata, chrome, and files with their own nodes. Values list sources of this node; an empty list awaits a governing spec through audit. `sources[].uses` on the directory node holds the union of rules applicable to its linked files. |
| `audited` | Code nodes only; defined in **Audit**. |

| Consumer | Sources |
|----------|---------|
| perspective | None; omit `sources`. |
| technical | Perspective specs it serves and technical specs it depends on, integral or selective via `uses`. A global technical spec lists no perspective. |
| code | At least one technical spec, every technical spec it obeys directly, and every perspective rule those specs serve, integral or selective via `uses`. A test lists the tested file's sources and selections plus applicable rules of its test runner's technical spec. |

Product changes reach code through directly listed perspectives, never through technical dependency chains. Required perspective coverage is the union of technical → perspective selections across the code node's listed technical specs (integral if any listed technical spec serves the perspective integrally). Listing a perspective integrally when technical support is selective, or selecting unserved rules, is drift; omitting required served rules is `missing` coverage.

### States

Computed by `pending`, never stored:

| State | Condition | Owner |
|-------|-----------|-------|
| drift | Invalid nodes/fields/coverage above; missing, unreadable, or mismatched metadata; duplicate rule titles; empty, duplicate, case-mismatched, or unknown `uses`; forbidden sources; a code node listing no technical spec, or a perspective/rule none of its technical specs serves; technical dependency cycles; a TODO gap matching no rule, or an Audit gap naming no code node that owes an audit; broken spec links; invalid or orphan checkpoints in `.sdd/reviews/`. | Writer of the affected artifact. |
| stale | Integral edge: source `hash`, `rules`, or technical `adopted` changed. Selective edge: a selected rule was added, changed, removed, or renamed, or technical `adopted` changed; snapshot entries outside `uses` that still exist in the source are ignored as verdict reductions. Also, a code node omits a served perspective or required perspective rule (`missing`). | `/sdd-design` for technical nodes; `/sdd-implement` for code. |
| dangling | A source path names no node. A directory path gets a hint naming its spec file. | Same as stale. |
| unconsumed | A perspective has no technical consumer or has rules no technical consumer selects; a feature technical spec has no code consumer or has rules no code consumer selects (with its served perspective rules still lacking a code consumer); a global technical spec has no code consumer. | `/sdd-design`; `/sdd-implement`, respectively. |
| review | A spec has an open checkpoint in `.sdd/reviews/` or has selective consumers with `hash != reviewed`. | Source writer (`/sdd-spec` or `/sdd-design`) to classify and close; `/sdd-design` for technical candidates; `/sdd-audit` for code candidates. |
| waits | Code depends on a stale/dangling technical spec, upstream included; a source has blocking drift; or every rule of a unit is parked by a Design gap. | Design, the drift's owner, or the gap's owner first. |
| fresh | None of the above. For code, sync certifies implementation and tests through the Implementation gate. | — |

**Links:** a markdown link in a spec must resolve, with exact case, to a tracked or trackable existing path and, when present, a heading in its markdown target. Its writer fixes drift before syncing that spec: `/sdd-spec` for perspective specs, `/sdd-design` for technical specs. Broken links alone do not block consumers' syncs. A `#anchor` link from a selected rule to an unselected rule is surfaced as a hint in the audit dossier.

**Blocking drift:** content or metadata that is not what it claims prevents sync against that source. Code also checks upstream technical specs. A stale/dangling technical spec, including a `# Design` entry, waits when one of its own sources has this drift.

**Units:** one directory's pending code nodes and the union of their rules, or one technical spec with no code consumer (or unconsumed feature rules). A unit waits if every pending node waits; runnable units skip waiting nodes. Batch ordering and size belong to `/sdd-implement`'s **Continue**.

### Audit

`audited` counts approvals of a code node's source list and `uses` selections at its current hash. The judge verifies that each listed technical and perspective selection covers every applicable rule and none is missing; for directory nodes, it judges each file's links and consolidates the union of applicable rules in `sources[].uses`. Coverage is not an audit approval.

The count resets to 0 when content changes, or when `--sync` records a newly listed source, a newly added title in `uses` (except a confirmed rename with identical body hash), or a selective → integral transition (`uses` omitted with empty `hash`). Verdict reductions do not reset `audited`, and `--approve` prunes `sources[].rules` to `uses` and clears selective `hash`. Approval requires no drift and no implementation unit. Only 0 versus ≥ 1 affects work; larger counts are informational.

`pending` lists nodes at 0 under `# Audit`:

- **Blocked:** invalid or mismatched content/metadata, including invalid links or sources.
- **After implementation:** the node is an implementation unit.
- **Parked:** an Audit gap names the node, a linked file (parking only its directory node), or an ancestor directory.
- **Owed:** ready for judgment. Missing technical sources, unselected applicable rules, and ungoverned linked files go to the judge; `/sdd-audit` drops unserved perspectives/rules or blocks when an applicable perspective rule lacks technical support; none blocks judgment.

### Writers

All commands below run as `go run -C .agents/tools ./metadata <flags>`.

| Writer | Writes | Metadata operation |
|--------|--------|--------------------|
| `/sdd-spec` | Product specs | `--write <dir>`; `--review <spec> --scope …` and `--close` |
| `/sdd-design` | Technical specs and their skeletons | `--sync <node…>`; `--review <spec> --scope …`, `--verdict … <tech-cand>`, `--apply <tech-cand…>`, and `--close` |
| `/sdd-implement` | Code and its skeletons (`uses` included) | `--write <dir>` after writes; `--sync <node…>` after the Implementation gate |
| `/sdd-audit` | Code sources, `uses`, file links on a verdict; Audit gaps | `--write <node…>` for edits; `--approve <node…>` for approved nodes; `--review <spec> --verdict … <code-cand>` and `--apply <code-cand…>`; never sync |
| pre-commit | Stages refreshed metadata; validates `.sdd/reviews/` | `--staged`; never sync |

A target is a content path (a linked file selects its directory node), a metadata path, or a directory (its node and descendants). `--sync` and `--approve` require one node per target; use `<dir>/.metadata.json` to select a directory node alone.

Sync refuses broken links in the target spec, dangling or forbidden sources, blocking source drift, removed/renamed/invalid titles in `uses`, a code source list without a technical spec or not matching the perspective coverage its technical specs serve, and code waiting on design.

Enable `.githooks/pre-commit` with `git config core.hooksPath .githooks`; Go is required. It hashes the working tree, including unstaged portions of staged files, and rejects staged files without nodes and orphan metadata left by deletions or renames. Commit grouping belongs to `/sdd-commit`.
