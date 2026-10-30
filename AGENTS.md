Product specs (`specs/`) are the product source of truth: what the system does, for whom, one perspective per actor. Technical specs (`src/specs/`) say how it is built. Code (`src/`) materializes both. Reach `specs/AGENTS.md` for product behavior or specs, `src/AGENTS.md` for technical specs or code. Every spec directory and every code directory `/sdd-implement` declares is a node of **Metadata**; `pending` is the observer that reads them.

## Harnesses

Cursor, Claude Code (v2.1.277+), Codex, and Antigravity (1.20.5+) read this file and every nested `AGENTS.md`. No `CLAUDE.md`, `CLAUDE.local.md`, or `GEMINI.md` exists in this repository: a `CLAUDE.md` or `CLAUDE.local.md` stops Claude Code from reading `AGENTS.md`.

An `AGENTS.md` holds the structure of its tree: what lives where, who writes what, and the law that does not change with the product. What grows or varies with the product — which perspectives exist, global rules, the glossary, parked work, bug autopsies, testing guidelines — lives in a sibling file named by its theme (`PERSPECTIVES.md`, `spec.md`, `CONTEXT.md`, `TODO.md`, `BUG-AUTOPSIES.md`, `INCREMENTAL-TESTING-GUIDELINES.md`), referenced once from the `AGENTS.md` beside it. An `AGENTS.md` never grows with the product.

Skills live once, in `.agents/skills/<name>/SKILL.md`. Claude Code reads only `.claude/skills/`, so `.claude/skills/<name>` is a symlink to `../../.agents/skills/<name>`. A new skill gets both; edits go to `.agents/skills/`.

`/<name>` names a skill. Invoke it as `/<name>` in Cursor, Claude Code, and Antigravity, and as `$<name>` in Codex. `$ARGUMENTS` in a skill is the text typed with the invocation; where the harness leaves it literal, read it from the user's message.

A skill that delegates uses the harness's subagent tool: `Task` in Cursor, `Agent` in Claude Code, subagents in Codex, `invoke_subagent` in Antigravity. Diagnostics mean the harness LSP; where the harness has none, the project's type checker and linter.

## Working files

`.sdd/` holds working files of the flow `/sdd-brainstorming` → `/sdd-prototype` → `/sdd-spec` → `/sdd-design` → `/sdd-implement`, never rules. `specs/` and `src/specs/` stay the source of truth; a brainstorm never overrides a rule.

- `.sdd/.sdd-config.json`: the framework version this repository is on, the remote it came from, and the layout the tools read: `paths.specs` (product specs, `specs/`), `paths.src` (code, `src/`), `paths.technical` (technical specs, `src/specs/`). The file marks the repository root for every tool; a path left out takes the default in parentheses.
- `.sdd/brainstorming/<name>.md`: versioned draft of an idea. `/sdd-brainstorming` writes it, `/sdd-prototype` reads it, `/sdd-spec` consumes and deletes it.
- `.sdd/prototypes/`: git-ignored. One worktree and one manifest per prototype, owned by `/sdd-prototype`.

## Tools

`.agents/tools/` holds procedures agents turned into code, in Go, so a later session runs them instead of redoing them by hand. Its law and catalog are in `.agents/tools/AGENTS.md`. Reach it before any mechanical procedure over the repository — a scan, a cross-check, a lookup.

- **Reuse.** A catalog entry answers the question → run it; do not redo it by hand. Wrong or incomplete output → fix the tool in the same session; do not work around it.
- **Create.** A session that performs by hand a deterministic procedure — same inputs, same answer, no judgment — that a later session will repeat, such as a skill step or a scan over `specs/`, writes it as a tool, runs it against the repository, and adds its catalog row, in the same session.
- **One tool per question.** A neighboring question becomes a flag of the existing tool, not a second tool.
- **Judgment is not a tool.** A recurring procedure that needs judgment goes in the report as a proposed skill change.
- **Read-only.** A tool reports; the skill writes. The one exception is `.metadata.json`, which `metadata` writes beyond the skeleton a skill authors: `version` and `sources[].path`.

Scripts and tooling of this repository are written in Go, standard library only; none is Python, TypeScript, or JavaScript. The pre-commit hook is the one shell file, a shim whose only command execs a Go tool. Every skill may create and edit files under `.agents/tools/` on top of its own permissions. `/sdd-prototype` is the exception: it names tool candidates in its report.

## Metadata

Sync between a spec and what consumes it is not written by hand. Every consumer records the version of each source it last validated against; a tool computes what is stale. Nothing records what consumes it.

**Node** = a directory with `.metadata.json`. Under `specs/`, every directory holding `spec.md` is a node, and the tool creates its file. Under `src/`, a node is declared by the skill that writes its `.metadata.json`. Kinds, by place in the layout of `.sdd/.sdd-config.json`: **perspective** under the specs root (`specs/`), **technical** under the technical root (`src/specs/`), **code** elsewhere under the code root (`src/`). `src/` itself may be a code node: the one that holds the root manifests and configuration a global technical spec decides (`go.mod`, `package.json`, lockfiles, compiler settings).

**Content** = every regular file git tracks or would track (not ignored) under the node's directory, recursively, except `.metadata.json`, `AGENTS.md`, the files directly under `src/` ending in `.md` (chrome), and the subtrees of other nodes. **Hash** = sha256 of the sorted entries `<path relative to the node>\0<sha256 of the bytes>\n`. Raw bytes, no normalization. **Rules**, when the node holds `spec.md`: each `###` title outside code fences → sha256 of its body, the lines up to the next heading of level 1–3 with blank lines at both ends removed. The title stays outside its hash, so a rename is told from a change.

**Sources** a node may list: a technical node lists the perspective nodes it serves and the technical nodes it depends on; a code node lists at least one technical node — every technical node whose rules its code obeys directly, not only the one it is named after — and every perspective node those serve, so a product-only change reaches the code; a perspective node lists none. A product change reaches code only through the code nodes that list that perspective; it does not travel through technical dependencies, which is why a technical node no code node lists is pending.

Every field, for illustration; which ones a node carries depends on its kind, as stated below the example:

```json
{
  "version": 1,
  "algorithm": "sha256",
  "hash": "<hash of this node>",
  "modified": "2026-10-29T18:04:12Z",
  "audited": 0,
  "rules": { "<title>": "<sha256 of the body>" },
  "sources": [
    { "path": "specs/business/invite", "hash": "<hash the source had at the last sync>", "rules": { "<title>": "<hash>" } }
  ]
}
```

`modified` moves only when `hash` moves. `audited` is present only on a code node (**Audit** below). `rules` is absent without `spec.md`; `sources` is absent on a perspective node. A skill writes only the skeleton — `version` and `sources[].path`; every hash and `audited` are the tool's. The file is never edited by hand beyond those fields.

**States**, computed by `pending`, never stored:

| State | Means | Owed by |
|-------|-------|---------|
| drift | a `.metadata.json` missing, orphan (no `spec.md` beside it), unreadable, or not matching its content; a duplicate rule title; a source a node may not list; a code node listing no technical spec; a perspective node a code node lists that none of its technical specs serves; a file under `src/` outside any node; technical specs that source each other in a cycle; a gap in `TODO.md` that matches no rule, or an audit gap naming no code node that owes an audit | whoever last wrote there: `metadata --write`, or fix the sources |
| stale | a source's current hash differs from the recorded one; the rule delta names what was added, changed, removed, or renamed. On a code node, also: a perspective node its technical specs serve that it does not list yet | `/sdd-design` on a technical node, `/sdd-implement` on a code node |
| dangling | a listed source is no node | same as stale |
| unconsumed | a perspective node no technical node lists; a technical node no code node lists | `/sdd-design`; `/sdd-implement` |
| waits | a code node whose technical spec, or one upstream of it, is stale or dangling; a unit whose every rule a `## Design gap` item parks | `/sdd-design` first |
| fresh | none of the above; on a code node it means implemented and tested, because only a passed Implementation gate syncs it | — |

**Audit**, orthogonal to the states: a code node is `fresh` and may still owe an audit. `audited` counts the approvals `/sdd-audit` recorded at the node's current `hash`: a judge read the code and confirmed that every technical spec the node lists governs it and that no technical spec it obeys is missing. It returns to 0 when `hash` moves and when a sync records a source listed since the last sync; an approval is recorded only on a node that has no drift and is no `/sdd-implement` unit, and the count then persists until one of those resets it. `pending` lists every code node at 0 under `# Audit` — owed to `/sdd-audit`; blocked while its content or metadata is not what it claims (a hash or rule map that does not match, an unreadable file, an invalid source); after `/sdd-implement` when the node is one of its units; or parked under `## Audit gap` of `TODO.md` because no technical spec governs the code yet. A node listing no technical spec, or a perspective none of its technical specs serves, is owed: the judge names the spec, and `/sdd-audit` drops the perspective. `pending --audit` prints the dossier a judge needs. Only 0 against ≥ 1 is read; the count is informational.

**Writers:**

| Writer | Writes | Then runs |
|--------|--------|-----------|
| `/sdd-spec` | `specs/**/spec.md` | `metadata --write <dir>` |
| `/sdd-design` | `src/specs/<name>/spec.md` and the `sources[].path` of its node | `metadata --sync <dir>`, also to acknowledge a stale source that needs no change |
| `/sdd-implement` | code and the `sources[].path` of its code nodes: a new node, a perspective node its technical specs came to serve, a dangling source dropped | `metadata --write` on a new node; `metadata --sync <dir>` only after the Implementation gate |
| `/sdd-audit` | the `sources[].path` of a code node on a judge's verdict — a technical spec the code obeys added, one whose placement puts the code elsewhere dropped, and the perspective nodes that follow — and `## Audit gap` of `TODO.md` | `metadata --write <dir>` on a node it changed; `metadata --approve <dir>` on a node the judge approved as listed |
| pre-commit hook | — | `metadata --staged`: refreshes and stages the metadata of every node holding a staged file, never a sync. It hashes the working tree, so a partially staged file is hashed as it is on disk |

The hook is `.githooks/pre-commit`; enable it once per clone with `git config core.hooksPath .githooks`. It needs Go. A `.metadata.json` whose own hash moved travels in the commit of the content it hashes; a sync that changed only its recorded sources — an acknowledgement, a revalidation — or an approval or sources edit of `/sdd-audit`, which never moves a hash, is a commit of its own, type `docs`.
