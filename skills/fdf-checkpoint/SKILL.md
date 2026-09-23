---
name: fdf-checkpoint
description: Use periodically and before a release — and after dependency, tooling or infrastructure work that no feature recorded, after `fdf migrate` or `fdf install`, or after a hand edit to CLAUDE.md/AGENTS.md — to audit an FDF project's Context documents, SPEC.md and agent instruction files against the code and against each other, and propose approved fixes for anything stale, repeated or contradictory. Not for unfilled stubs (fdf-init).
---

# FDF Checkpoint

Audit the documents every agent reads before it works — the five Context
documents, the vendored `SPEC.md`, and the agent instruction files
(`CLAUDE.md`, `AGENTS.md`) — against the code and against each other, and
propose the edits that make them true again.

New to FDF? Run `fdf spec` for the format rules and `fdf help` for the CLI.
The fdf-help skill explains how the fdf skills fit together.

The project-document review that ends fdf-execute and fdf-change catches what
one piece of work changed. It cannot catch the rest. A dependency upgrade, a
CI move, a refactor or an infrastructure migration changes no behavior, so no
feature or change records it and no review ever runs; an instruction file
edited by hand is reviewed by nobody. Stale context is worse than none —
every later feature is designed against it. This is the sweep that finds it.

It **proposes; the user decides.** Nothing is edited without explicit
approval. It edits documents, never code. It never writes into `SPEC.md` or the
managed primer — `fdf migrate` and `fdf install` rewrite them. And it touches
episodic documents — `slug.spec.md`, plans, tasks, changes and logs — only for
a lexicon fix: they record the past, and a past that disagrees with today is
not drift, except in the words `DOMAIN.md` bans.

Most of the work is judgment — which copy is the original, whether the
document or the code moved — so where you can choose the model, run it on the
most capable one (in Claude, an Opus-class model).

## When to run it

- Before cutting a release.
- After work that changed the stack, tooling or infrastructure without
  changing behavior: a dependency or runtime upgrade, a CI or build change, a
  refactor that moved modules, an infrastructure migration.
- After `fdf migrate` or `fdf install`, and after anyone edits `CLAUDE.md` or
  `AGENTS.md` by hand.
- Otherwise on a cadence the team picks — every few features, or monthly.

## One home per fact

A fact written in two places is right on the day it is written and wrong the
first time only one copy is updated. Every fact has one home; everywhere else
it appears as a link to that home, not a copy.

| Fact | Home |
|---|---|
| Languages, runtimes, frameworks, libraries, data-store engines — and their versions | `STACK.md` |
| Architecture style, module map, where new code goes, key decisions, which practices exist | `ARCHITECTURE.md` |
| How one recurring mechanism is done — its rules | `practices/<slug>.md` |
| Interface conventions per surface, and the wording a surface shows for a term | `SURFACES.md` |
| Build, test, package and deploy commands; CI; environments; targets; where services and data stores are hosted | `INFRA.md` |
| Canonical names, the words banned in their place, how each appears in code | `DOMAIN.md` |
| The FDF format rules | `SPEC.md` — vendored, never edited |
| How agents work in this repository: harness notes, pointers into the bundle | `CLAUDE.md` / `AGENTS.md` |
| Why one feature was built the way it was | that feature's `slug.spec.md` — frozen |

## Process

1. **Gate.** Run `fdf validate`. A Context document that is still a stub is
   fdf-init's job; any other failure goes through fdf-validate first — audit a
   bundle that validates. Its `warn:` lines are findings too: carry them into
   the report — routed as fdf-validate says when they carry a rule code, and
   simply listed when they do not. (A banned word is fixed in place with a
   lexicon fix, in every document that uses it — delivered features and
   finished changes included.)
2. **Set the baseline** — a commit: the one that logged the last checkpoint,
   or, with none, the oldest of the Context documents' last commits (the
   widest window misses least). Then list what changed since:

   ```bash
   git log -1 --format=%H -S '**Checkpoint**' -- docs/features/LOG.md
   for d in STACK ARCHITECTURE SURFACES INFRA DOMAIN; do git log -1 --format='%ct %H' -- docs/features/$d.md; done | sort -n | head -1
   git log --name-only --format= <baseline>..HEAD | sort -u
   ```

   Manifests, lockfiles, CI and deploy config, and new top-level directories
   in that list are where drift hides. Start there — but the checks below
   still cover every claim, not only the changed files.
3. **Run the four checks below, in order** — mechanical, Context documents
   against the code, documents against each other, instruction files. Each
   is a procedure: work through all of it.
4. **Cite by quoting.** Every finding quotes the line it rests on, from a
   file you opened in this session — or, when the evidence is that something
   does not exist, the command that shows it. Never a paraphrase, and never a
   memory of how the project used to be. If you cannot quote it, you have not
   checked it.
5. **Report before editing anything.** One numbered list, grouped by
   document; a finding that spans documents is listed once, under the
   document that is the fact's home. For each finding: where (file and
   line), what is wrong (stale, repeated, contradictory, or a broken
   reference), the quoted evidence, and the exact edit you propose. The user
   may approve item by item or all at once. A proposed rewrite lists every
   line it drops and why: a line that disappears inside a rewrite has skipped
   its own finding.

   When the evidence cannot say which side is true, it is a question, not a
   finding. A finding can also be certain while its fix is not — the
   document is plainly wrong, but only the user knows the replacement — and
   then its edit waits on the question. Questions follow the findings, in the
   order you will ask them, one at a time.

   Close the report with its **coverage**: every section of every Context
   document, every practice, and every instruction file, each marked
   *current*, *unverified* (checked, but nothing in the repository can
   confirm or refute it), or with the numbers of its findings and questions
   (`#4`, `Q2`) —

   ```text
   STACK.md — Languages & runtimes #3 · Frameworks & libraries current ·
     Data stores #4 · Notable dependencies current
   CLAUDE.md — Setup #9 · Testing #10 · House rules #11, #12 ·
     Feature Document Format (managed) current
   ```

   A section you did not check cannot be marked current, so this list is
   what shows the sweep was complete — and it is what the log entry's "found
   current" rests on.
6. **Apply only what the user approves** — a document edit, a debt filed with
   `fdf debt`, a practice scaffolded with `fdf practice`, an `fdf install` or
   `fdf migrate`. A Context document you edit gets a new `timestamp`; a
   practice edit is logged in `practices/<slug>.log.md`.
   What the user declines stays as it is — do not propose it again in this
   checkpoint.
7. **Log and gate.** Once the user has decided, one entry in the root
   `LOG.md`, newest first: what changed, and what was checked and found
   current. The entry records decisions already made, so it needs no approval
   of its own. Log a clean checkpoint too — it is the next one's baseline.

   ```markdown
   ## 2026-09-23
   * **Checkpoint**: STACK.md — Node 22 (was 18); CLAUDE.md — stack list
     replaced by a pointer to STACK.md. ARCHITECTURE, SURFACES, INFRA,
     DOMAIN and SPEC.md current.
   ```

   Then `fdf validate` exit 0 — fdf-validate on failure.

## 1. Mechanical checks

- **The pin.** `fdf_version` in the root `INDEX.md`, against the version
  `fdf spec --list` marks current. An older pin is a proposal to run
  `fdf migrate` — the user's call, because a new spec version can hold feature
  work until a new Context document is filled.
- **`SPEC.md`** is the pinned version's spec, vendored, and matches it exactly
  below its frontmatter:

  ```bash
  diff <(fdf spec -v <pin>) <(awk 'f; /^---$/ && ++n==2 {f=1}' docs/features/SPEC.md | tail -n +2)
  ```

  Any output means a hand edit or a copy vendored by an older fdf. Never
  repair it by editing lines: copy any project content found in it to its
  home first, then — with the pin already current — propose `fdf migrate`,
  which rewrites the whole copy and changes nothing else.
- **Installed skills.** Each installed fdf skill has a `.fdf-version` file
  beside its `SKILL.md` — in a project install under `.claude/skills/`,
  `.codex/skills/` or `.opencode/skills/` — reading
  `<version> root=<bundle root>`. A version older than `fdf version` prints,
  or a root that is not this bundle's, means an install is due: propose
  `fdf install --project <harness>`, where the harness is `claude-code`,
  `codex` or `opencode`. It refreshes the skills and the managed primer
  together; with everything current it writes nothing. A user-level install
  (under the home directory) loads too: report its version, never change it.
- **The managed primer** — the `## Feature Document Format` section of each
  instruction file. `fdf install` writes it word for word and refreshes it
  only while it is exactly a text some fdf release shipped; once anyone edits
  it, every later install leaves it alone (reporting `differs from the
  shipped primer (user-edited?)`) and it goes stale with the next spec
  version. Any line about *this* project inside it is a hand edit. To see
  the current primer without touching the project, install into a scratch
  repository and compare the sections:

  ```bash
  s=$(mktemp -d) && git -C "$s" init -q && (cd "$s" && fdf install --project <harness> >/dev/null) && cat "$s/CLAUDE.md"   # AGENTS.md for codex/opencode
  ```

  The fix for an edited section: move the project's lines into the file's
  own sections — checked like any other line, so one that points at nothing
  is dropped with its own finding — then delete the managed section (the one
  hand step, since `fdf install` will not replace an edited section) and
  propose `fdf install --project <harness>` to write the current one.

## 2. Context documents against the code

Go through each Context document **claim by claim**. Every sentence that
names something checkable is a claim, and each kind has one place to check:

| The document names… | Check it against |
|---|---|
| a path or directory ("new endpoints go in `src/routes/`") | the tree — it exists, and that code still lives there |
| a version ("Node 18", "MySQL 8.0") | manifests, lockfiles, runtime pins, CI, container base images, IaC |
| a command or tool ("`npm run e2e`", "`tilt up`") | the task runner, the CI workflows, and the file the tool needs (its config) |
| a deployment target, environment or hosted service | deploy config, CI deploy jobs, IaC |
| a `code:` identifier in `DOMAIN.md` | `git grep -nw '<identifier>' -- ':!*.md'` — the type, table or field exists under exactly that name, in code rather than prose |
| a route, event or error shape | route registrations, API schemas, handlers |
| a practice, module or component by name | `practices/` and the tree |
| a rule or convention | a sample of the code it governs |

Then look the other way: what the evidence shows that no document mentions —
a new top-level directory, a new data store or client library, a new deploy
target, models, tables and routes added since the baseline that `DOMAIN.md`
has no term for.

Every mismatch has two readings, and they take opposite fixes:

- **The document is stale** — the project moved on deliberately: a dependency
  was upgraded, CI moved, a module was split. Propose the edit that makes the
  document true, written as today's snapshot. History belongs in `LOG.md`
  ("migrated from MySQL in 2025"), and a plan belongs nowhere until it is true
  ("we will move to Kubernetes").
- **The code drifted** — the document still states a decision the project
  holds, and newer code ignores it. That is not a document edit; never rewrite
  a rule to match the worst code in the tree. Check `fdf debt --open` first —
  the gap may already be filed. If not, propose a debt
  (`fdf debt [<group>/]<slug>`) naming the paths; and whatever part of the
  difference a user can observe goes through fdf-help as well.

When the evidence does not say which reading is true, ask. A `DOMAIN.md`
term whose `code:` names nothing that exists, while the code uses a banned
word for the same thing, is the classic case: the lexicon may be out of date,
or the code may have drifted from it. A practice whose `# Rules` the code
under its `applies-to` has visibly left behind takes the same two readings.

`DOMAIN.md` governs the bundle's documents and the code's identifiers. A
banned word in a type, table, field or route name is drift worth reporting.
The same word in a UI label, a locale file, help text or other user-facing
copy is a surface decision (`SURFACES.md`) and never a finding.

## 3. The documents against each other

Read every section of every Context document and ask one question of each:
**is this fact's home here?** (the table in *One home per fact*). True is not
the same as current here: a fact outside its home is a finding even when it
is accurate today — it is the copy that goes stale next.

- **Repetition** — a fact outside its home: a dependency list in
  `ARCHITECTURE.md`, build commands in `STACK.md`, a practice's rules restated
  in `ARCHITECTURE.md`. Propose replacing the copy with a link to its home.
- **Contradiction** — one fact, two values: `STACK.md` says Redis 6 and
  `INFRA.md` provisions Redis 7. Settle it from the evidence, not from which
  document is newer, and keep the fact in its home only.
- **Misfiling** — a mechanism described in depth inside `ARCHITECTURE.md` is a
  practice waiting to be extracted (`fdf practice`, then a one-line link where
  the text was); one feature's behavior written into `SURFACES.md` belongs to
  that feature.
- **Practices** — the practices `ARCHITECTURE.md` lists are the ones under
  `practices/`, and none it calls current is `superseded`.
- **Vocabulary** — every document in the bundle is internal language,
  episodic ones included, and so are the instruction files: they use
  `DOMAIN.md`'s canonical names. F12 scans only Gherkin and declared scenario
  names, so the prose is yours. Build the pattern from every `instead-of`
  word in `DOMAIN.md`:

  ```bash
  grep -rnwiE --exclude=SPEC.md '(<word>|<word>|…)s?' docs/features CLAUDE.md AGENTS.md
  ```

  A hit that names the concept is a finding, and its fix is a lexicon fix —
  in a spec, plan, task, change or log as much as in a Context document. A
  hit in another sense is not — quoted surface wording (the label a screen
  shows for the term), a scaffold heading such as `## Data stores` when
  "store" is banned, or the examples in the managed primer, which is fdf's
  text.

## 4. Agent instruction files

List every instruction file the repository carries, at any depth —
`CLAUDE.md`, `AGENTS.md`, and any other a harness in use reads (`GEMINI.md`,
`.github/copilot-instructions.md`):

```bash
git ls-files | grep -E '(^|/)(CLAUDE|AGENTS|GEMINI)\.md$|copilot-instructions\.md$'
```

A file outside the repository, such as a user-level `~/.claude/CLAUDE.md`,
belongs to the user's machine, not the project: report what you notice there,
and never edit it.

An instruction file should **point into the bundle, not copy it**. Go through
each file section by section, and sort every section, bullet by bullet, into
one of these:

| It is… | Then |
|---|---|
| a pointer into the bundle ("Stack: see `docs/features/STACK.md`") | keep it — check that it resolves |
| a harness or workflow note that lives nowhere else | keep it |
| the managed `## Feature Document Format` primer | `fdf install` owns it — see *Mechanical checks* |
| a fact whose home is a Context document or a practice: a stack list, build or test commands, an architecture sketch, a glossary, conventions | propose a one-line pointer instead. If the user wants it loaded into every session anyway, it may stay — but it must match its home exactly, and the home wins any disagreement |
| a standing rule ("always do X when Y") | if the code already does X, it is a practice: its `applies-to` routes it to exactly the work it governs, which an instruction file cannot. Propose extracting it (`fdf practice`) and leaving a pointer |
| an instruction that conflicts with the FDF workflow ("write the feature doc after the code is merged") | ask whether it is deliberate — explicit user instructions outrank the skills, so it is not drift to delete. If deliberate, it should say so plainly; if not, it goes |

Then check what the sections refer to:

- **References resolve.** Every path the file names exists — the bundle root
  above all, if the project moved it or sets `FDF_ROOT_DIR`. Every `fdf`
  command it names exists (`fdf help` lists them), and every skill it names
  is installed. The `/fdf-init`, `/fdf-new` and `/fdf-validate` slash
  commands were removed when the skills replaced them; a file still teaching
  them is stale.
- **No contradiction with the bundle.** "Tests run with `npm test`" where
  `INFRA.md` says pnpm is fixed at whichever end the evidence says is wrong.
- **Two files, one source.** When `CLAUDE.md` and `AGENTS.md` both carry
  project content, it lives in one and the other points to it — Claude Code,
  for example, can import `@AGENTS.md` — so the two cannot disagree. Each
  keeps its own managed primer.

## A document that is wrong throughout

When a Context document no longer describes the project at all — a rewrite, a
new platform — do not patch it line by line. Interview the user about that
area one question at a time (the *What to ask* list in fdf-init is the
question bank), draft the whole document, and get it approved as a whole.

## Rules

- Evidence before findings: every finding quotes the line it rests on, from a
  file opened in this session.
- Propose first; edit only on explicit approval; log every approved change.
- Documents only. Code that breaks a rule the project still holds is a debt or
  routed work, never a reason to weaken the document.
- One home per fact; everywhere else, a link.
- Never write into `SPEC.md` or the managed primer — `fdf migrate` and
  `fdf install` rewrite them. Deleting an edited primer section so
  `fdf install` can write it fresh is the one hand step.
- Never drop a line inside a rewrite without saying so.
- Never edit an episodic document to agree with today — except a lexicon
  fix, which changes only the words `DOMAIN.md` bans.
- Context documents are snapshots: no history, no plans.
- `fdf validate` exit 0 at the end.
