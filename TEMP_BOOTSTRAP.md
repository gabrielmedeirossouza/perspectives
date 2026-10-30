# Bootstrap

Read by an agent in a fresh clone of the SDD framework, with the person who owns the product answering. It configures the framework for one product: which perspectives exist, where specs and code live, which skills stay. It never writes a product spec, a technical spec, or code — those are `/sdd-spec`, `/sdd-design`, and `/sdd-implement`, once the project exists — and it never asks the stack: that is a technical decision `/sdd-design` makes in a spec. Perspectives and skills stay changeable afterwards (**Perspectives** of `specs/AGENTS.md`; `skills` of the config); `paths` is set here once. It ends by deleting this file.

## Preconditions

Stop and say why when any of these fails:

- `git` and Go 1.22 or newer are installed (`go version`).
- The working directory is the root of the clone: `.sdd/templates/` exists and holds `AGENTS.md`, `specs/`, `src/`, `.agents/`, `.claude/`, `.githooks/`, `.gitignore`, and `.sdd/.sdd-config.json`.
- `git status --porcelain` is empty.
- Note whether `.git` is a directory (a clone) or a file (a git worktree, such as a framework sandbox): a worktree never resets its history.

## Interview

One question at a time, with a recommendation; the person decides. Record every answer; answers 1–5 become `.sdd/.sdd-config.json`.

1. **Product.** Its name and one sentence: what it is and for whom. → `product`.
2. **Perspectives.** Show the three the framework ships with their one-line scope from `.sdd/templates/specs/PERSPECTIVES.md`: business (product policy), user (what the user sees and does), operator (how the system is shipped and run). For each: keep, rename (a game calls its user `player`), or drop; then ask for actors of this product these do not cover. Each answer is one entry: `{"name": "<kebab-case>", "from": "<business|user|operator>"}` for a kept or renamed one, `{"name": "<kebab-case>"}` for a new kind. For a new kind also ask what a feature of it is, what its `spec.md` holds and leaves out, and what its prose writes and leaves out: you will write its `AGENTS.md` by analogy. Recommend keeping all three, renamed where the product's language differs. → `perspectives`.
3. **Paths.** Where product specs, code, and technical specs live: recommend `specs`, `src`, and `src/specs`; a monorepo may want `apps` for code and `apps/specs` for technical specs. The three never overlap, and technical specs sit under the code root or beside it. → `paths`.
4. **Skills.** List the skills under `.sdd/templates/.agents/skills/` with the first sentence of each description; recommend all: the law and the other skills name every skill, a dropped one keeps those mentions, and without `sdd-update-framework` the project has no update path. → `skills`, a list of names (`["sdd-spec", …]`), omitted to keep all.
5. **Remote.** The URL `/sdd-update-framework` reads new versions from: recommend `git remote get-url origin` of this clone when it is a URL; a local path works only on this machine. → `remote`.
6. **History.** Start the project's history fresh (recommended: the framework's history is not the product's) or keep it. Not asked in a worktree.

## Materialize

Run exactly these, from the clone root, after the interview:

1. Write `.sdd/.sdd-config.json` — `version` stays 0; the tool records the real one:

   ```json
   {
     "version": 0,
     "remote": "<answer 5>",
     "product": "<answer 1>",
     "paths": { "specs": "<answer 3>", "src": "<answer 3>", "technical": "<answer 3>" },
     "perspectives": [ <answer 2> ],
     "skills": [ <answer 4, or omit the field> ]
   }
   ```

2. Remove the framework's own root, and nothing else: `rm -rf AGENTS.md .gitignore .githooks .agents .claude`.
3. Materialize: `go run -C .sdd/templates/.agents/tools ./update --apply --root "$(pwd)" --from .` — `--root` is the project root, because `go run -C` moves the working directory under `.sdd/templates/`, which holds a config of its own, and `--from` is relative to `--root`. Read its plan: every template file is an `add` at its place in this layout, a `skip` names a dropped perspective or skill, and each `by hand` line is yours in **Adapt**.
4. Remove what only the framework repository needs: `rm -rf .sdd/templates .sdd/sandbox`. This file stays until **Finish**.
5. Enable the hook: `git config core.hooksPath .githooks`.

## Adapt

Prose the tool cannot write. English, the shape of the shipped file, nothing about the product's rules — those are specs.

1. `<paths.specs>/PERSPECTIVES.md`: the purpose sentence names the product; one bullet per perspective of the config, in its order — the name in bold, the directory in parentheses, what it holds and leaves out. A dropped kind loses its bullet.
2. The `AGENTS.md` of every renamed perspective (`<paths.specs>/<name>/AGENTS.md`, materialized from its `from`): the title and every mention of the old actor become the new one; the feature boundary, the scope table, and the vocabulary keep their shape, reworded for the actor. The `AGENTS.md` of every new kind (a `by hand` line): by analogy to the closest shipped one, from the answers of the interview.
3. Skill examples that name a renamed or dropped perspective, such as `/sdd-spec user/home:…` and `/sdd-spec operator/release:…` in `.agents/skills/sdd-spec/SKILL.md`: rename them to a perspective of this product.

## Verify

Every line holds before finishing:

- `go vet ./... && go test ./...` in `.agents/tools` are green.
- `go run -C .agents/tools ./metadata` prints `no node under …`; `go run -C .agents/tools ./pending` prints `No pending work.`
- `find .claude/skills -type l ! -exec test -e {} \; -print` prints nothing; `test -x .githooks/pre-commit` holds.
- `test ! -e .sdd/templates && test ! -e .sdd/sandbox` holds; `ls .agents/skills` lists exactly the skills kept.
- `.sdd/.sdd-config.json` holds a version above 0, the remote, the paths, and the perspectives.

## Finish

1. History: when the answer was fresh and `.git` is a directory, `rm -rf .git && git init -b main`, then `git config core.hooksPath .githooks` again — the setting lives in `.git`.
2. Delete this file: `rm TEMP_BOOTSTRAP.md`.
3. Commit: `git add -A && git commit -m "chore: bootstrap sdd framework v<version>"` — the one commit `git add -A` is right for; no spec exists yet, so the hook refreshes nothing.
4. Report: the layout, the perspectives with where each came from, the skills kept, the version, and the next steps — `/sdd-brainstorming` for the first idea, `/sdd-spec` for the first rule, `/sdd-update-framework` when the framework moves.

**Done when:** every line of **Verify** holds, this file is gone, and the project has its first commit.
