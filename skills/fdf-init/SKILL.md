---
name: fdf-init
description: Use right after `fdf init` on a new FDF bundle, or whenever STACK.md / ARCHITECTURE.md / SURFACES.md / INFRA.md / DOMAIN.md are still unfilled stubs — runs the project-context interview that fills the five critical Context documents, and writes down the practices an existing codebase already follows, before any feature work. Documents already filled but possibly stale are fdf-checkpoint's.
---

# FDF Init

Interview the user about the project, then write the five bundle-root
**Context documents** — `STACK.md`, `ARCHITECTURE.md`, `SURFACES.md`,
`INFRA.md`, `DOMAIN.md` — that every later feature relies on. This is the
difference between agentic engineering and vibe coding: with accurate context
an agent builds *this* project's way; without it, it guesses. Take it slowly
and do it thoroughly: it is the most leveraged conversation in the whole
workflow.

New to FDF? The format is defined in the bundle at `docs/features/SPEC.md`.
`fdf spec` prints the format rules and `fdf help` documents every command.

The five Context docs live at the bundle root beside it; `fdf init` writes
them as stubs carrying `<!-- fdf:stub -->` and a ⚠️ banner. Your job is to
replace each stub with real content. Validation rule F9 rejects any bundle
that has features while a Context doc is still a stub — so this interview
comes first.

## The five documents

| File | Captures |
|---|---|
| `STACK.md` | Languages, runtimes, frameworks, libraries, data stores — and their versions. What the code is written in and with. |
| `ARCHITECTURE.md` | The architecture style, how code is organized, the design principles and conventions contributors follow, and the load-bearing decisions behind them. |
| `SURFACES.md` | Interface and interaction principles for **all surfaces** through which people or systems engage the project — APIs, human UIs, CLIs, events/webhooks, file formats, and other inputs (not “UI only”). Naming, errors, versioning, UX principles, CLI conventions, assets/exemplars. |
| `INFRA.md` | How the project is built, tested, packaged, deployed; its environments and runtime targets; the operational dependencies (CI, hosting, queues, caches, secrets). |
| `DOMAIN.md` | The project's **domain language**: one canonical name per concept, the words banned in its place, and how each name appears in the code. Internal vocabulary only — what a surface shows a person (labels, locale strings) is SURFACES.md's business. |

Each is a **current snapshot**, not a roadmap and not a changelog. Write only
what is true now (or true for the project being started).

A **surface** is any interface through which people or systems engage the
project (API surface, UI, CLI, events). FDF deliberately avoids “design”/“UX”
as document names — both are GUI-connoted, and “design doc” collides with
Spec/Architecture territory.

## Process

1. **Locate the bundle and read the stubs.** Run `fdf validate` (respects
   `--root`/`FDF_ROOT_DIR`). Read the five stub files so you match their
   heading structure. If they don't exist yet, run `fdf init` first.
2. **Survey what already exists.** If there's code, read enough to ground your
   questions — `README`, manifests (`package.json`, `go.mod`, `Cargo.toml`,
   `pyproject.toml`, `pom.xml`), lockfiles, CI config, Dockerfiles, IaC, OpenAPI
   or route trees, UI entry points, CLI command trees — and the agent
   instruction files (`CLAUDE.md`, `AGENTS.md`), which often already state the
   stack, the commands and the conventions. Come to the interview with
   informed guesses to confirm, not a blank slate. For a greenfield project
   there's nothing to read — the interview *is* the design.
   For DOMAIN.md specifically, harvest the **nouns**: model and entity class
   names, database tables, top-level API resource paths, event names. Where
   two of them plainly mean the same thing, you have found the first entry the
   lexicon needs, and the most useful question to ask. UI copy and locale
   files are evidence of what users call a thing — worth an `instead-of`
   entry so the documents do not drift toward it — but a label that differs
   from the code's name is not a defect, and the interview never proposes
   renaming it.
3. **Interview — one question at a time.** Prefer multiple-choice with a
   recommended default; let the user redirect. Do not dump a questionnaire.
   Cover the areas below, skipping what genuinely doesn't apply (and say why
   you're skipping). Chase vague answers ("scalable", "modern", "cloud") into
   specifics. Ask follow-ups when an answer implies more (chose microservices
   → how do services communicate? chose a SPA → what backend serves it?).
   **Start with surfaces** when the project is interface-heavy: who/what
   talks to this system, through which APIs/UIs/CLIs/events, and what those
   interactions must feel like — then stack and architecture that serve them.
4. **Draft each document and get approval section by section.** Present a
   draft, take edits, confirm. These become immutable-without-approval the
   moment they're written — so the user must genuinely agree now.
5. **Write the five files.** Replace the entire stub (remove the
   `<!-- fdf:stub -->` sentinel and the ⚠️ banner — their presence is what
   F9 treats as "unfilled"). Keep `type: Context` and the frontmatter; update
   `timestamp`.
6. **Surface the practices the project already follows.** On an existing
   codebase this is the highest-value part of the whole interview — see
   *Practices in an existing project* below. On a greenfield project there is
   nothing to surface yet; say so and move on.
7. **Offer to map what the code already does.** An existing codebase's
   capabilities have no feature documents yet. Mapping them — one adopted
   *map entry* per capability, breadth first — is its own phased job, run by
   the **fdf-adopt** skill once the five documents are filled. Offer it; do
   not start it inside this interview.
8. **Point the instruction files at the new documents.** Where `CLAUDE.md` or
   `AGENTS.md` restates what the five documents now hold — a stack list,
   build commands, conventions — propose replacing each with a one-line
   pointer, so every fact has one home from day one. fdf-checkpoint's
   *Agent instruction files* section is the full check.
9. **Log and gate.** Log the interview and its key decisions in the root
   `LOG.md` — it is bundle-wide:
   `fdf log "**Context**: the five Context documents filled by the fdf-init interview; <key decisions>."`
   Run `fdf validate` — exit 0 (F9 now satisfied) before you're done. If it
   fails on anything else, use fdf-validate.
10. **Hand off the responsibility.** Tell the user plainly (see Closing).

## What to ask

**Purpose & scope**
- What is this project, in one sentence? Who uses it and to do what?
- What are the primary use cases? What is explicitly out of scope?
- Greenfield or existing? Any hard constraints (compliance, offline, latency,
  budget, team size/skills)?

**Surfaces first → SURFACES.md**
- Which surfaces exist or will exist: HTTP APIs, human UIs (web/mobile/
  desktop), CLIs, events/webhooks, file formats / import-export, other
  inputs?
- For each: who is the audience (end user, operator, another service)?
- API: naming, versioning, error envelope, pagination, auth presentation.
- Human UI: structure, UX principles, brand, accessibility baseline, links
  to design systems or exemplars.
- CLI: command shape, flags, output formats, exit codes.
- Events / inputs: shapes, validation philosophy, operator-facing feedback.
- Assets and exemplars to point agents at; what is explicitly out of scope
  for surface work.

**Stack → STACK.md**
- Primary language(s) and runtime/version. Why these?
- Frameworks and major libraries (web, ORM, UI, testing, etc.).
- Data stores: relational / document / key-value / cache / search / blob —
  which engines, and what each holds.
- External services and APIs the project depends on.
- Version floors that matter (language edition, framework major).

**Architecture → ARCHITECTURE.md**
- Shape: modular monolith? microservices? serverless / cloud functions?
  library/SDK? CLI? A frontend, a backend, or fullstack? A mix?
- If multiple services/modules: how do they communicate (HTTP/gRPC/queue/
  events)? Where are the boundaries?
- Code organization: by layer, by feature/domain, by service? Monorepo or
  multi-repo? Where does new code go?
- Design principles and conventions to follow (e.g. DDD, hexagonal, DI,
  functional core, error-handling style, API design rules).
- State & data flow: where does state live, how does data move through the
  system?
- The few decisions a new contributor must not silently violate.

**Domain language → DOMAIN.md**
- What are the core things this system is about? Aim for the 5–15 nouns a new
  contributor must get right, not an exhaustive ontology.
- For each: what does the team call it — and what *else* has it been called
  (legacy name, vendor's name, slang, the name in the system it replaced)?
  Those others are the `instead-of` list.
- Which pairs are easy to confuse, and where is the line between them (a
  Product versus the Line Item that sells one)?
- How does each name appear in the code — type, table, field, route segment?
- Where does the wording people see differ from the term — the screen says
  "store", the code says `Venue`? Keep the label; note the mapping under
  SURFACES.md's conventions for that surface. The lexicon is for the inside.
- Any word the team has already argued about? Settle it here; that is exactly
  what this document is for.
- Start small and true. A lexicon of six real terms beats thirty invented
  ones; it grows as features arrive.

**Infrastructure → INFRA.md**
- Build & test: how is it built and tested? What commands? What CI?
- Packaging & artifacts: binaries, containers, packages, bundles?
- Environments: local, staging, production — how do they differ?
- Deployment: where does it run (cloud provider, k8s, serverless, on-prem,
  app stores, package registries)? How does a change reach production?
- Operational dependencies: managed DBs, caches, queues, CDNs, secrets,
  observability. What has to exist for the system to run?
- Targets: OS/arch, browsers, mobile platforms, runtime versions.

## Writing DOMAIN.md

The `# Terms` section is **parsed** (rule F12), so its shape is not free-form:

```markdown
# Terms

## Venue
A physical location where a merchant sells. Owns its own inventory and staff.
- instead-of: store, business, tenant, shop
- code: `Venue` (model), `venues` (table), `venue_id` (foreign key)
- see: /practices/multi-venue-scoping
```

One `## <Term>` per canonical term; the first line under it is the definition;
`instead-of` lists the banned words. F12 rejects a duplicate term, a term with
no definition, a banned word that is itself a term, and a word claimed by two
terms. A banned word is reported wherever the bundle chooses its words — every
document except `SPEC.md`, `DOMAIN.md`, `slug.test.md` and `slug.surface.md`,
and every group, slug and task name — as a warning; `strict: true` in
`DOMAIN.md`'s frontmatter makes them errors in every validation, worth
proposing once the bundle is clean.

When a banned word has an ordinary second sense in this project — *store* in
"data store", *branch* in "git branch" — list the qualified phrase under
`- except:` so it is never reported:

```markdown
## Venue
A physical location where a merchant sells.
- instead-of: store, shop, branch
- except: data store, git branch
```

An `except:` entry is always a phrase, never the banned word alone (F12 rejects
that: it would un-ban the word). Before banning a word, ask how often the
project uses it in another sense; `fdf lexicon` shows every occurrence once the
term is in place.

A bundle that already uses a word you ban — any bundle migrated from an
earlier version — gets swept in the same approved edit: a **lexicon fix** in
every document that uses the word for the concept, frozen specs, plans, tasks,
changes and logs included. A lexicon fix changes words, never what was
delivered, so it needs no Change; fdf-validate (F12) says how to do it without
breaking the scenario-name joins.

Depth does not belong here: a term needing more than a definition gets a
practice, and the term links to it with `see:`.

**Scope.** The lexicon governs the project's internal language: every
document in the bundle and the identifiers in the code. It does not govern
what a person reads on a surface — UI labels, locale and translation files,
help text, notifications, marketing copy — nor the names an external system
fixes at its own boundary. Those may use a banned word on purpose, and the
mapping ("Venue is shown to merchants as *Store*") belongs in SURFACES.md,
not in `# Terms`. Say this to the user when you present the document, so
nobody later "fixes" the UI to match the lexicon.

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
- **The first few lines of every handler, job, or command.** What do they all
  do before doing their own work? Auth? Tenant scoping? Transaction opening?
  Input validation? That is a practice.
- **Test setup.** What every test has to arrange reveals what every code path
  assumes.
- **Prose that already exists** — `CONTRIBUTING.md`, ADRs, a wiki, the long
  comment at the top of a core file, and the review comments in git history.
  Someone already wrote some of this; do not make them write it twice.
- **Where the codebase disagrees with itself** — two ways of doing one thing.
  This is the most valuable practice to write and the one you must not decide
  alone: present both, with counts and file paths, and ask which is canonical.

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
"eighteen do, twelve don't", and that used to be a dilemma: write the practice
and it overstates reality, or stay silent and the rule goes unwritten.

It is not a dilemma. The **practice** states the rule the project follows; a
**debt** names the files that have not caught up. Both are true, and neither
has to lie for the other to exist:

```markdown
---
type: Debt
status: open
title: Legacy handlers read roles directly
resource: [internal/http/admin.go, internal/http/reports.go]
timestamp: 2026-09-16T00:00:00Z
---

# Gap

Twelve handlers in `internal/http` read `user.Role` directly instead of
calling `authz.Can`. They predate the permission-checks practice, which the
rest of the codebase follows.

# Cost

Each is a place a permission rule can be changed in one spot and missed in
another.
```

What you find while reading the code that is not a gap but a **defect** —
the software doing something wrong that someone could observe — is a bug, not
a debt: `fdf bug`, with the evidence under `# Symptom` and what should happen
under `# Expected`. Do not fix it inside this interview.

`fdf debt [<group>/]<slug>` scaffolds it. `# Gap` is required and must be
concrete enough for someone else to confirm — name files and counts, not
impressions (F13). `resource` names the paths carrying the gap and is the whole
link: the practice it violates is derived by intersecting those paths with the
practice's `applies-to`, so never write a pointer to the practice by hand.

This is the moment brownfield debt is cheapest to capture — you are already
reading the code and already have the counts. Do not turn it into an audit: a
debt per practice that is not universal, not a debt per file.

**Restraint.** Practices are not gated by any rule — a bundle with none is
perfectly valid, so there is no quota to fill.

- Write the **3–7** that a contributor must not get wrong, even on a mature
  codebase. A library of twenty is a library nobody reads.
- Every practice must be backed by evidence you can point at: the files where
  it already happens. If you cannot cite them, you are inventing it — ask
  instead.
- If the user wants a *new* way of doing something, that is work on the code,
  not a practice document claiming it is already how things are. File it as a
  debt if it is worth tracking, and move on.

**Writing one.** `fdf practice [<group>/]<slug>` scaffolds it; fill it in:

```markdown
---
type: Practice
status: active
title: Permission checks
description: Where authorization decisions are made, and how they are expressed.
applies-to: [internal/authz, internal/http]
timestamp: 2026-09-16T00:00:00Z
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

# Exceptions

- `internal/http/health.go` — unauthenticated by design.
```

`# Rules` is required and non-empty (F11) — imperative, short, what code MUST
do; everything else explains it. `applies-to` lists the repo paths it governs
and those paths must exist (R1); it is how later work is routed to the
practice, since a feature never lists the practices it follows. A practice
carries no Gherkin, no spec, no plan, no tasks. Get each one approved before
writing it, exactly like a Context document, and link it from
`practices/INDEX.md`.

## Closing (say this explicitly)

Tell the user, in your own words:

> STACK.md, ARCHITECTURE.md, SURFACES.md, INFRA.md, and DOMAIN.md are written
> and validated. Treat them as **critical, living documents**: from here on I
> will not change them without your explicit approval, and after each feature
> I'll ask whether any of them needs updating (a new dependency, a new
> pattern, a new surface convention, new infrastructure, a new term) and only
> edit on your say-so, logging the change. Work that no feature records — a
> dependency upgrade, a CI move — can still make them stale, so run the
> fdf-checkpoint skill now and then, and before each release: it re-checks
> them, `SPEC.md` and CLAUDE.md/AGENTS.md against the code and each other.
> The practices under `practices/` are the same: binding on all code that
> matches their `applies-to`, and mine to propose but yours to approve. The
> code that already exists has no feature documents yet; the fdf-adopt skill
> maps it — capability by capability, breadth first — whenever you are ready.
> Keeping all of this accurate is what makes this agentic engineering rather
> than vibe coding — stale context produces confidently wrong work. They're
> yours to own; I'll help maintain them.

## Rules

- One question at a time; confirm each document before writing it.
- Write only what's true now — snapshots, not aspirations or history.
- Removing the stub sentinel/banner is required; a Context doc that still
  contains `<!-- fdf:stub -->` counts as unfilled and fails F9 once features
  exist.
- After this interview, these five files change only with explicit user
  approval, always logged. Whichever skill finds the need proposes the edit —
  most often the review that ends fdf-execute and fdf-change, or
  fdf-checkpoint. Never edit them casually mid-implementation.
- A practice records what the project **already does**, with files you can
  cite. Never write one from what would be good practice in general; that is
  how a bundle starts lying on day one.
- Where a practice is followed almost everywhere, the practice states the rule
  and a debt names the holdouts. Never soften a rule to match the worst code
  in the tree.
- Practices are never mandatory. Writing none is a valid outcome and a better
  one than writing fiction.
