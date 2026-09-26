---
name: fdf-init
description: Use right after `fdf init` on a new FDF bundle, or whenever `fdf validate` says a Context document (STACK.md, ARCHITECTURE.md, SURFACES.md, INFRA.md, DOMAIN.md) is still an unfilled stub — before any feature work. Documents already filled but possibly stale are fdf-checkpoint's.
---

# FDF Init

Interview the user about the project, then write the five bundle-root
**Context documents** — `STACK.md`, `ARCHITECTURE.md`, `SURFACES.md`,
`INFRA.md`, `DOMAIN.md` — that every later feature relies on, and write down
the practices an existing codebase already follows. This is the difference
between agentic engineering and vibe coding: with accurate context an agent
builds *this* project's way; without it, it guesses. Take it slowly and do it
thoroughly: it is the most leveraged conversation in the whole workflow.

New to FDF? The fdf-help skill explains how the fdf skills fit together and
the mechanics they share. The format rules are in the bundle at
`docs/fdf/SPEC.md`; `fdf help` documents every command.

**Before you finish**, check that you did each of these — they are the steps
most often skipped:

- one question per topic, asked before you draft (step 3);
- an explicit yes before you write each document (step 4), and before each
  practice, debt and instruction-file edit;
- a fresh `date -u +%Y-%m-%dT%H:%M:%SZ` output as each document's
  `timestamp`, one run per file (step 5);
- the offer to map the code with fdf-adopt (step 7);
- the strict-mode question (step 10), and the closing message (step 11).

`fdf init` writes the five Context documents as stubs: each carries a
`<!-- fdf:stub -->` marker and a ⚠️ banner. Your job is to replace each stub
with real content. Rule F9 fails any bundle that has a feature while a Context
document is still a stub — so this interview comes first.

## Mechanics

- A `timestamp:` you set is the output of `date -u +%Y-%m-%dT%H:%M:%SZ`, run
  at the moment of the edit — never a time you estimate, round or reuse, and
  never your local time with `Z` added. A restored document keeps its old
  one; a malformed one already committed is repaired by fdf-validate's F1,
  never set to now.
- Log with single quotes — `fdf log '**Context**: …'` — because inside double
  quotes the shell runs anything between backticks. An apostrophe would end
  the quotes: write ’ instead. A practice's log takes its full ID,
  `<practice-id>`, such as `practices/permission-checks`. An entry that names
  a banned word — recording the ban itself — puts it in a code span, or F12
  flags the log.
- `fdf practice`, `fdf debt` and `fdf bug` scaffold a document full of `TODO`
  text, and add a listing line to the `INDEX.md` beside it that ends
  `- TODO.` — and one for each group it creates, in the parent `INDEX.md`.
  Replace every one with real text; `fdf validate` warns until you do.

## The five documents

| File | Captures |
|---|---|
| `STACK.md` | Languages, runtimes, frameworks, libraries, data-store engines — and their versions. What the code is written in and with. |
| `ARCHITECTURE.md` | The architecture style, how code is organized, the design principles and conventions contributors follow, the load-bearing decisions behind them (hard constraints included), and which practices exist. |
| `SURFACES.md` | The project's purpose and what is out of scope, and the interface principles for **every** surface through which people or systems engage the project — APIs, human UIs, CLIs, events and webhooks, file formats, other inputs (not "UI only"). Naming, errors, versioning, UX principles, CLI conventions, assets and exemplars. |
| `INFRA.md` | How the project is built, tested, packaged and deployed; its environments and runtime targets; where its services and data stores are hosted; its operational dependencies (CI, hosting, queues, caches, secrets). |
| `DOMAIN.md` | The project's **domain language**: one canonical name per concept, the words banned in its place, and how each name appears in the code. Internal vocabulary only — what a surface shows a person (labels, locale strings) is SURFACES.md's business. |

Each fact has **one** home: a data store's engine and version go in
`STACK.md`, where it is hosted goes in `INFRA.md`; an API design rule goes in
`SURFACES.md`, not `ARCHITECTURE.md`. A fact written in two documents is the
copy that goes stale first.

Each document is a **current snapshot** — not a roadmap and not a changelog.
Write only what is true now (or true for the project being started).

A **surface** is any interface through which people or systems engage the
project (API, UI, CLI, events). FDF avoids "design" and "UX" as document names
— both are GUI-connoted, and "design doc" collides with the spec's and the
architecture's territory.

## Process

1. **Find the stubs.** Run `fdf validate` (it respects `--root` and
   `FDF_ROOT_DIR`).
   - No bundle yet → run `fdf init` first.
   - The bundle pins a version before 1.0, or it is a `docs/features/` whose
     `INDEX.md` pins none (`fdf validate` says so) → it must be upgraded
     first, and the upgrade is **the user's decision**: propose it, say what
     it does, and wait (fdf-help, *A bundle from before 1.0*). If the user
     declines, stop here.
   - A Context document reported **missing** (`recommended context document
     is missing`, or `required once the bundle has features`) → run
     `fdf init` again: it adds only what is missing, as a stub.
   - Work **only** on the documents `fdf validate` names as
     `still an unfilled stub` (`fdf migrate` says `freshly scaffolded stub`).
     A document that is already filled is fdf-checkpoint's, however thin it
     looks — never re-interview or rewrite it here. Read each stub, so you
     keep its heading structure.
   - Run `fdf adopt` (no arguments): it lists the features the bundle already
     has. A bundle with features — typically one just migrated — needs only
     its stubs filled: skip steps 6 and 7 unless the user asks for them, and
     say only what is true of it in the *Closing*.
2. **Survey what already exists.** If there is code, read enough to ground your
   questions: `README`, manifests (`package.json`, `go.mod`, `Cargo.toml`,
   `pyproject.toml`, `pom.xml`), lockfiles, CI config, Dockerfiles, IaC,
   OpenAPI or route trees, UI entry points, CLI command trees — and the agent
   instruction files (`CLAUDE.md`, `AGENTS.md`), which often already state the
   stack, the commands and the conventions. Come to the interview with
   informed guesses to confirm, not a blank slate. For a greenfield project
   there is nothing to read — the interview *is* the design.
   For `DOMAIN.md`, harvest the **nouns**: model and entity names, database
   tables, top-level API resources, event names. Where two of them plainly
   mean the same thing, you have found the lexicon's first entry, and the most
   useful question to ask. UI copy and locale files show what users call a
   thing — worth an `instead-of` entry so the documents don't drift toward it
   — but a label that differs from the code's name is not a defect, and the
   interview never proposes renaming it.
3. **Interview — one question at a time.** Prefer multiple-choice questions
   with a recommended answer (your informed guess from step 2); let the user
   redirect. Never send a questionnaire. Cover the areas in *What to ask*
   below, skipping what genuinely does not apply (and say why). Chase vague
   answers ("scalable", "modern", "cloud") into specifics. Ask follow-ups when
   an answer implies more (chose microservices → how do services
   communicate? chose a SPA → what backend serves it?). When the project is
   interface-heavy, start with its surfaces — who and what talks to it, and
   how — then the stack and architecture that serve them.
4. **Draft each document, and get it approved one at a time.** Before you
   present a draft, check each convention and principle it states against
   the code (search for it, and for its alternatives). Where the code and an
   answer disagree, show both and ask. Where a principle has holdouts, name
   them in the draft ("…except `refunds.Retry` and `payouts.Hold`, which
   open their own database connection"), or propose a practice and a debt
   (step 6). Present
   the draft of one document, take edits, and ask whether the user approves
   it before you move to the next. File each fact in its one home (the table
   above). These documents change only with approval once written, so the
   user must genuinely agree now.

   When you present `DOMAIN.md`, say this, in your own words: "This governs
   the words in the bundle and the names in the code — not what screens,
   messages and help text say, which may use other words on purpose." Then
   nobody later "fixes" the UI to match the lexicon.
5. **Write each approved document.** Fill the stub in place: keep its
   frontmatter and every heading outside a comment — write `None.` under one
   that does not apply — and delete the `<!-- fdf:stub -->` line (it is what
   F9 reads as "unfilled"), the ⚠️ banner, and every `<!-- … -->` guidance
   comment: `DOMAIN.md`'s holds a sample term the validator would read as
   real. Keep `type: Context`, and set its
   `timestamp` to the output of `date -u +%Y-%m-%dT%H:%M:%SZ`, run now for this
   document — never the stub's old value, and never a local time with `Z`
   added.
6. **Surface the practices the project already follows.** On an existing
   codebase this is the most valuable part of the interview — see *Practices
   in an existing project* below, which ends with the steps to write one. On a
   greenfield project there is nothing to surface yet; say so and move on.
   Once practices exist, `ARCHITECTURE.md` lists them (a line and a link
   each, under a `## Practices` heading of its own when it has none): propose
   that edit with them. A design principle that became a practice keeps only
   that one line in `ARCHITECTURE.md`: its rules live in the practice, and
   the files that do not follow it yet in its debt. Any later edit to a
   Context document — this list, `strict: true` — sets its `timestamp` again
   from a fresh `date -u +%Y-%m-%dT%H:%M:%SZ`.
7. **Offer to map what the code already does.** An existing codebase's
   capabilities have no feature documents yet. Mapping them — one adopted map
   entry per capability, breadth first — is its own phased job, run by the
   **fdf-adopt** skill once the five documents are filled. Ask whether the user
   wants it now; do not start it inside this interview.
8. **Point the instruction files at the new documents.** Where `CLAUDE.md` or
   `AGENTS.md` restates what the five documents now hold — a stack list, build
   commands, conventions — propose replacing each such passage with a one-line
   pointer ("Build and test: see `docs/fdf/INFRA.md`"), so every fact has one
   home from day one. Only the project's own sections: **never edit the
   `## Feature Document Format` section** — `fdf install` wrote it and owns it,
   stops refreshing it once anyone edits it, and its description of the
   Context documents is the format, not a restated fact. fdf-checkpoint's
   *Agent instruction files* section is the full check.
9. **Log and gate.** Log the interview in the root `LOG.md` — it concerns the
   whole bundle — naming the documents you filled:
   `fdf log '**Context**: STACK.md, ARCHITECTURE.md, SURFACES.md, INFRA.md and DOMAIN.md filled by the fdf-init interview; <key decisions>.'`
   Then run `fdf validate`: it must exit 0 **and** print no
   `still an unfilled stub` line and no missing Context document — with no
   feature yet, a stub is only a warning, so exit 0 alone does not prove the
   work is done. If it fails on anything else, use fdf-validate.
10. **Ask about strict mode.** Run `fdf lexicon --all`. When it reports no
    banned word, ask: "No document uses a banned word — shall I make banned
    words errors from now on (`strict: true` in `DOMAIN.md`)?" Add it only on
    a yes: then set `DOMAIN.md`'s `timestamp`, log it, and validate again.
    With no yes, leave it off.
11. **Hand off the responsibility.** Tell the user plainly (see *Closing*).

## What to ask

These are topics, not a script: ask one question at a time, led by what you
found in step 2 — "Node 20 with Postgres 15, deployed on Fly.io: is that
right?" — and move on once the answer is clear.

**Purpose & scope → `SURFACES.md` (`## Purpose`, `## Out of scope`); hard
constraints → `ARCHITECTURE.md` (`## Key decisions`)**
- What is this project, in one sentence? Who uses it, and to do what?
- What are the primary use cases? What is explicitly out of scope?
- Greenfield or existing? Any hard constraints (compliance, offline, latency,
  budget, team size or skills)?

**Surfaces → `SURFACES.md`**
- Which surfaces exist or will exist: HTTP APIs, human UIs (web, mobile,
  desktop), CLIs, events and webhooks, file formats and import/export, other
  inputs?
- For each: who is the audience (end user, operator, another service)?
- API: naming, versioning, error envelope, pagination, auth presentation.
- Human UI: structure, UX principles, brand, accessibility baseline, links to
  design systems or exemplars.
- CLI: command shape, flags, output formats, exit codes.
- Events and inputs: shapes, validation philosophy, operator-facing feedback.
- Assets and exemplars to point agents at; what is explicitly out of scope for
  surface work.

**Stack → `STACK.md`**
- Primary language(s) and runtime version. Why these?
- Frameworks and major libraries (web, ORM, UI, testing, …).
- Data stores: relational, document, key-value, cache, search, blob — which
  engines and versions, and what each holds.
- External services and APIs the project depends on.
- Version floors that matter (language edition, framework major).

**Architecture → `ARCHITECTURE.md`**
- Shape: modular monolith? microservices? serverless functions? a library or
  SDK? a CLI? a frontend, a backend, or both? A mix?
- If there are several services or modules: how do they communicate (HTTP,
  gRPC, queues, events)? Where are the boundaries?
- Code organization: by layer, by feature or domain, by service? Monorepo or
  several repositories? Where does new code go?
- Design principles and conventions to follow (DDD, hexagonal, dependency
  injection, a functional core, an error-handling style).
- State and data flow: where does state live, and how does data move through
  the system?
- The few decisions a new contributor must not silently violate.

**Domain language → `DOMAIN.md`**
- What are the core things this system is about? Aim for the 5–15 nouns a new
  contributor must get right, not an exhaustive ontology.
- For each: what does the team call it — and what *else* has it been called
  (a legacy name, a vendor's name, slang, the name in the system it replaced)?
  Those others are its `instead-of` list. Ban only the words the user
  confirms.
- Which pairs are easy to confuse, and where is the line between them (a
  Product versus the Line Item that sells one)?
- How does each name appear in the code — type, table, field, route segment?
- Where does the wording people see differ from the term — the screen says
  "store", the code says `Venue`? Keep the label; note the mapping in
  SURFACES.md's conventions for that surface. The lexicon is for the inside.
- Any word the team has already argued about? Settle it here; that is exactly
  what this document is for.
- Start small and true. A lexicon of six real terms beats thirty invented
  ones; it grows as features arrive.

**Infrastructure → `INFRA.md`**
- Build and test: how is it built and tested? Which commands? Which CI?
- Packaging and artefacts: binaries, containers, packages, bundles?
- Environments: local, staging, production — how do they differ?
- Deployment: where does it run (a cloud provider, Kubernetes, serverless,
  on-premises, app stores, package registries)? How does a change reach
  production?
- Operational dependencies: where the data stores, caches, queues, CDNs,
  secrets and observability are hosted. What has to exist for the system to
  run?
- Targets: OS and architecture, browsers, mobile platforms, runtime versions.

## Writing DOMAIN.md

The `# Terms` section is **parsed** (rule F12), so its shape is not free-form:

```markdown
# Terms

## Venue
A physical location where a merchant sells. Owns its own inventory and staff.
- instead-of: store, shop, tenant
- code: `Venue` (model), `venues` (table), `venue_id` (foreign key)
```

One `## <Term>` per canonical term; the first line under it is the
definition; `instead-of` lists the banned words — only the words the user
confirmed, often just one. `code:` names identifiers you found in the code;
a concept with none says so: `- code: none — a string field on Order`.

A banned word that validate flags in another sense puts the ban in doubt:
ask the user whether to drop a word they never confirmed, or whether that
sense becomes an `except:` phrase. Never reword a document only to hide a
ban. F12 rejects a duplicate term,
a term with no definition, a banned word that is itself a term, and a word
claimed by two terms. A banned word is reported wherever the bundle chooses its
words — every document except `SPEC.md`, `DOMAIN.md`, `slug.test.md` and
`slug.surface.md`, and every group, slug and task name — as a warning;
`strict: true` in `DOMAIN.md`'s frontmatter makes them errors in every
validation (step 10 asks about it).

When a banned word has an ordinary second sense in this project — *store* in
"data store", *branch* in "git branch" — list the qualified phrase under
`- except:` so it is never reported:

```markdown
## Venue
A physical location where a merchant sells.
- instead-of: store, shop, branch
- except: data store, git branch
```

An `except:` entry is always a phrase, never the banned word alone (F12
rejects that: it would un-ban the word). Before banning a word, ask how often
the project uses it in another sense; `fdf lexicon --all` shows every
occurrence once the term is in place.

A bundle that already uses a word you ban — any bundle migrated from an
earlier version — gets swept in the same approved edit: a **lexicon fix** in
every document that uses the word for the concept, frozen specs, plans, tasks,
changes and logs included. A lexicon fix changes words, never what was
delivered, so it needs no Change; fdf-validate (F12) says how to do it without
breaking the scenario-name joins.

Depth does not belong here: a term needing more than a definition gets a
practice, and the term links to it with a `- see: /practices/<slug>` line —
added once the practice exists (nothing checks that it does).

**Scope.** The lexicon governs the project's internal language: every document
in the bundle, and the identifiers in the code. It does not govern what a
person reads on a surface — UI labels, locale and translation files, help
text, notifications, marketing copy — nor the names an external system fixes
at its own boundary. Those may use a banned word on purpose, and the mapping
("Venue is shown to merchants as `Store`" — the label in a code span, so the
lexicon reads it as quoted) belongs in SURFACES.md, not in
`# Terms`.

## Practices in an existing project

A project that has been running for a while **already has practices**. Nobody
wrote them down, so they live in the code and in the heads of whoever wrote
it, and every new contributor — human or agent — rediscovers them by guessing
and being corrected. Writing them down is the single biggest thing this
interview does for a brownfield codebase.

Your job is to **find what is already true**, not to propose what would be
nice. A practice that describes an aspiration is a lie the next agent will act
on.

**Where to look.** Read the code for *repetition*, not for features:

- **Entry points that everything passes through** — middleware or interceptor
  chains, filters, base classes, decorators, framework hooks. The list of
  middleware in the router setup is very close to a list of practices.
- **Shared packages** — `internal/`, `common/`, `lib/`, anything named
  `*_helper`, `*_util`, `Base*`. Code put there because it is used everywhere
  is a mechanism used everywhere.
- **The first few lines of every handler, job or command.** What do they all
  do before their own work? Auth? Tenant scoping? Opening a transaction?
  Input validation? That is a practice.
- **Test setup.** What every test has to arrange reveals what every code path
  assumes.
- **Prose that already exists** — `CONTRIBUTING.md`, ADRs, a wiki, the long
  comment at the top of a core file, the review comments in git history.
  Someone already wrote some of this; do not make them write it twice.
- **Where the codebase disagrees with itself** — two ways of doing one thing.
  This is the most valuable practice to write and the one you must not decide
  alone: present both, with counts and file paths, and ask which is canonical.

For each candidate, search for **every** place it applies and every place it
is broken (`grep -rn` the pattern and its alternatives), so your counts are
real.

**What to ask.** These questions get at practices better than asking about
practices directly:

- What do you find yourself saying in code review over and over?
- What did a new contributor get wrong in their first week?
- What is the thing someone did once that you had to revert?
- If someone adds a new endpoint tomorrow, what must they not forget?
- Which part of this codebase would you be nervous about an agent touching,
  and what is the rule that would make you less nervous?

**Partial conformance — write the practice *and* the debt.** The honest
finding on a real codebase is almost never "all thirty handlers do this". It is
"eighteen do, twelve don't". The **practice** states the rule the project
follows; a **debt** names the files that have not caught up. Both are true,
and neither has to lie for the other to exist. Do not turn it into an audit: a
debt per practice that is not universal, not a debt per file.

**Restraint.** Practices are not gated by any rule — a bundle with none is
perfectly valid, so there is no quota to fill.

- Write at most about seven — the ones a contributor must not get wrong —
  and fewer, or none, when fewer are backed by evidence. A library of twenty is
  a library nobody reads.
- Every practice must be backed by evidence you can point at: the files where
  it already happens. If you cannot cite them, you are inventing it — ask
  instead.
- If the user wants a *new* way of doing something, that is work on the code,
  not a practice claiming it is already how things are. File it as a debt if
  it is worth tracking, and move on.

What you find while reading the code that is not a gap but a **defect** — the
software doing something wrong that someone could observe — is a bug, not a
debt. File it, and tell the user you did:
`fdf bug --resource <paths> [<group>/…]<slug>`, with the evidence under
`# Symptom` (say it was found by reading) and what should happen under
`# Expected` (fdf-debug, *Not repairing it now*, step 3). Add
`--affects <feature-id>` when a feature already documents that code; leave
`affects` unset only when none does. Do not fix it inside this interview.

**Writing one** — in this order, and nothing is written before step 2:

1. **Propose it.** Present the rule, the files that follow it and the ones that
   don't, with counts, and the line `ARCHITECTURE.md`'s `## Practices` list
   will gain (a new heading at the end of the document when it has none). If
   some files don't follow it, propose the debt that names them in the same
   message. Ask whether the user approves.
2. **On approval, scaffold and fill the practice**: `fdf practice
   [<group>/…]<slug>`, then fill it in:

   ```markdown
   ---
   type: Practice
   status: active
   title: Permission checks
   description: Where authorization decisions are made, and how they are expressed.
   applies-to: [internal/authz, internal/http]
   timestamp: <output of date -u +%Y-%m-%dT%H:%M:%SZ, run now for this file>
   ---

   # Rules

   - Every handler resolves permission through `authz.Can(ctx, action, resource)`.
     No handler reads roles or plan flags directly.
   - A denial on a resource the caller may not know exists returns 404, not 403.

   # How

   `authz.Middleware` resolves the principal once per request onto the context;
   `Can` is a pure function with no I/O, so handlers stay testable.

   # Boundaries

   Background jobs run as the system principal and do not call `Can`.

   # Rationale

   One place decides who may do what, so a rule changes in one file.
   ```

   Only for a divergence the user approved, add `# Exceptions` with one
   bullet per file: `- internal/http/health.go — unauthenticated by design.`

   `# Rules` is required and non-empty (F11) — imperative, short, what code
   MUST do; everything else explains it. `fdf practice` scaffolds `# Rules`,
   `# How`, `# Boundaries` and `# Rationale`. On approval, also add the
   practice's line to `ARCHITECTURE.md`'s `## Practices` list and set that
   document's `timestamp`. `applies-to` lists the repo paths it
   governs, and each must exist (R1): it is how later work finds the practice,
   since a feature never lists the practices it follows. A practice carries no
   Gherkin, spec, plan or tasks. Replace the scaffold's `TODO` text and its
   listing line in `practices/INDEX.md` (or its group's index) with the real
   description. Keep the `timestamp` the scaffold wrote, or paste fresh
   `date -u` output — never type one.
3. **On approval, file the debt** for the files that have not caught up:
   `fdf debt --resource <paths> [<group>/…]<slug>`, then fill it in:

   ```markdown
   ---
   type: Debt
   status: open
   title: Legacy handlers read roles directly
   description: Twelve handlers check roles by hand instead of calling authz.Can.
   resource: [internal/http/admin.go, internal/http/reports.go]
   timestamp: <output of date -u +%Y-%m-%dT%H:%M:%SZ, run now for this file>
   ---

   # Gap

   Twelve handlers in `internal/http` read `user.Role` directly instead of
   calling `authz.Can`. They predate the permission-checks practice, which the
   rest of the codebase follows.

   # Cost

   Each is a place a permission rule can be changed in one spot and missed in
   another.
   ```

   `# Gap` is required and must be concrete enough for someone else to
   confirm — files and counts, not impressions (F13). `resource` names the
   paths carrying the gap, and it is the whole link: the practice the debt
   concerns is found by comparing those paths with the practice's
   `applies-to`, so never write a pointer to the practice by hand. Replace its
   listing line in `debts/INDEX.md` with the description. Keep the
   `timestamp` the scaffold wrote, or paste fresh `date -u` output — never
   type one.
4. **Log the practice's approval** in its own log:
   `fdf log <practice-id> '**Decision**: written with the user’s approval; followed in <n> files.'`
   Log only a yes you received in step 1; name a person only as the user
   named themselves.

If the user declines a practice or a debt, write nothing for it.

## Closing (say this explicitly)

Tell the user, in your own words, naming the documents you filled:

> STACK.md, ARCHITECTURE.md, SURFACES.md, INFRA.md and DOMAIN.md are written
> and validated. Treat them as **critical, living documents**: from here on I
> will not change them without your explicit approval, and after each feature
> I'll ask whether any of them needs updating (a new dependency, a new
> pattern, a new surface convention, new infrastructure, a new term) and only
> edit on your say-so, logging the change. Work that no feature records — a
> dependency upgrade, a CI move — can still make them stale, so run the
> fdf-checkpoint skill now and then, and before each release: it re-checks
> them, `SPEC.md` and CLAUDE.md/AGENTS.md against the code and each other.
> DOMAIN.md governs the words in the bundle and the names in the code, not
> what screens and messages say. The practices under `practices/` are the
> same as the Context documents: binding on all code that
> matches their `applies-to`, and mine to propose but yours to approve.
> *(When no feature documents the existing code yet:)* The code that already
> exists has no feature documents yet; the fdf-adopt skill maps it —
> capability by capability, breadth first — whenever you are ready.
> Keeping all of this accurate is what makes this agentic engineering rather
> than vibe coding — stale context produces confidently wrong work. They're
> yours to own; I'll help maintain them.

## Rules

- One question at a time; approve each document before writing it. Answers
  you think you already hold are guesses: ask one question per area anyway,
  and present each draft for a yes.
- Write only what is true now — snapshots, not aspirations or history. Every
  claim about the code (a principle, a convention, a count) is one you checked
  in the code.
- Work only on documents that are still stubs; a filled one is
  fdf-checkpoint's.
- Deleting the `<!-- fdf:stub -->` line is required: a Context document that
  still contains it counts as unfilled and fails F9 once a feature exists.
- After this interview, these five files change only with explicit user
  approval, always logged. Whichever skill finds the need proposes the edit —
  most often the review that ends fdf-execute and fdf-change, or
  fdf-checkpoint. Never edit them casually mid-implementation.
- A practice records what the project **already does**, with files you can
  cite. Never write one from what would be good practice in general; that is
  how a bundle starts lying on day one.
- Where a practice is followed almost everywhere, the practice states the rule
  and a debt names the holdouts. Never soften a rule to match the worst code in
  the tree.
- Practices are never mandatory. Writing none is a valid outcome, and a better
  one than writing fiction.
