Product specs (`specs/`) are the product source of truth: what the system does, for whom, one perspective per actor. Technical specs (`src/specs/`) say how it is built. Code (`src/`) materializes both. Reach `specs/AGENTS.md` for product behavior or specs, `src/AGENTS.md` for technical specs or code. Every spec file, every code file, and every directory whose `.metadata.json` links the files that are not code is a node of **Metadata**; `pending` is the observer that reads them.

## Harnesses

Cursor, Claude Code (v2.1.277+), Codex, and Antigravity (1.20.5+) read this file and every nested `AGENTS.md`. No `CLAUDE.md`, `CLAUDE.local.md`, or `GEMINI.md` exists in this repository: a `CLAUDE.md` or `CLAUDE.local.md` stops Claude Code from reading `AGENTS.md`.

An `AGENTS.md` holds the structure of its tree: what lives where, who writes what, and the law that does not change with the product. What grows or varies with the product — which perspectives exist, global rules, the glossary, parked work, bug autopsies, testing guidelines — lives in a sibling file named by its theme (`PERSPECTIVES.md`, `spec.md`, `CONTEXT.md`, `TODO.md`, `BUG-AUTOPSIES.md`, `INCREMENTAL-TESTING-GUIDELINES.md`), referenced once from the `AGENTS.md` beside it. An `AGENTS.md` never grows with the product.

Skills live once, in `.agents/skills/<name>/SKILL.md`. Claude Code reads only `.claude/skills/`, so `.claude/skills/<name>` is a symlink to `../../.agents/skills/<name>`. A new skill gets both; edits go to `.agents/skills/`. A skill this project does not want is removed with its symlink and dropped from `skills` of `.sdd/.sdd-config.json`; the law and the other skills keep naming it, as what the framework offers, and `/sdd-update-framework` brings it back.

`/<name>` names a skill. Invoke it as `/<name>` in Cursor, Claude Code, and Antigravity, and as `$<name>` in Codex. `$ARGUMENTS` in a skill is the text typed with the invocation; where the harness leaves it literal, read it from the user's message.

A skill that delegates uses the harness's subagent tool: `Task` in Cursor, `Agent` in Claude Code, subagents in Codex, `invoke_subagent` in Antigravity. Diagnostics mean the harness LSP; where the harness has none, the project's type checker and linter.

## Working files

`.sdd/` holds the configuration the tools read and the working files of the flow `/sdd-brainstorming` → `/sdd-prototype` → `/sdd-spec` → `/sdd-design` → `/sdd-implement`, never rules. `specs/` and `src/specs/` stay the source of truth; a brainstorm never overrides a rule.

- `.sdd/.sdd-config.json`: the framework version this repository is on, the remote it came from, and the layout the tools read: `paths.specs` (the product specs), `paths.src` (the code), `paths.technical` (the technical specs). The file marks the repository root for every tool; a path left out takes the layout the framework ships. `perspectives` names each perspective of this product with the framework kind it came from, mirrored by `specs/PERSPECTIVES.md` and written by `/sdd-spec` (**Perspectives** in `specs/AGENTS.md`), absent for every shipped kind; `skills` names the skills kept, absent for all; the two are the one record of what this project keeps from the framework, so a file removed by hand comes back at the next update; `/sdd-update-framework` moves `version`; `paths` is set at the bootstrap and not moved afterwards.
- `.sdd/brainstorming/<name>.md`: versioned draft of an idea. `/sdd-brainstorming` writes it, `/sdd-prototype` reads it, `/sdd-spec` consumes and deletes it.
- `.sdd/prototypes/`: git-ignored. One worktree and one manifest per prototype, owned by `/sdd-prototype`.

## Tools

`.agents/tools/` holds procedures agents turned into code, in Go, so a later session runs them instead of redoing them by hand. Its law and catalog are in `.agents/tools/AGENTS.md`. Reach it before any mechanical procedure over the repository — a scan, a cross-check, a lookup.

- **Reuse.** A catalog entry answers the question → run it; do not redo it by hand. Wrong or incomplete output → fix the tool in the same session; do not work around it.
- **Create.** A session that performs by hand a deterministic procedure — same inputs, same answer, no judgment — that a later session will repeat, such as a skill step or a scan over `specs/`, writes it as a tool, runs it against the repository, and adds its catalog row, in the same session.
- **One tool per question.** A neighboring question becomes a flag of the existing tool, not a second tool.
- **Judgment is not a tool.** A recurring procedure that needs judgment goes in the report as a proposed skill change.
- **Read-only.** A tool reports; the skill writes. Two exceptions: metadata, which `metadata` writes beyond the skeleton a skill authors (`version`, `sources[].path`, and `files`), and the files of the framework with `version` of `.sdd/.sdd-config.json`, which `update --apply` writes for `/sdd-update-framework`.

Scripts and tooling of this repository are written in Go, standard library only; none is Python, TypeScript, or JavaScript. The pre-commit hook is the one shell file, a shim whose only command execs a Go tool. Every skill may create and edit files under `.agents/tools/` on top of its own permissions. `/sdd-prototype` is the exception: it names tool candidates in its report.

## Metadata

Sync between a spec and what consumes it is not written by hand. Every consumer records the version of each source it last validated against; a tool computes what is stale. Nothing records what consumes it.

**Node** = a file with `<file>.metadata.json` beside it, or a directory under `src/` whose `.metadata.json` links the files of the directory that are not code — a **directory node**. Kinds, by place in the layout of `.sdd/.sdd-config.json`: **perspective** = a `spec.md` under the specs root (`specs/`), whose metadata the tool creates; **technical** = a `spec.md` under the technical root (`src/specs/`), whose skeleton `/sdd-design` writes; **code** = any other file under the code root (`src/`), whose skeleton `/sdd-implement` writes, or a directory node. A node is named by its path — `specs/business/invite/spec.md`, `src/api/auth/db.ts`, `src/api` for a directory node — and its metadata is that path plus `.metadata.json`: `src/api/.metadata.json` is the directory's. A perspective or technical node that is not `spec.md`, a `.metadata.json` beside a spec — under `specs/` or `src/specs/`, where the node of a spec is `spec.md.metadata.json` — a `<file>.metadata.json` with no `<file>` beside it, and a metadata file of a metadata file, of chrome, or of a directory are drift; a malformed technical node is drift, never a unit. A `spec.md` under the code root outside the technical root is code.

**Code** is what the compiler, interpreter, bundler, or test runner reads as source — a module, a component, a test, a migration script — and every code file is a node of its own; a test lists the sources of the file it tests, plus the technical spec of the test runner. Every other file under `src/` a spec governs — a manifest, a lockfile, compiler or bundler settings, a static asset, a fixture, a snapshot — is not code: it has no metadata of its own and is linked in `files` of the `.metadata.json` of its directory. The tool accepts either placement, because telling code from configuration is judgment; the judge of `/sdd-audit` reports a file placed wrong.

**Content** of a file node = the file. Of a directory node = the files `files` names. **Hash** = sha256 of the sorted entries `<name>\0<sha256 of the bytes>\n` of the content, `<name>` relative to the directory; a file node has one entry. Raw bytes, no normalization. **Rules**, on a perspective or technical node: each `###` title outside code fences → sha256 of its body, the lines up to the next heading of level 1–3 with blank lines at both ends removed. The title stays outside its hash, so a rename is told from a change.

**Coverage:** every regular file git tracks or would track under `specs/` or `src/` is one of: the file of a node, a `*.metadata.json`, a file a directory node links, or chrome — `AGENTS.md` anywhere, the files directly under `specs/`, the `.md` files directly under `src/`. Any other file is drift: in no node.

**Sources** a node may list: a technical node lists the perspective nodes it serves and the technical nodes it depends on; a code node lists at least one technical node — every technical node whose rules its code obeys directly, not only the one it is named after — and every perspective node those serve, so a product-only change reaches the code; a perspective node lists none. A source names the spec file (`src/specs/toolchain/spec.md`); a path that names its directory is dangling, and the tool says which file was meant. A product change reaches code only through the code nodes that list that perspective; it does not travel through technical dependencies, which is why a technical node no code node lists is pending.

Every field, for illustration; which ones a node carries depends on its kind, as stated below the example:

```json
{
  "version": 2,
  "algorithm": "sha256",
  "hash": "<hash of this node>",
  "modified": "2026-10-29T18:04:12Z",
  "audited": 0,
  "rules": { "<title>": "<sha256 of the body>" },
  "sources": [
    { "path": "specs/business/invite/spec.md", "hash": "<hash the source had at the last sync>", "rules": { "<title>": "<hash>" } }
  ],
  "files": { "package.json": ["src/specs/toolchain/spec.md"], "favicon.ico": [] }
}
```

`modified` moves only when `hash` moves. `audited` is present only on a code node (**Audit** below). `rules` is present only on a perspective or technical node; `sources` is absent on a perspective node and on a global technical node, which lists none. `files` is present only on a directory node, which links at least one file: each key is the name of a tracked or trackable file directly in the directory — not a metadata file, not chrome, not a file with metadata of its own — and each value lists sources the node lists, or is empty while no spec governs the file yet: the file is covered, and the judge names its spec or `/sdd-audit` parks an audit gap. `sources` of a directory node may list more than `files` names, such as the perspective nodes its technical specs serve. A skill writes only the skeleton — `version`, `sources[].path`, and `files`; every hash and `audited` are the tool's. The file is never edited by hand beyond those fields.

**States**, computed by `pending`, never stored:

| State | Means | Owed by |
|-------|-------|---------|
| drift | a metadata file missing, orphan (no file beside it), unreadable, or not matching its content; a technical `spec.md` without its skeleton; a perspective or technical node that is not `spec.md`; a `.metadata.json` outside the code root; a duplicate rule title; a link in a spec that names no file, or a heading the markdown file it names does not hold — a title removed or renamed; `files` on a node that is no directory node, absent on one, or naming a file that is no file of the directory, chrome, a node of its own, or a source the node does not list; a source a node may not list; a code node listing no technical spec; a perspective node a code node lists that none of its technical specs serves; a file under `specs/` or `src/` in no node; technical specs that source each other in a cycle; a gap in `TODO.md` that matches no rule, or an audit gap naming no code node that owes an audit | whoever last wrote there: `metadata --write`, or fix the sources or the links; a broken link in a spec, its writer — `/sdd-design` in a technical spec, `/sdd-spec` in a perspective spec |
| stale | a source's current hash differs from the recorded one; the rule delta names what was added, changed, removed, or renamed. On a code node, also: a perspective node its technical specs serve that it does not list yet | `/sdd-design` on a technical node, `/sdd-implement` on a code node |
| dangling | a listed source is no node | same as stale |
| unconsumed | a perspective node no technical node lists; a technical node no code node lists | `/sdd-design`; `/sdd-implement` |
| waits | a code node whose technical spec, or one upstream of it, is stale or dangling; a unit whose every rule a `## Design gap` item parks | `/sdd-design` first |
| fresh | none of the above; on a code node it means implemented and tested, because only a passed Implementation gate syncs it | — |

`pending` lists what `/sdd-implement` owes as **units**: one unit is one directory with its pending code nodes — the directory node and the file nodes directly in it — and the union of their rules, or one technical spec no code node lists. A unit waits when every pending node in it waits; a node that waits inside a unit that runs is listed and skipped.

**Audit**, orthogonal to the states: a code node is `fresh` and may still owe an audit. `audited` counts the approvals `/sdd-audit` recorded at the node's current `hash`: a judge read the code and confirmed that every technical spec the node lists governs it and that no technical spec it obeys is missing — on a directory node, that each linked file is governed by the specs it is linked to. It returns to 0 when `hash` moves and when a sync records a source listed since the last sync; an approval is recorded only on a node that has no drift and is no `/sdd-implement` unit, and the count then persists until one of those resets it. `pending` lists every code node at 0 under `# Audit` — owed to `/sdd-audit`; blocked while its content or metadata is not what it claims (a hash or rule map that does not match, an unreadable file, an invalid source or link); after `/sdd-implement` when the node is one of its units; or parked under `## Audit gap` of `TODO.md` because no technical spec governs the code yet — the gap names a code node, a linked file, which parks the directory node that links it, or a directory, which parks every code node at or under it. A node listing no technical spec, a file linked to no spec, or a perspective none of its technical specs serves, is owed: the judge names the spec, and `/sdd-audit` drops the perspective. `pending --audit` prints the dossiers a judge needs, grouped by directory: one judge per directory. Only 0 against ≥ 1 is read; the count is informational.

**Writers:**

| Writer | Writes | Then runs |
|--------|--------|-----------|
| `/sdd-spec` | `specs/**/spec.md` | `metadata --write <dir>` |
| `/sdd-design` | `src/specs/<name>/spec.md` and the `sources[].path` of its node | `metadata --sync <dir>`, also to acknowledge a stale source that needs no change |
| `/sdd-implement` | code, the `sources[].path` of its code nodes — a new node, a perspective node its technical specs came to serve, a dangling source dropped — and `files` of its directory nodes | `metadata --write <dir>` after every write; `metadata --sync <node…>` only after the Implementation gate, on the nodes that passed it |
| `/sdd-audit` | the `sources[].path` and `files` of a code node on a judge's verdict — a technical spec the code obeys added, one that governs none of it dropped, a link corrected, and the perspective nodes that follow — and `## Audit gap` of `TODO.md` | `metadata --write <node…>` on a node it changed; `metadata --approve <node…>` on a node the judge approved as listed |
| pre-commit hook | — | `metadata --staged`: refreshes and stages the metadata of every node holding a staged file, never a sync. It hashes the working tree, so a partially staged file is hashed as it is on disk. It refuses the commit when a staged file is in no node — a code file added without its metadata, a metadata file deleted without its file — and when a renamed file left its metadata behind as an orphan |

A tool takes a node — its path or its metadata file — or a directory, which names its directory node and every node at or under it; `--sync` and `--approve` take one node per target and refuse a directory that names several, so a directory node whose directory holds other nodes is named by its metadata file (`src/api/.metadata.json`). The hook is `.githooks/pre-commit`; enable it once per clone with `git config core.hooksPath .githooks`. It needs Go. A metadata file whose own hash moved travels in the commit of the content it hashes; a sync that changed only its recorded sources — an acknowledgement, a revalidation — or an approval or a `sources` edit, which never moves a hash, is a commit of its own, type `docs` — unless the file was never committed, when it travels whole with its content. A `files` edit that links or unlinks a file, by `/sdd-audit` or by `/sdd-implement`, moves the hash because the content of the directory node changed; it is still a commit of its own, unless a linked file changed with it, when it travels with that file.
