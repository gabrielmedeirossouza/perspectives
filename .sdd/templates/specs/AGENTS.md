# Specs

Product specifications state what the system does and for whom. `PERSPECTIVES.md` lists the actors; choose by each candidate's **Scope** in its `AGENTS.md`, which also holds its feature boundary and vocabulary.

## Tree

```text
AGENTS.md
PERSPECTIVES.md                  # actors and their directories
CONTEXT.md                       # domain glossary
TODO.md                          # parked work
<perspective>/
  AGENTS.md                     # stable scope, boundary, vocabulary
  spec.md                       # optional global rules
  spec.md.metadata.json
  <feature>/                    # ASCII kebab-case, no numeric prefix
    spec.md
    spec.md.metadata.json
```

A feature holds only its spec and metadata, directly under the perspective. Omit features without rules and empty spec files. This root owns no product rule.

## Perspectives

`PERSPECTIVES.md` has one bullet per actor: bold name and directory in parentheses. It mirrors `perspectives` in `.sdd/.sdd-config.json`; neither the catalog nor a perspective's law is a spec node.

`/sdd-spec` adds, renames, or removes a perspective by changing its config entry, directory, and catalog bullet in one session. `from` names its shipped kind (business, user, operator); omit it for a product-only kind. Create its `AGENTS.md` by analogy to the closest kind, retaining Feature, Scope, and Vocabulary. Existing perspective law is not edited during a spec write. Renames move the specs; removals delete them. Their consumers then appear as dangling.

## Glossary

`CONTEXT.md` defines what domain terms are; specs use those terms without redefining them. Update a definition immediately when a term is resolved, renamed, or retired.

Use English, one or two sentences per term, and rivals only in `_Avoid_:`. Exclude general programming concepts. Group under `##` when useful; omit empty groups.

Illustration:

```markdown
# Glossary

## Access

**Member**:
Person with an account in the workspace.
_Avoid_: User, account
```

## TODO

`TODO.md` holds only work the current request cannot close. Resolve decisions available through interview; park remaining blockers in the same session. Delete resolved items; keep the file even when empty.

| Heading | Writer parks | Removed by |
|---------|--------------|------------|
| `## Missing spec` | `/sdd-spec`: a needed feature or rule outside this request | The spec write that supplies it |
| `## Spec gap` | `/sdd-design`: absent, ambiguous, or incomplete product behavior | The spec write that closes it |
| `## Design gap` | `/sdd-implement`: an open technical decision | The design write that closes it |
| `## Audit gap` | `/sdd-audit`: code with no governing technical spec | The design write or audit verdict that closes it |

### Shape

One English line per item, no checkbox: ``- **<title>** (`<owner>`) <blocker>``. `pending` parses the three gap headings in this shape; keep the headings exact and omit empty ones.

```markdown
- **Invite expiry** (`business/invite`) leaves the grace period undefined.
```

The title is the verbatim rule title for Spec/Design gaps; for Audit gaps, what the missing spec would decide. The owner is `perspective/feature`, `perspective` for global rules, or `src/specs/<name>`; an Audit gap names a code node, linked file, or directory, with the reach **Audit** in the root `AGENTS.md` defines.

## Scope

The perspective's scope test decides product ownership. What **Technical spec** in `src/AGENTS.md` lists belongs to `src/specs/`, regardless of the request's wording. Name those items for `/sdd-design`, with their product rules written first.

## Spec file

Specs are written in English, using this skeleton:

```markdown
# Access

Three immutable roles and the account life cycle.

**Out of scope:** authentication.

## Roles

### Fixed catalog of three roles

The system **must** support exactly these roles: Owner, Editor, Viewer.
```

The `#` names the feature, perspective (global rules), or technical subject; then one purpose sentence and optional bold-led out-of-scope prose, both chrome. Then use only `##` subjects and `###` rules, except the technical **Adopted patterns** defined in `src/AGENTS.md`. A subject groups related rules, not audiences, unrelated topics, or an isolated rule.

## Rule format

A rule has a human-readable title and a normative body asserting an obligation or a closed definition. The body is the entire block after the title, including conditions, exceptions, and effects; it may be prose, a table, or a diagram.

### Title

Use a short English noun phrase in sentence case, unique per file ignoring case; proper names and quoted UI labels keep their case. Use spaces, no sequential IDs or feature prefixes. A title must distinguish the rule without reading its body; `General`, `Rules`, and `Patterns` do not.

### Citation

Name the owner's rule title; add a markdown link when the reader needs to reach it. A perspective spec never cites `src/`. A technical spec cites a product rule by title and link only to identify what it serves; global technical specs cite no product rule.

### Anchors

Lowercase the title; remove characters other than letters, numbers, underscores, spaces, and hyphens; replace each space with a hyphen. Preserve diacritics and consecutive hyphens.

When renaming, moving, or removing a title/file, fix every affected mention and anchor within the writer's scope. Revisit rules citing removed obligations. The other scope's writer receives its broken links through `pending`; **Metadata** in the root `AGENTS.md` defines their effect.

## Postulates

All apply to spec writes and reviews; volume, age, or prior review never excuses a violation:

- **P1 Single source of truth.** Each obligation has one owner and a self-contained body. Other files cite its title without copying, summarizing, or paraphrasing it; when two files assert it, it stays where it is most precise. No build step assembles rule bodies.
- **P2 Scope.** Classify by **Scope** and the perspective's law; technical subjects are never product features.
- **P3 Norm before example.** A scenario alone is no rule, and a table is one only when it obligates. Put illustrations below an explicit obligation and label them.
- **P4 Broken windows.** Repair inherited violations on the request's path before adding rules.
- **P5 Shared ownership.** An obligation applying across the perspective or multiple feature boundaries belongs to the perspective's `spec.md`; its explicit scope proves those consumers. Otherwise promote at the second real consumer, never speculatively. If changing A correctly requires changing B, they share an obligation; independent behavior stays local. Global rules never depend on a feature.
- **P6 Integrity and value.** State useful, decidable obligations without contradictions, missing exceptions, or undefined real paths.
- **P7 Reading closure.** An obligation is understandable from its owner and directly cited scopes. Avoid transitive chains and fragmented owners; the perspective's global spec is always in reading range.

## Writing gate

1. Consult the glossary and parked work; classify each item and choose its owner using P2/P5. Justify a new feature against its perspective's boundary.
2. Validate P1–P7, then write to **Spec file**. Update resolved terms and parked work under their contracts above.
3. Refresh each written directory under root **Writers**. Remove a deleted feature's directory; report affected technical consumers to `/sdd-design`.

Completion requires all steps. Permissions: product specs and their directories; perspective lifecycle under **Perspectives**; glossary definitions; parked work. Metadata is tool-written under the root contract.
