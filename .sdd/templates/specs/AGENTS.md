# Specs

This tree holds the product specifications. They are the single source of truth for what the system does and for whom.

A perspective is documentation written from the viewpoint of one actor of the system. The actors of this product, and what lives in each perspective, are the catalog `PERSPECTIVES.md` (**Perspectives** below); each nested `AGENTS.md` is the law of one perspective: feature boundary, scope test, vocabulary.

How any of it is built — stack, placement, contracts, persistence, toolchain — is a technical rule and lives in `src/specs/`, written by `/sdd-design` under `src/AGENTS.md`. No spec file under `specs/` cites `src/` (**Citation**).

Apply the **Writing gate** before mutating a spec file under `specs/`. Reach `CONTEXT.md` when writing, changing, or reviewing any spec file here. Update it in the same session a term is defined, renamed, or retired. Shape is under **Glossary**. Reach `TODO.md` when a needed spec is missing or a parked item may close. Shape is under **TODO**.

## Tree

```
AGENTS.md                        # this file: law of the tree
PERSPECTIVES.md                  # catalog: the perspectives of this product
TODO.md                          # parked work this repository cannot close yet
CONTEXT.md                       # glossary: what domain terms are
<perspective>/
  AGENTS.md                      # law of the perspective: feature boundary, scope, vocabulary
  spec.md                        # iff the perspective owns a global rule
  spec.md.metadata.json          # beside every spec.md, written by the metadata tool
  <feature>/                     # ASCII kebab-case, no numeric prefix
    spec.md
    spec.md.metadata.json
```

A feature directory holds **only** `spec.md` and `spec.md.metadata.json`. There is no intermediate directory between the perspective and the feature. Omit a feature that has no rules. Do not leave empty files.

The root of this tree owns no rule. Every rule classifies into one perspective by that perspective's scope test, so the highest owner a rule can reach is a perspective's `spec.md` — never this file.

**Spec file** is the skeleton of every `spec.md`. Reach it when creating or filling one.

`<feature>` names a feature of this perspective. The boundary criterion is defined in that perspective's `AGENTS.md`. Reach it when creating or justifying a new feature.

## Perspective AGENTS.md

The `AGENTS.md` of a perspective is law only: feature boundary, scope test, vocabulary. It asserts nothing about the product, so it is not content of any node (**Content** in the root `AGENTS.md`).

A rule whose owner is the perspective, not one feature — one obligation that holds across every feature of the perspective — lives in the sibling `spec.md`, to the same **Spec file** skeleton with the perspective name as `#`. It is a rule like any other: same **Rule format**, same **Title** rules, same normative body, served and implemented like any other.

Create `spec.md` when its first rule exists.

## Perspectives

`PERSPECTIVES.md` is the catalog of the perspectives of this product: one bullet per perspective — the name in bold, its directory in parentheses, what it holds and what it leaves out. Reach it to know which perspectives exist before scoping a rule. It mirrors `perspectives` of `.sdd/.sdd-config.json`: the two move in the same session. Not a spec file, no node.

`/sdd-spec` adds, renames, or removes a perspective, in one session: the entry in `perspectives` of `.sdd/.sdd-config.json` (`from` names the shipped kind it derives from — business, user, operator — and is absent for a kind of this product alone), the directory, its `AGENTS.md` written by analogy to the closest shipped kind (same headings: feature boundary, scope table, vocabulary table), and its bullet here. A rename moves the directory with its specs; a removal deletes them; the technical and code nodes that listed a spec under it show as dangling in `pending`, for `/sdd-design` and `/sdd-implement`. The law of an existing perspective's `AGENTS.md` is not edited by a spec write.

## Glossary

`CONTEXT.md` is the canonical vocabulary of this tree. Terms define **what things are**. Behavior and rules live in spec files. No node.

Specs use the canonical term; they do not redefine it. When a term is resolved, write it here immediately — do not batch.

### Shape

```markdown
# Glossary

Canonical vocabulary of this tree. Terms here define what things are — behavior and rules live in spec files.

## Access

**Member**:
Person with an account in the workspace.
_Avoid_: User (generic), account (generic)

**Role**:
One of the fixed permission sets bound to a member — Owner, Editor, or Viewer.
_Avoid_: profile, permission group
```

- One canonical term; rivals only under `_Avoid_:`.
- One or two sentences. What it is, not what it does.
- Domain terms of this product only. General programming concepts stay out.
- Group under `##` when clusters emerge; a flat list is fine for one subject.
- English, matching spec language.
- The first term creates the first entry. Omit empty clusters.

## TODO

`TODO.md` is the stateful working set of work this repository still owes. Park what the current session cannot close — a needed feature or rule that does not exist, and this request does not create. Interview closes decisions the user can make now. Write only the rules this session can close.

Four writers, one heading each. `/sdd-spec` parks under `## Missing spec` a needed feature or rule that does not exist and the request does not create. `/sdd-design` parks under `## Spec gap` a product rule it cannot design against — the body is open (a hole, an ambiguity, an undefined real path) or the rule does not exist yet; the spec write that closes it deletes the item. `/sdd-implement` parks under `## Design gap` a rule it cannot implement because the technical spec is open — an unnamed mechanism, an open alternative, a hole `/sdd-design` has to close or escalate; the design write that closes it deletes the item. `/sdd-audit` parks under `## Audit gap` a code node no technical spec governs — its judge found no spec under `src/specs/` whose rules the code obeys; the design write that closes it deletes the item. `pending` reads the three gap headings: a design gap parks its rule out of the next batch, a spec gap is named on the stale technical spec it holds, and an audit gap marks its code node as parked under `# Audit`, for `/sdd-design`.

Read it at the start of a spec write, and again after writing a feature or rule that may unblock an item. Park in the same session as the block. Delete the item in the same session the parked work is done. A finished item leaves the file; this is remaining work, not a changelog.

Not a spec file, no node. Chrome stays. The first parked item creates the first entry. Omit empty clusters.

### Shape

```markdown
# TODO

Work this repository still owes, parked because the current session cannot close it.

## Missing spec

- **Invite expiry** (`business/invite`) waits on **Member roles** (`business/access`).

## Spec gap

- **Invite expiry** (`business/invite`) leaves the grace period undefined.

## Design gap

- **Invite token storage** (`src/specs/invite`) names no persistence package.

## Audit gap

- **Rate limiting** (`src/api`) code obeys a limit no technical spec states.
```

- One line per item, no checkbox: the work, the intended owner, and the blocker. The owner is `perspective/feature`, `perspective` for a global rule, `src/specs/<name>` for a technical spec, or, for an audit gap, a code node, a linked file — which parks the directory node that links it — or a directory, which parks every code node at or under it. A spec or design gap names the rule title verbatim in bold; an audit gap names what the missing technical spec would decide. Then the owner in backticks between parentheses, then the blocker; `pending` parses that shape.
- English, matching spec language.
- Group under `##` when clusters emerge; a flat list is fine for one subject.

## Spec file

Every `spec.md` has this skeleton. What it holds is in **Scope**. Titles, wording, and citation direction stay in **Rule format** and **Citation**.

1. `#` the feature name — or the perspective name, for its global rules — not the kebab-case directory.
2. One purpose sentence. File chrome: not a rule.
3. Optional out-of-scope prose with a bold lead-in and no `##` heading. Also chrome.
4. Then only `##` subjects and `###` rules.

The file is self-contained: it carries the rules it owns and nothing copied.

```markdown
# Access

Three immutable roles and the account life cycle of this feature.

**Out of scope:** authentication itself.

## Roles

### Fixed catalog of three roles

The system **must** support exactly three roles: **Owner**, **Editor** and **Viewer**. No other role **must** exist.
```

## Scope

A perspective spec states what the product does and for whom: who may, when it expires, what the screen shows, what the click does, who may ship. What each perspective holds and leaves out is in that perspective's `AGENTS.md`. Reach it when scoping an item, writing a spec file, or reviewing a heading.

A passage about a library, placement, contract, persistence, toolchain, or command is a technical item, whatever the request called it. It is not written here: it is named for `/sdd-design`, and the product rule it will serve is written first, so the technical rule can cite it by title.

## Citation

Documentation is self-contained: a rule body is never copied into a second file, and no build step assembles it. A file states the rules it owns.

**Cite by title.** Naming the human-readable title is the citation. Add a markdown link (`[Fixed catalog of three roles](../access/spec.md#fixed-catalog-of-three-roles)`) when the reader has to reach the owner. The link complements the title; it never replaces the nucleus of the citing rule.

A perspective spec never cites `src/`. A technical spec cites a perspective rule by title, with a link into `specs/`, only to name **which** rule it serves — it does not restate, summarize, or paraphrase that rule.

## Feature vs global

A feature directory holds the rules that belong to **that** boundary alone.

A rule whose body applies to the whole perspective or to more than one of its features **belongs** to the perspective's `spec.md`. Its stated scope proves multiple real consumers; existing feature specs are not required as evidence. Features cite it by title; they do not restate it. A global rule never depends on one feature's spec.

Otherwise, a rule needed by more than one feature **moves** at the **second real consumer** — not in anticipation of reuse.

Resolve the owner in this order:

1. Obligation is about how code is written, built, verified, or placed → technical item, `/sdd-design`.
2. Obligation applies to the whole perspective or multiple feature boundaries → global rule of that perspective.
3. One feature boundary owns the obligation independently → feature rule.

A technical subject area is not a feature boundary. Persistence, infrastructure, localization mechanisms, code architecture, testing, and tooling belong to `src/specs/`. Do not create a feature directory to group rules by technical concern.

Copy vs coincidence: if the correct change to A requires the same change in B, it is copy → one owner in the perspective's `spec.md`. If A and B may diverge without an inconsistent product, each stays in its feature.

## Rule format

A **rule** asserts. It has an owner, a title, and a normative body. It obligates (*the invite expires in 7 days*) or constitutes (*the catalog is this closed set of roles*).

Text that only **restates** an assertion made elsewhere is not a rule: no heading, no parallel upkeep. If two artifacts assert the same thing, the assertion stays where it is most precise; the other becomes a citation by title.

### Shape

- Identifier = **human-readable title** (English), anchored on a **block**, not a line.
- Body = the whole block after the title (paragraph, table, or diagram). Conditions, exceptions, and side effects belong in that body.
- Stated **once**, at the owner.
- Self-contained: the obligation is understandable from this rule alone. Cross-references complement; they do not replace the nucleus.
- Example only **below** an already-asserted obligation, labeled as illustration.

The filled file is under **Spec file**.

### Title

- Unique **per file** (case-insensitive).
- Short noun phrase from the theme: `Fixed catalog of three roles`, `Confirmation before deletion`.
- First word capitalized; rest lowercase except proper names and quoted UI labels.
- Spaces between words — not kebab-case.
- No sequential numbers, no feature prefixes (`RF-001`, `RT-001`).
- Distinct from neighboring titles without reading the body. `General`, `Rules`, `Patterns` are not titles.

### Sections

`##` groups rules of one **subject**. They are not split by audience into `## Business` / `## Interface` / `## Screens`.

A section is a coherent block of one subject — not a single orphan rule, and not unrelated topics glued together.

### Vocabulary

Write and leave-out are defined in that perspective's `AGENTS.md`. Reach it when drafting a rule body or a title.

### Anchors

The `#…` fragment is the slug the editor and GitHub generate from the target title:

1. Lowercase.
2. Strip everything that is not a letter, number, `_`, space, or `-`. Accented letters **stay**.
3. Each space becomes `-`.

Do not strip diacritics: `Invalid credentials with generic message` → `#invalid-credentials-with-generic-message`. Punctuation removed between words leaves consecutive hyphens; that is correct (`Multi-tab session — product rules` → `#multi-tab-session--product-rules`).

Renaming a title updates every mention and `#anchor`. The technical specs that serve the rule show the rename in `pending`, told from a body change by the rule hash. A link in a spec that names no file, or no heading of the markdown file it names, is drift (**Metadata** in the root `AGENTS.md`): a title removed or renamed leaves it in every spec that cites the title until that spec's writer fixes it — `/sdd-spec` under `specs/`, `/sdd-design` under `src/specs/`.

## Metadata

Every `spec.md` here is a node of **Metadata** in the root `AGENTS.md`: a sibling `spec.md.metadata.json` carries its hash and the hash of each rule. Nothing here records what consumes it; the technical specs and code that do record this spec as a source, and `pending` names them when it changes.

A spec write ends with `go run -C .agents/tools ./metadata --write <dir>` for every directory written; the pre-commit hook refreshes what a session forgot. The file is never edited by hand.

## Postulates

Violation is a grave failure. Volume, age, or “already reviewed” does not excuse it.

**P1 Single source of truth.** One owner. Self-contained. Others cite by title — they do not restate. Verify: two wordings of the same obligation, or “as described in…” plus paraphrase → merge into the owner. A rule restated in a feature and in the perspective's `spec.md` keeps the owner that has the rule.

**P2 Scope.** The rule is a product rule by the scope test of its perspective's `AGENTS.md`; a technical item is not written here. Reach the scope test when scoping, writing, or reviewing a rule.

**P3 Examples are not rules.** A title plus only a scenario, illustrative list, or “for example…” with no obligation is a failure. Extract the norm; keep the example below it. A table is a rule only when it obligates.

**P4 Broken windows.** A filled file is not a conforming file. On the path of the request, inherited violations are repaired before new rules pile on.

**P5 One owner for a shared obligation.** Same obligation, same domain concept → one owner. Promotion target is the perspective's `spec.md` at the second **real** consumer. Coincidence stays local. Promoting with one consumer violates P7. A scope error is P2, not a P5 win.

**P6 Integrity and value.** The rule has sense, value to the user or the system, and no holes — contradictions, omitted exceptions, or undefined real paths. If a competent reader cannot say whether a given case matches the rule, the rule is vague. Ceremonial rules are not written.

**P7 Minimum reading closure.** Closed by reading the owner file plus scopes it cites **directly**. A→B→C for the same obligation is depth 3. A large cohesive feature file is cheaper than split owners that must cite each other. The perspective's `spec.md` is always in reading range, so a global rule adds no depth.

## Writing gate

**Done when:** every added, changed, renamed, moved, or removed rule is in its owner file; every applicable postulate holds; every written directory has fresh metadata; the glossary matches the terms the session resolved; items this session unblocked are gone from `TODO.md`; new blockers are parked there; technical items are named for `/sdd-design`.

1. **Scope** the item with that perspective's `AGENTS.md`: a product rule continues; a technical item is named for `/sdd-design` and the rest continues.
2. **Resolve the owner:** an existing feature, the perspective's `spec.md` (second real consumer), or a new feature justified by the boundary criterion in that perspective's `AGENTS.md`.
3. **Consult `CONTEXT.md` and `TODO.md`**, then draft against the postulates — checklist below. Fix violations before writing.
4. **Write** the rule at its owner to the **Spec file** skeleton. Spec language is English. If a term was defined, renamed, or retired, update `CONTEXT.md` in the same session. Park new blockers and delete unblocked items in `TODO.md` in the same session.
5. **Refresh metadata:** `go run -C .agents/tools ./metadata --write <dir>` for each directory written. A removed feature deletes its directory; the technical specs that served it show as dangling in `pending`, for `/sdd-design`.
6. **Stop.**

| | Ask |
|--|------|
| P1 | Does this assertion already live elsewhere? Is it self-contained? |
| P2 | Is this a product rule by the scope test in this perspective's `AGENTS.md`? |
| P3 | Is there an obligation, or only an example? |
| P4 | Am I copying a neighboring violation? |
| P5 | Copy (promote) or coincidence? Second real consumer? |
| P6 | Sense, value, no holes? Can a reader judge the rule from the text? |
| P7 | Owner plus direct cites — no transitive chain? |
| Glossary | Does the spec use the canonical name? Does every new or shifted term have an entry? |
| TODO | Did this session park what it cannot close? Did unblocked items leave the file? |

Permissions in each perspective: create, edit, and delete `<feature>/spec.md` with its directory, and the perspective's `spec.md`. A perspective is created, renamed, or removed as **Perspectives** says; the law of an existing perspective's `AGENTS.md` is not edited by a spec write. `spec.md.metadata.json` is written only through the tool. Edit `CONTEXT.md` (glossary only). Edit `TODO.md` (parked work only).
