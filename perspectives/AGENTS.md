# Perspectives

Perspectives hold the system specifications. They are the single source of truth.

A perspective is documentation written from the viewpoint of one actor of the system: the business, the user, the operator, the developer.

What lives in which perspective is named here. Each nested `AGENTS.md` fills the slots this file points at for that viewpoint.

- **Business** (`perspectives/business/`): product policy and the mechanism of each business feature. Product policy lives here, not on the user surface.
- **User** (`perspectives/user/`): what the user sees and does — screens, flows, copy — and the mechanism of each user-facing feature. Screen behavior lives here, not in business.
- **Operator** (`perspectives/operator/`): how the system is shipped and run — environments, deployment, and observability — and the mechanism of each operational feature. Nothing the product does for its users lives here.
- **Development** (`perspectives/development/`): the mechanism that constrains how code is written, organized, built, and verified, regardless of which rule that code materializes — toolchain, runtime, test runner, code layering. It owns global rules only: no features, no policy layer.

Apply the **Writing gate** before mutating a spec file or a tracking heading or flag under `perspectives/`. Path lists on tracking entries are defined in this file; the implementation postulates and the Implementation gate are in `src/AGENTS.md`. Reach `CONTEXT.md` when writing, changing, or reviewing any spec file under `perspectives/`. Update it in the same session a term is defined, renamed, or retired. Shape is under **Glossary**. Reach `TODO.md` when a needed spec is missing or a parked item may close. Shape is under **TODO**.

## Tree

```
AGENTS.md                        # this file: law of the tree
TODO.md                          # parked work this session cannot close
CONTEXT.md                       # glossary: what domain terms are
<perspective>/
  AGENTS.md                      # law of the perspective + its global rules
  policy-tracking.md             # iff AGENTS.md owns a global policy rule
  mechanism-tracking.md          # iff AGENTS.md owns a global mechanism rule
  <feature>/                     # ASCII kebab-case, no numeric prefix
    policy.md                    # iff this feature has policy rules
    mechanism.md                 # iff this feature has mechanism rules
    policy-tracking.md           # iff policy.md has a rule
    mechanism-tracking.md        # iff mechanism.md has a rule
```

A feature directory uses **only** those four names. There is no intermediate directory between the perspective and the feature. Omit a spec file that has no rules; omit its tracking file too. Do not leave empty files.

`development/` is the exception: it holds `AGENTS.md` and `mechanism-tracking.md` only — no `<feature>/`, no `policy-tracking.md`.

The root of this tree owns no rule. Every rule classifies into one perspective by that perspective's layer test, so the highest owner a rule can reach is a perspective's `AGENTS.md` — never this file. No tracking file sits beside it.

**Spec file** is the skeleton of `policy.md` and `mechanism.md`. Reach it when creating or filling either.

`<feature>` names a feature of this perspective. The boundary criterion is defined in that perspective's `AGENTS.md`. Reach it when creating or justifying a new feature.

## Perspective AGENTS.md

The `AGENTS.md` of a perspective carries three natures. They do not mix inside one heading.

| Nature | Heading | Tracking |
|--------|---------|----------|
| Law of the folder — feature boundary, layer test, vocabulary | as that file defines | none |
| Global rule — one obligation that holds across every feature of the perspective | `## Global policy rules`, `## Global mechanism rules` | sibling `policy-tracking.md` or `mechanism-tracking.md`, at the perspective root |
| Pattern — the system uses a mechanism, and each feature **must** decide whether it adopts it | `## Adopted patterns` | none |

A global rule is a rule: same **Rule format**, same **Title** rules, same normative body, tracked and implemented like any other. It lives here because its owner is the perspective, not one feature.

A pattern obligates the person writing a spec, not the product. It names the mechanism, says where the concrete rules live once adopted, and requires each feature to state its decision. It asserts nothing about runtime behavior, so it gets no tracking entry.

Create a slot when its first entry exists. Omit empty headings.

## Glossary

`CONTEXT.md` is the canonical vocabulary of this tree. Terms define **what things are**. Behavior and rules live in spec files. No tracking entry.

Specs use the canonical term; they do not redefine it. When a term is resolved, write it here immediately — do not batch.

### Shape

```markdown
# Perspectives

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

`TODO.md` is the stateful working set of work this tree still owes. Park what the current session cannot close — a needed feature or rule that does not exist, and this request does not create. Interview closes decisions the user can make now. Tracking `Dependency:` flags a rule already written. Write only the rules this session can close.

`/implement` is the second writer, under `## Spec gap` only: it parks a written rule it cannot implement because the body is open — a hole, an ambiguity, an unnamed mechanism, an open alternative. The spec write that closes the gap deletes the item; until then `/implement continue` skips the rule.

Read it at the start of a spec write, and again after writing a feature or rule that may unblock an item. Park in the same session as the block. Delete the item in the same session the parked work is done. A finished item leaves the file; this is remaining work, not a changelog.

No tracking entry. Not a spec file. Chrome stays. The first parked item creates the first entry. Omit empty clusters.

### Shape

```markdown
# TODO

Work this tree still owes, parked because the current session cannot close it.

## Missing spec

- [ ] **Invite expiry** (`business/invite`, policy) waits on **Member roles** (`business/access`).

## Spec gap

- [ ] **Invite token storage** (`business/invite`, mechanism) names no persistence package.
```

- One line per item: the work, the intended owner (perspective/feature, layer), and the blocker. A spec gap names the rule title verbatim and its owner — `perspective/feature`, or `perspective` for a global rule.
- English, matching spec language.
- Group under `##` when clusters emerge; a flat list is fine for one subject.

## Spec file

`policy.md` and `mechanism.md` share this skeleton. What each file holds is in **Layers**. Titles, wording, and citation direction stay in **Rule format** and **Citation**.

1. `#` the feature name — not the kebab-case directory.
2. One purpose sentence. File chrome: not a rule, not a tracking entry.
3. Optional out-of-scope prose with a bold lead-in and no `##` heading. Also chrome.
4. Then only `##` subjects and `###` rules.

The file is self-contained: it carries the rules it owns and nothing copied. The mechanism title is prose; the package identifier lives in the body; a policy title in a mechanism body is a mention, not a restatement.

`policy.md`:

```markdown
# Access

Three immutable roles and the account life cycle of this feature.

**Out of scope:** authentication itself.

## Roles

### Fixed catalog of three roles

The system **must** support exactly three roles: **Owner**, **Editor** and **Viewer**. No other role **must** exist.
```

Same feature, `mechanism.md`:

```markdown
# Access

How this feature is built: placement, persistence, and the HTTP client.

## HTTP client

### Single HTTP client

Every outbound call **must** use one `acme-http` client. Serves **Fixed catalog of three roles**.
```

## Layers

Each feature uses `policy.md` and `mechanism.md`. The same two layers split the global rules of a perspective. What each layer holds and does not hold is defined in that perspective's `AGENTS.md`. Reach it when classifying a rule, writing a spec file, or reviewing a heading against the wrong layer.

## Citation

Documentation is self-contained: a rule body is never copied into a second file, and no build step assembles it. A file states the rules it owns.

**Cite by title.** Naming the human-readable title is the citation. Add a markdown link (`[Fixed catalog of three roles](../access/policy.md#fixed-catalog-of-three-roles)`) when the reader has to reach the owner. The link complements the title; it never replaces the nucleus of the citing rule.

Policy never cites `mechanism.md`. Mechanism cites a policy rule by title only to name **which** policy rule the mechanism rule serves — it does not restate, summarize, or paraphrase that rule.

## Feature vs global

A feature directory holds the rules that belong to **that** boundary alone.

A rule whose body applies to the whole perspective or to more than one of its features **belongs** to the `AGENTS.md` of the perspective. Its stated scope proves multiple real consumers; existing feature specs are not required as evidence. It becomes a global rule under **Perspective AGENTS.md**, and its tracking entry lives at the perspective root. Features cite it by title; they do not restate it. A global rule never depends on one feature's spec. A mechanism rule that constrains the form of the code regardless of which rule that code materializes belongs to `development/`; the boundary test is in its `AGENTS.md`.

Otherwise, a rule needed by more than one feature **moves** at the **second real consumer** — not in anticipation of reuse.

Resolve the owner in this order:

1. Obligation constrains the form of the code — how it is written, organized, built, or verified — regardless of which rule that code materializes → global mechanism rule of `development/`.
2. Obligation applies to the whole perspective or multiple feature boundaries → global rule of that perspective.
3. One feature boundary owns the obligation independently → feature rule.
4. Every feature must decide independently whether to use a mechanism → adopted pattern.

A technical subject area is not a feature boundary. Persistence conventions, cross-cutting infrastructure, and localization belong to the global rules of the perspective whose features they constrain. Code architecture, testing, and repository-wide tooling belong to `development/`. Do not create a feature directory only to group rules by technical concern.

Copy vs coincidence: if the correct change to A requires the same change in B, it is copy → one owner in the perspective's `AGENTS.md`. If A and B may diverge without an inconsistent product, each stays in its feature.

## Rule format

A **rule** asserts. It has an owner, a title, and a normative body. It obligates (*the invite expires in 7 days*) or constitutes (*the catalog is this closed set of roles*).

Text that only **restates** an assertion made elsewhere is not a rule: no heading, no tracking entry, no parallel upkeep. If two artifacts assert the same thing, the assertion stays where it is most precise; the other becomes a citation by title.

### Shape

- Identifier = **human-readable title** (English), anchored on a **block**, not a line.
- Body = the whole block after the title (paragraph, table, or diagram). Conditions, exceptions, and side effects belong in that body.
- Stated **once**, at the owner.
- Self-contained: the obligation is understandable from this rule alone. Cross-references complement; they do not replace the nucleus.
- Example only **below** an already-asserted obligation, labeled as illustration.

The filled file is under **Spec file**.

### Title

- Unique **per file** (case-insensitive).
- Short noun phrase from the theme: `Fixed catalog of three roles`, `Single HTTP client for [...]`.
- First word capitalized; rest lowercase except proper names and quoted UI labels.
- Spaces between words — not kebab-case.
- No sequential numbers, no feature prefixes (`RF-001`, `RT-001`).
- Distinct from neighboring titles without reading the body. `General`, `Rules`, `Patterns` are not titles.

### Sections

`##` groups rules of one **subject**. The two layers of this perspective stay in their own files; they are not split into `## Business` / `## Interface` / `## Screens`.

A section is a coherent block of one subject — not a single orphan rule, and not unrelated topics glued together.

### Vocabulary

Write and leave-out per layer are defined in that perspective's `AGENTS.md`. Reach it when drafting a rule body or a title.

Mechanism titles stay prose (`Single HTTP client`). A mechanism rule that names a library **states the package** (or system tool) in the body (`acme-http`).

Mechanism states **one** decision with `must`, `must not`, `never`, `only`, `prohibited`. Facultative modals (`may`, `can be`, `ideally`, `prefer`) are not a rule.

### Anchors

The `#…` fragment is the slug the editor and GitHub generate from the target title:

1. Lowercase.
2. Strip everything that is not a letter, number, `_`, space, or `-`. Accented letters **stay**.
3. Each space becomes `-`.

Do not strip diacritics: `Invalid credentials with generic message` → `#invalid-credentials-with-generic-message`. Punctuation removed between words leaves consecutive hyphens; that is correct (`Multi-tab session — product rules` → `#multi-tab-session--product-rules`).

Renaming a title updates every mention and `#anchor`, plus the tracking entry.

## Tracking

Tracking is the link from each rule to its **materialization**. `[x]` means the rule is fully implemented: its code, tests, and applicable validation are complete. `[ ]` means the rule is not fully implemented. The `/implement` queue is the skill argument; under `continue` it is the next batch of `[ ]` entries.

Origin is the **file**, not a prefix. Tracking sits beside the owner. A global rule's entry lives at the perspective root, beside its `AGENTS.md`; a feature that cites the rule does not duplicate the entry.

| Tracking file | Rule owner |
|---------------|------------|
| `<feature>/policy-tracking.md` | sibling `policy.md` |
| `<feature>/mechanism-tracking.md` | sibling `mechanism.md` |
| `<perspective>/policy-tracking.md` | `## Global policy rules` of the sibling `AGENTS.md` |
| `<perspective>/mechanism-tracking.md` | `## Global mechanism rules` of the sibling `AGENTS.md` |

Two writers. Spec skills own the heading and change flags. The `/implement` skill owns the completion mark and path list.

| Writer | Owns |
|--------|------|
| `/spec` | Entry heading, `[ ]` after a new or changed rule, `Changed:`, `Removed:`, `Dependency:` |
| `/implement` | `[x]` after full implementation and paths under an existing heading; deleting the entry after `Removed:` is cleaned |

Spec never writes paths. Implement never creates, renames, or deletes a heading except to delete the whole entry after cleaning `Removed:`.

Entry heading is the rule title **verbatim**. A new or changed rule is left unticked until `/implement` confirms full implementation:

```markdown
### [ ] Fixed catalog of three roles
```

When code fully materializes the rule, `/implement` marks the entry `[x]` and appends one path per line, relative to the repository root, in backticks:

```markdown
### [x] Fixed catalog of three roles
- `src/backend/src/modules/access/access.types.ts`
- `src/backend/src/modules/access/access.spec.ts`
```

`Dependency:` is allowed. Canonical flags with that exact prefix:

- `Changed:` — title or body changed (spec). Remove `[x]`; paths stay for revalidation.
- `Removed:` — spec, only when the entry already has paths. Implement deletes exclusive code, then deletes the entry.

| Operation | Tracking |
|-----------|----------|
| Add | Spec: `### [ ] <title>`. No paths. |
| Remove, no paths | Spec: delete the entry. |
| Remove, has paths | Spec: keep `[ ]`, write `Removed:`. Implement cleans exclusive paths, then deletes the entry. |
| Rename | Spec: update the title, set `[ ]`, `Changed: previous title "…".` Paths stay. |
| Change the body | Spec: set `[ ]`, write `Changed: body updated.` Paths stay for revalidation. |
| Move (feature ↔ perspective `AGENTS.md`, or between features) | Spec: remove on the source tracking + add on the destination. Paths move with the entry. |
| Write or update code | Implement: add, correct, or drop paths under the existing heading; set `[x]` only after full implementation. |

Sync the heading, completion mark, and flags in the **same session** as the spec change. Sync completion marks and paths in the **same session** as the code change.

## Postulates

Violation is a grave failure. Volume, age, or “already reviewed” does not excuse it.

**P1 Single source of truth.** One owner. Self-contained. Others cite by title — they do not restate. Verify: two wordings of the same obligation, or “as described in…” plus paraphrase → merge into the owner. A contract restated in a feature and in the perspective's `AGENTS.md` keeps the owner that has the contract.

**P2 Layer.** The layer test is defined in that perspective's `AGENTS.md`. Reach it when classifying, writing, or reviewing a rule.

**P3 Examples are not rules.** A title plus only a scenario, illustrative list, or “for example…” with no obligation is a failure. Extract the norm; keep the example below it. A file tree or contract table is a rule only when it obligates.

**P4 Broken windows.** A filled file is not a conforming file. On the path of the request, inherited violations are repaired before new rules pile on.

**P5 One owner for a shared obligation.** Same obligation, same domain concept → one owner. Promotion target is the perspective's `AGENTS.md` at the second **real** consumer. Coincidence stays local. Promoting with one consumer violates P7. A layer error is P2, not a P5 win.

**P6 Integrity and value.** The rule has sense, value to the user or the system, and no holes — contradictions, omitted exceptions, or undefined real paths. If a competent reader cannot say whether a given case matches the rule, the rule is vague. Ceremonial rules are not written.

**P7 Minimum reading closure.** Closed by reading the owner file plus scopes it cites **directly**. A→B→C for the same obligation is depth 3. A large cohesive feature file is cheaper than split owners that must cite each other. The perspective's `AGENTS.md` and `development/AGENTS.md` are always in reading range, so a global rule adds no depth.

## Writing gate

**Done when:** every added, changed, renamed, moved, or removed rule has the matching tracking heading and flags from the table; path lists are untouched; every applicable postulate holds; the glossary matches the terms the session resolved; items this session unblocked are gone from `TODO.md`; new blockers are parked there.

1. **Classify** the item into the policy or the mechanism layer using that perspective's `AGENTS.md`. Reach it before choosing a file.
2. **Resolve the owner:** `development/` for a mechanism rule that passes its boundary test, an existing feature, a global rule in the perspective's `AGENTS.md` (second real consumer), or a new feature justified by the boundary criterion in that perspective's `AGENTS.md`.
3. **Consult `CONTEXT.md` and `TODO.md`**, then draft against the postulates — checklist below. Fix violations before writing.
4. **Write** the rule at its owner: a feature spec file to the **Spec file** skeleton, or a global rule under the slot in **Perspective AGENTS.md**. Spec language is English. Sync tracking **headings and flags** in the same session. If a term was defined, renamed, or retired, update `CONTEXT.md` in the same session. Park new blockers and delete unblocked items in `TODO.md` in the same session.
5. **Stop.**

| | Ask |
|--|------|
| P1 | Does this assertion already live elsewhere? Is it self-contained? |
| P2 | Does this match the layer test in this perspective's `AGENTS.md`? |
| P3 | Is there an obligation, or only an example? |
| P4 | Am I copying a neighboring violation? |
| P5 | Copy (promote) or coincidence? Second real consumer? |
| P6 | Sense, value, no holes? Can a reader judge the rule from the text? |
| P7 | Owner plus direct cites — no transitive chain? |
| Glossary | Does the spec use the canonical name? Does every new or shifted term have an entry? |
| TODO | Did this session park what it cannot close? Did unblocked items leave the file? |

Permissions in each perspective: create, edit, and delete the four names under `<feature>/`, and the global-rule and pattern slots of that perspective's `AGENTS.md`. `development/` has no `<feature>/`; its only writable slots are the global mechanism rules and patterns of its `AGENTS.md`. The law of the folder in `AGENTS.md` — feature boundary, layer test, vocabulary — is not edited by a spec write. Sync tracking **headings and flags**. Path lists belong to implement. Edit `CONTEXT.md` (glossary only). Edit `TODO.md` (parked work only).
